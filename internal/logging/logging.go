// Package logging writes the app's log to a file that rotates at one megabyte,
// keeping three files.
package logging

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

// FileName is the current log file's name.
const FileName = "fermata.log"

// maxSize is where a log file rotates.
const maxSize = 1 << 20

// keep is how many files one log may fill, the current one included.
const keep = 3

// Setup writes the log below dir and installs it as the default logger. When
// the file cannot be opened, the logger writes to standard error instead.
func Setup(dir string, debug bool) (*slog.Logger, io.Closer) {
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}
	w := &rotator{dir: dir, name: FileName}
	var out io.Writer = w
	if err := os.MkdirAll(dir, 0o755); err != nil {
		out = os.Stderr
	}
	logger := slog.New(slog.NewTextHandler(out, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)
	return logger, w
}

// rotator is a writer that starts a new file before one grows past maxSize.
type rotator struct {
	mu   sync.Mutex
	dir  string
	name string
	file *os.File
	size int64
}

// Write appends p, rotating first when the file would grow too large.
func (r *rotator) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.file == nil {
		if err := r.open(); err != nil {
			return 0, err
		}
	}
	if r.size+int64(len(p)) > maxSize {
		r.rotate()
	}
	n, err := r.file.Write(p)
	r.size += int64(n)
	return n, err
}

// Close closes the current file.
func (r *rotator) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.file == nil {
		return nil
	}
	err := r.file.Close()
	r.file = nil
	return err
}

func (r *rotator) open() error {
	path := filepath.Join(r.dir, r.name)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return err
	}
	r.file, r.size = f, info.Size()
	return nil
}

func (r *rotator) rotate() {
	path := filepath.Join(r.dir, r.name)
	if r.file != nil {
		_ = r.file.Close()
		r.file = nil
	}
	oldest := path + "." + itoa(keep-1)
	_ = os.Remove(oldest)
	for i := keep - 2; i >= 1; i-- {
		_ = os.Rename(path+"."+itoa(i), path+"."+itoa(i+1))
	}
	_ = os.Rename(path, path+".1")
	_ = r.open()
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [8]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
