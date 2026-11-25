// Package paths locates the directories Fermata writes: its settings, its
// background images and its logs.
package paths

import (
	"os"
	"path/filepath"

	"github.com/egoist/mygo"
	"github.com/gythialy/fermata/internal/settings"
)

// UserData is the app's per-user data directory.
func UserData() (string, error) { return mygo.App.Path(mygo.PathUserData) }

// Config is the settings file.
func Config() (string, error) {
	dir, err := UserData()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, settings.FileName), nil
}

// Backgrounds is the directory the break window's background images come
// from; it is created when it is needed.
func Backgrounds() (string, error) {
	dir, err := UserData()
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, "backgrounds")
	if err := os.MkdirAll(path, 0o755); err != nil {
		return "", err
	}
	return path, nil
}

// Logs is the directory the log file lives in.
func Logs() (string, error) { return mygo.App.Path(mygo.PathLogs) }
