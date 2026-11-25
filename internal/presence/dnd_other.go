//go:build !darwin

package presence

import "errors"

// FocusReadable reports that this system has no Focus the app can read.
func FocusReadable() bool { return false }

// ProbeDND reports that this system has no Focus API the app can read, so the
// app hides the setting.
func ProbeDND() (bool, error) {
	return false, errors.New("Focus or Do Not Disturb cannot be read on this system")
}
