// Package overlay lifts a break window above the desktop's own bars, so that
// a field the size of the screen reaches its edges.
package overlay

import "github.com/egoist/mygo"

// Cover puts win over the menu bar and the Dock of its display, at r: the
// window's frame in screen coordinates, from the top left of the main display,
// where mygo places windows. The system keeps a window with a title bar below
// the menu bar and every window below the Dock, so a break window the size of
// the display would otherwise leave both showing. It is called once the
// window shows, and does nothing where the system already draws a screen-sized
// window over its own bars.
func Cover(win *mygo.Window, r mygo.Rectangle) error { return cover(win, r) }
