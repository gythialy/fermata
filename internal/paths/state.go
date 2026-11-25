package paths

import (
	"path/filepath"
)

// Schedule is the file that remembers the next break across a quick restart.
func Schedule() (string, error) {
	dir, err := UserData()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "schedule.json"), nil
}
