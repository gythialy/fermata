//go:build !darwin

package fullscreen

// Active reports that this system has no fullscreen watcher.
func Active([]Rect, int) (bool, error) { return false, nil }
