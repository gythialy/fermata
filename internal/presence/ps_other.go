//go:build !unix

package presence

import "errors"

// RunningProcesses reports that this system's process list is not available
// to the app, so exclusion rules never match.
func RunningProcesses([]string) (bool, error) {
	return false, errors.New("listing processes is not supported on this system")
}

// ProcessesReadable reports whether the app can list running processes: this
// system does not expose one to the app.
func ProcessesReadable() bool { return false }
