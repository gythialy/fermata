// Package savedstate keeps the next break of a waiting app, so that a quick
// restart continues the countdown instead of starting it over.
package savedstate

import (
	"encoding/json/v2"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// Version is the format of the file.
const Version = 1

// MaxAge is how old a saved schedule may be to resume it.
const MaxAge = 10 * time.Minute

// State is the saved schedule.
type State struct {
	Version  int       `json:"version"`
	SavedAt  time.Time `json:"savedAt"`
	N        int       `json:"n"`
	BreakAt  time.Time `json:"breakAt"`
	NotifyAt time.Time `json:"notifyAt,omitzero"`
}

// Load reads a schedule, and reports whether it is still worth resuming:
// saved recently enough, for a break still ahead, in this format.
func Load(path string, now time.Time) (State, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return State{}, false
	}
	var st State
	if err := json.Unmarshal(data, &st); err != nil {
		return State{}, false
	}
	switch {
	case st.Version != Version:
		return State{}, false
	case st.N < 1:
		return State{}, false
	case st.BreakAt.IsZero() || !st.BreakAt.After(now):
		return State{}, false
	case st.SavedAt.IsZero() || st.SavedAt.After(now) || now.Sub(st.SavedAt) > MaxAge:
		return State{}, false
	case !st.NotifyAt.IsZero() && !st.NotifyAt.Before(st.BreakAt):
		return State{}, false
	}
	return st, true
}

// Save writes the state atomically.
func Save(path string, st State) error {
	st.Version = Version
	data, err := json.Marshal(st)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".schedule-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		tmp.Close()
		os.Remove(name)
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		os.Remove(name)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(name)
		return err
	}
	if err := os.Rename(name, path); err != nil {
		os.Remove(name)
		return err
	}
	return nil
}

// Clear removes the state, as when a break starts or breaks are paused.
func Clear(path string) error {
	err := os.Remove(path)
	if err == nil || errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}
