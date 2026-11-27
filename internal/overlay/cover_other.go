//go:build !darwin

package overlay

import "github.com/egoist/mygo"

// cover does nothing: a window the size of the screen already covers the
// taskbar or panels of the other desktops.
func cover(*mygo.Window, mygo.Rectangle) error { return nil }
