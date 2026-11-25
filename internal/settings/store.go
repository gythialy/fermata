package settings

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// FileName is the configuration's file name inside the user data directory.
const FileName = "config.json"

// SaveDelay is how long a change waits before it is written, so that dragging
// a slider writes once.
const SaveDelay = 150 * time.Millisecond

// knownKeys are the top-level members the struct has: everything else in the
// file is kept as it was.
var knownKeys = []string{
	"version", "general", "schedule", "presence", "breakWindow",
	"sound", "shortcuts", "health", "isFirstRun",
}

// Store is the settings in memory, with a writer that saves them atomically
// after a short delay.
//
// Get and Update, and the values bound into views, are used on the main
// thread. The writer goroutine sees only the encoded snapshots it is handed,
// never the Settings itself.
type Store struct {
	path  string
	s     Settings
	extra map[string]jsontext.Value

	writeCh chan []byte
	flushCh chan chan error
	done    chan struct{}
	wg      sync.WaitGroup

	mu       sync.Mutex
	closed   bool
	writeErr error

	changes []func()
	onErr   []func(error)
}

// Load reads the settings at path, or writes the defaults when there is no
// file yet. A file that cannot be read is backed up as
// config.corrupt.<timestamp>.json and replaced with the defaults. The returned
// warnings describe everything that was corrected.
func Load(path string) (*Store, []string, error) {
	st := &Store{
		path:    path,
		s:       Default(),
		extra:   map[string]jsontext.Value{},
		writeCh: make(chan []byte, 1),
		flushCh: make(chan chan error),
		done:    make(chan struct{}),
	}
	var warnings []string
	data, err := os.ReadFile(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		warnings = append(warnings, "no config yet; writing the defaults")
	case err != nil:
		return nil, nil, fmt.Errorf("read settings: %w", err)
	default:
		if len(data) == 0 {
			warnings = append(warnings, "config was empty; using the defaults")
			break
		}
		var all map[string]jsontext.Value
		parseErr := json.Unmarshal(data, &all)
		s := Default()
		if parseErr == nil {
			parseErr = json.Unmarshal(data, &s)
		}
		if parseErr != nil {
			if backup, rerr := backupCorrupt(path); rerr != nil {
				warnings = append(warnings, fmt.Sprintf("config was unreadable (%v); keeping it and using the defaults", parseErr))
				_ = rerr
			} else {
				warnings = append(warnings, fmt.Sprintf("config was unreadable (%v); backed up to %s", parseErr, backup))
			}
			s = Default()
			// The user configured the app before: do not show the welcome
			// window again because of a read error.
			s.FirstRun = false
		} else {
			for _, k := range knownKeys {
				delete(all, k)
			}
			st.extra = all
		}
		st.s = s
	}
	migrate(&st.s)
	warnings = append(warnings, st.s.Validate()...)
	if err := st.writeAtomic(st.encode()); err != nil {
		return nil, warnings, fmt.Errorf("write settings: %w", err)
	}
	st.wg.Go(st.run)
	return st, warnings, nil
}

// New returns a store for path with s already loaded, and its writer running.
// It is for tests and for a caller that builds settings itself.
func New(path string, s Settings) *Store {
	st := &Store{
		path:    path,
		s:       s,
		extra:   map[string]jsontext.Value{},
		writeCh: make(chan []byte, 1),
		flushCh: make(chan chan error),
		done:    make(chan struct{}),
	}
	st.wg.Go(st.run)
	return st
}

// Path returns the file the store writes.
func (st *Store) Path() string { return st.path }

// Get returns the live settings, which the main thread may read and bind to
// views. Call Touched or Update after changing them so that they are
// validated and saved.
func (st *Store) Get() *Settings { return &st.s }

// Update changes the settings, validates them and queues a save, returning
// what validation corrected.
func (st *Store) Update(fn func(*Settings)) []string {
	fn(&st.s)
	return st.Touched()
}

// Touched validates the settings after a bound control changed them, queues
// a save and tells the listeners. It returns what validation corrected.
func (st *Store) Touched() []string {
	warnings := (&st.s).Validate()
	data := st.encode()
	st.mu.Lock()
	closed := st.closed
	st.mu.Unlock()
	if data != nil && !closed {
		st.queue(data)
	}
	for _, fn := range st.changes {
		fn()
	}
	return warnings
}

