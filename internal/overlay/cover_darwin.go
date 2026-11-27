//go:build darwin

package overlay

import (
	"fmt"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/egoist/mygo"
)

// macOS draws the menu bar at window level 24 and the Dock at 20, and AppKit
// keeps a window with a title bar below the menu bar. mygo creates a panel
// with a title bar at the floating level (3), which suits a launcher but not a
// field that fills the screen: the window arrives below the menu bar, so it
// has to be made borderless, raised above both, and placed again.
const (
	// statusLevel is NSStatusWindowLevel: over the menu bar and the Dock,
	// under menus (101) and alerts, so the system's own menus still show.
	statusLevel = 25
	// styleNonactivatingPanel is NSWindowStyleMaskNonactivatingPanel, alone:
	// no title bar, so AppKit lets the frame reach the menu bar, and the
	// panel still does not activate the app when it shows.
	styleNonactivatingPanel = 1 << 7
)

// nsRect is CGRect on a 64-bit platform.
type nsRect struct{ X, Y, W, H float64 }

var objc struct {
	once sync.Once
	err  error

	msgSend  func(receiver, selector uintptr, arg uint64)
	setFrame func(receiver, selector uintptr, rect nsRect, display uint64)
	selName  func(name uintptr) uintptr

	selStyleMask uintptr
	selLevel     uintptr
	selSetFrame  uintptr
}

// cover makes win borderless, raises it over the menu bar and the Dock, and
// places it at r again.
func cover(win *mygo.Window, r mygo.Rectangle) error {
	if err := load(); err != nil {
		return err
	}
	var err error
	// Native windows belong to the main thread, where mygo shows them.
	mygo.RunOnMain(func() {
		handle := win.NativeHandle()
		if handle == 0 {
			err = fmt.Errorf("window has no AppKit handle")
			return
		}
		height := screenHeight()
		if height == 0 {
			err = fmt.Errorf("no display to place the window on")
			return
		}
		objc.msgSend(handle, objc.selStyleMask, styleNonactivatingPanel)
		objc.msgSend(handle, objc.selLevel, statusLevel)
		// The frame AppKit left the window at is not the one it was given: a
		// borderless one keeps what it is given, in AppKit's coordinates.
		objc.setFrame(handle, objc.selSetFrame, nsRect{
			X: float64(r.X),
			Y: float64(height - r.Y - r.Height),
			W: float64(r.Width),
			H: float64(r.Height),
		}, 1)
	})
	return err
}

// screenHeight is the height of the main display, which AppKit's frame origin
// counts up from.
func screenHeight() int {
	for _, d := range mygo.Screen.Displays() {
		if d.Primary {
			return d.Bounds.Height
		}
	}
	return 0
}

// load binds the Objective-C runtime once.
func load() (err error) {
	objc.once.Do(func() {
		defer func() {
			if r := recover(); r != nil {
				objc.err = fmt.Errorf("load libobjc: %v", r)
			}
		}()
		lib, err := purego.Dlopen("/usr/lib/libobjc.A.dylib",
			purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			objc.err = fmt.Errorf("open libobjc: %w", err)
			return
		}
		purego.RegisterLibFunc(&objc.msgSend, lib, "objc_msgSend")
		purego.RegisterLibFunc(&objc.setFrame, lib, "objc_msgSend")
		purego.RegisterLibFunc(&objc.selName, lib, "sel_registerName")
		// The selectors are read as names, so that no Objective-C source is
		// needed for three messages.
		sels := []struct {
			name string
			sel  *uintptr
		}{
			{"setStyleMask:", &objc.selStyleMask},
			{"setLevel:", &objc.selLevel},
			{"setFrame:display:", &objc.selSetFrame},
		}
		for _, s := range sels {
			if *s.sel, err = selector(s.name); err != nil {
				objc.err = err
				return
			}
		}
	})
	return objc.err
}

// selector registers one selector name with the runtime.
func selector(name string) (uintptr, error) {
	b := append([]byte(name), 0)
	sel := objc.selName(uintptr(unsafe.Pointer(&b[0])))
	if sel == 0 {
		return 0, fmt.Errorf("sel_registerName(%q)", name)
	}
	return sel, nil
}
