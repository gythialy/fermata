//go:build darwin

package fullscreen

import (
	"fmt"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

// The CoreGraphics window list options.
const (
	listOnScreenOnly        = 1 << 0
	listExcludeDesktopElems = 1 << 4
	nullWindowID            = 0
	cfNumberSInt64Type      = 4
	// kCFStringEncodingUTF8
	kCFStringEncodingUTF8 = 0x08000100
)

// cgRect is CGRect on a 64-bit platform.
type cgRect struct{ X, Y, W, H float64 }

var cg struct {
	once sync.Once
	err  error

	copyWindows  func(option uint32, relativeTo uint32) uintptr
	rectFromDict func(dict uintptr, rect *cgRect) bool

	arrayCount func(array uintptr) int64
	arrayValue func(array uintptr, index int64) uintptr
	dictGet    func(dict uintptr, key uintptr) uintptr
	numberGet  func(number uintptr, theType int64, value unsafe.Pointer) bool
	release    func(ref uintptr)
	getType    func(ref uintptr) uintptr
	numberType func() uintptr
	dictType   func() uintptr
	newString  func(alloc uintptr, cString uintptr, encoding uint32) uintptr

	keyLayer  uintptr
	keyPID    uintptr
	keyBounds uintptr
}

// Active reports whether the frontmost normal window covers a display. Self is
// the calling process, whose own windows never count.
func Active(displays []Rect, selfPID int) (bool, error) {
	if err := load(); err != nil {
		return false, err
	}
	if len(displays) == 0 {
		return false, nil
	}
	windows := cg.copyWindows(listOnScreenOnly|listExcludeDesktopElems, nullWindowID)
	if windows == 0 {
		return false, nil
	}
	defer cg.release(windows)

	n := cg.arrayCount(windows)
	if n <= 0 {
		return false, nil
	}
	list := make([]window, 0, n)
	for i := int64(0); i < n; i++ {
		dict := cg.arrayValue(windows, i)
		if dict == 0 {
			continue
		}
		layer, ok := number(dict, cg.keyLayer)
		if !ok {
			continue
		}
		pid, ok := number(dict, cg.keyPID)
		if !ok {
			continue
		}
		w := window{layer: int(layer), pid: int(pid)}
		if bounds := cg.dictGet(dict, cg.keyBounds); bounds != 0 && cg.getType(bounds) == cg.dictType() {
			var r cgRect
			if cg.rectFromDict(bounds, &r) {
				w.rect = Rect{X: r.X, Y: r.Y, W: r.W, H: r.H}
				w.hasRect = true
			}
		}
		list = append(list, w)
	}
	return fullscreenIn(list, displays, selfPID), nil
}

// number reads a CFNumber from a dictionary value, and reports whether the
// value is a number at all.
func number(dict, key uintptr) (int64, bool) {
	v := cg.dictGet(dict, key)
	if v == 0 || cg.getType(v) != cg.numberType() {
		return 0, false
	}
	var out int64
	if !cg.numberGet(v, cfNumberSInt64Type, unsafe.Pointer(&out)) {
		return 0, false
	}
	return out, true
}

// load binds CoreGraphics and CoreFoundation once.
func load() (err error) {
	cg.once.Do(func() {
		defer func() {
			if r := recover(); r != nil {
				cg.err = fmt.Errorf("load CoreGraphics: %v", r)
			}
		}()
		graphics, err := purego.Dlopen(
			"/System/Library/Frameworks/CoreGraphics.framework/CoreGraphics",
			purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			cg.err = fmt.Errorf("open CoreGraphics: %w", err)
			return
		}
		foundation, err := purego.Dlopen(
			"/System/Library/Frameworks/CoreFoundation.framework/CoreFoundation",
			purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			cg.err = fmt.Errorf("open CoreFoundation: %w", err)
			return
		}
		purego.RegisterLibFunc(&cg.copyWindows, graphics, "CGWindowListCopyWindowInfo")
		purego.RegisterLibFunc(&cg.rectFromDict, graphics, "CGRectMakeWithDictionaryRepresentation")
		purego.RegisterLibFunc(&cg.arrayCount, foundation, "CFArrayGetCount")
		purego.RegisterLibFunc(&cg.arrayValue, foundation, "CFArrayGetValueAtIndex")
		purego.RegisterLibFunc(&cg.dictGet, foundation, "CFDictionaryGetValue")
		purego.RegisterLibFunc(&cg.numberGet, foundation, "CFNumberGetValue")
		purego.RegisterLibFunc(&cg.release, foundation, "CFRelease")
		purego.RegisterLibFunc(&cg.getType, foundation, "CFGetTypeID")
		purego.RegisterLibFunc(&cg.numberType, foundation, "CFNumberGetTypeID")
		purego.RegisterLibFunc(&cg.dictType, foundation, "CFDictionaryGetTypeID")
		purego.RegisterLibFunc(&cg.newString, foundation, "CFStringCreateWithCString")
		// The window keys are CFStrings whose value is their name; making
		// them avoids reading CoreGraphics' data symbols.
		key := func(name string) (uintptr, error) {
			b := append([]byte(name), 0)
			ref := cg.newString(0, uintptr(unsafe.Pointer(&b[0])), kCFStringEncodingUTF8)
			if ref == 0 {
				return 0, fmt.Errorf("CFStringCreateWithCString(%q)", name)
			}
			return ref, nil
		}
		if cg.keyLayer, err = key("kCGWindowLayer"); err != nil {
			cg.err = err
			return
		}
		if cg.keyPID, err = key("kCGWindowOwnerPID"); err != nil {
			cg.err = err
			return
		}
		if cg.keyBounds, err = key("kCGWindowBounds"); err != nil {
			cg.err = err
			return
		}
	})
	return cg.err
}
