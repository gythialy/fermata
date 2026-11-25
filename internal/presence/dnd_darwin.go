//go:build darwin

package presence

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// FocusReadable reports whether the app can read Focus without asking the
// user for extra access. macOS 13 and later protect the assertion file, so
// this is false until the user grants Full Disk Access themselves; the app
// never asks for it.
func FocusReadable() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	dir := filepath.Join(home, "Library", "DoNotDisturb", "DB")
	path := filepath.Join(dir, "Assertions.json")
	switch _, err := os.ReadFile(path); {
	case err == nil:
		return true
	case errors.Is(err, fs.ErrNotExist):
		// No file is a readable "off" when the directory itself is readable.
		_, dirErr := os.Stat(dir)
		return dirErr == nil
	default:
		_, ok := legacyDND()
		return ok
	}
}

// ProbeDND reports whether Focus or Do Not Disturb is on. macOS 27 keeps it
// in ~/Library/DoNotDisturb/DB/Assertions.json, which needs Full Disk Access;
// older systems answer through defaults. A state that cannot be read is an
// error, which the app shows once and treats as off.
func ProbeDND() (bool, error) {
	home, err := os.UserHomeDir()
	if err == nil {
		path := filepath.Join(home, "Library", "DoNotDisturb", "DB", "Assertions.json")
		data, readErr := os.ReadFile(path)
		switch {
		case readErr == nil:
			return activeAssertion(data)
		case errors.Is(readErr, fs.ErrNotExist):
			if on, ok := legacyDND(); ok {
				return on, nil
			}
			return false, nil
		default:
			if on, ok := legacyDND(); ok {
				return on, nil
			}
			return false, fmt.Errorf("read Focus assertions: %w", readErr)
		}
	}
	if on, ok := legacyDND(); ok {
		return on, nil
	}
	return false, errors.New("cannot read Focus on this system")
}

// activeAssertion reports whether the file holds a live assertion. Its shape
// has changed between macOS versions, so the records are matched generically.
func activeAssertion(data []byte) (bool, error) {
	var doc any
	if err := json.Unmarshal(data, &doc); err != nil {
		return false, fmt.Errorf("parse Focus assertions: %w", err)
	}
	return hasRecords(doc), nil
}

func hasRecords(v any) bool {
	switch v := v.(type) {
	case map[string]any:
		for key, value := range v {
			if key == "storeAssertionRecords" {
				if records, ok := value.([]any); ok && len(records) > 0 {
					return true
				}
			}
			if hasRecords(value) {
				return true
			}
		}
	case []any:
		for _, item := range v {
			if hasRecords(item) {
				return true
			}
		}
	}
	return false
}

// legacyDND asks the older CoreFoundation preference, which macOS 13 to 15
// still write.
func legacyDND() (bool, bool) {
	out, err := exec.Command("defaults", "-currentHost", "read", "com.apple.notificationcenterui", "doNotDisturb").Output()
	if err != nil {
		return false, false
	}
	switch strings.TrimSpace(string(out)) {
	case "1", "true", "YES", "yes":
		return true, true
	case "0", "false", "NO", "no":
		return false, true
	}
	return false, false
}