// OnChanged adds a listener called after every Touched or Update.
func (st *Store) OnChanged(fn func()) { st.changes = append(st.changes, fn) }

// OnWriteError adds a listener called when saving fails. It runs on the
// writer goroutine.
func (st *Store) OnWriteError(fn func(error)) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.onErr = append(st.onErr, fn)
}

// LastWriteError returns the last save error, if any.
func (st *Store) LastWriteError() error {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.writeErr
}

// Flush writes any pending change and returns the last error.
func (st *Store) Flush() error {
	st.mu.Lock()
	closed := st.closed
	st.mu.Unlock()
	if closed {
		return st.LastWriteError()
	}
	reply := make(chan error, 1)
	select {
	case st.flushCh <- reply:
		return <-reply
	case <-st.done:
		return st.LastWriteError()
	}
}

// Close flushes and stops the writer.
func (st *Store) Close() error {
	st.mu.Lock()
	if st.closed {
		st.mu.Unlock()
		return st.writeErr
	}
	st.closed = true
	st.mu.Unlock()
	close(st.done)
	st.wg.Wait()
	return st.LastWriteError()
}

// queue hands a snapshot to the writer, replacing one not yet written.
func (st *Store) queue(data []byte) {
	select {
	case st.writeCh <- data:
	default:
		select {
		case <-st.writeCh:
		default:
		}
		select {
		case st.writeCh <- data:
		default:
		}
	}
}

// run writes the latest snapshot after SaveDelay, one write per burst.
func (st *Store) run() {
	var pending []byte
	timer := time.NewTimer(time.Hour)
	timer.Stop()
	armed := false
	for {
		select {
		case data := <-st.writeCh:
			pending = data
			timer.Reset(SaveDelay)
			armed = true
		case <-timer.C:
			armed = false
			pending = drain(st.writeCh, pending)
			st.write(pending)
			pending = nil
		case reply := <-st.flushCh:
			// Take every snapshot queued before the flush, so that it saves
			// the latest one and not the one the debounce was holding.
			pending = drain(st.writeCh, pending)
			if armed {
				timer.Stop()
				armed = false
			}
			if pending != nil {
				st.write(pending)
				pending = nil
			}
			reply <- st.LastWriteError()
		case <-st.done:
			if pending != nil {
				st.write(pending)
			}
			return
		}
	}
}

// drain returns the newest snapshot queued after pending, if any.
func drain(ch chan []byte, pending []byte) []byte {
	for {
		select {
		case data := <-ch:
			pending = data
		default:
			return pending
		}
	}
}

// write saves one snapshot and records the outcome.
func (st *Store) write(data []byte) {
	if data == nil {
		return
	}
	err := writeAtomic(st.path, data)
	st.mu.Lock()
	st.writeErr = err
	callbacks := append([]func(error){}, st.onErr...)
	st.mu.Unlock()
	if err != nil {
		for _, fn := range callbacks {
			fn(err)
		}
	}
}

// encode turns the settings, and the unknown members kept from the file, into
// the JSON the store writes.
func (st *Store) encode() []byte {
	known, err := json.Marshal(st.s)
	if err != nil {
		return nil
	}
	var m map[string]jsontext.Value
	if err := json.Unmarshal(known, &m); err != nil {
		return known
	}
	for k, v := range st.extra {
		if _, ok := m[k]; !ok {
			m[k] = v
		}
	}
	out, err := json.Marshal(m, json.Deterministic(true), jsontext.WithIndent("  "))
	if err != nil {
		return nil
	}
	return append(out, '\n')
}

// writeAtomic replaces the file with data in one step: a temporary file in
// the same directory, synced, then renamed.
func (st *Store) writeAtomic(data []byte) error { return writeAtomic(st.path, data) }

func writeAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".config-*.json.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if err := f.Chmod(0o644); err != nil {
		f.Close()
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// backupCorrupt renames an unreadable file out of the way and returns its new
// path.
func backupCorrupt(path string) (string, error) {
	dir, base := filepath.Split(path)
	stamp := time.Now().Format("20060102T150405")
	for i := range 100 {
		name := base + ".corrupt." + stamp
		if i > 0 {
			name = fmt.Sprintf("%s%d", name, i)
		}
		backup := filepath.Join(dir, name)
		if _, err := os.Stat(backup); !errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err := os.Rename(path, backup); err != nil {
			return "", err
		}
		return backup, nil
	}
	return "", errors.New("no free backup name")
}
