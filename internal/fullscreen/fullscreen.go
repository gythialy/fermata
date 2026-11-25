// Package fullscreen reports whether an application is fullscreen on screen.
//
// On macOS it reads window bounds, owner and layer only, which needs no
// Screen Recording permission (windows' titles or images would).
package fullscreen

import "math"

// Rect is a screen rectangle in points, from the top left of the main display.
type Rect struct{ X, Y, W, H float64 }

// tolerance is how far a window may be from a display's edge and still count,
// for rounded corners and window managers that inset a point.
const tolerance = 2.0

// minSide ignores status items, tooltips and other tiny helper windows.
const minSide = 100.0

// window is one window the platform listed, front to back.
type window struct {
	layer int
	pid   int
	rect  Rect
	// hasRect is false when the window's bounds could not be read.
	hasRect bool
}

// fullscreenIn decides whether the frontmost normal window covers a display.
// The list is ordered front to back; the first eligible window decides, so a
// fullscreen window behind a normal one does not count.
func fullscreenIn(windows []window, displays []Rect, selfPID int) bool {
	for _, w := range windows {
		if w.layer != 0 || w.pid == selfPID || !w.hasRect {
			continue
		}
		if w.rect.W < minSide || w.rect.H < minSide {
			continue
		}
		return matchesDisplay(w.rect, displays)
	}
	return false
}

func matchesDisplay(r Rect, displays []Rect) bool {
	for _, d := range displays {
		if math.Abs(r.X-d.X) <= tolerance &&
			math.Abs(r.Y-d.Y) <= tolerance &&
			math.Abs(r.W-d.W) <= tolerance &&
			math.Abs(r.H-d.H) <= tolerance {
			return true
		}
	}
	return false
}
