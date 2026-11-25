//go:build darwin

package fullscreen

import "testing"

// TestActiveSmoke loads CoreGraphics and reads the window list, which proves
// the bindings work; whether an app is fullscreen depends on the desktop.
func TestActiveSmoke(t *testing.T) {
	displays := []Rect{{X: 0, Y: 0, W: 1440, H: 900}}
	active, err := Active(displays, 0)
	if err != nil {
		t.Fatalf("Active: %v", err)
	}
	t.Logf("fullscreen active: %v", active)
}
