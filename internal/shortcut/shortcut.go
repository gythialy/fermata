// Package shortcut names the keyboard shortcut the break window and the
// scaled previews show. The handler itself lives in the app; this is only
// the label, so no view writes a platform's own symbols by hand.
package shortcut

import "runtime"

// End is the shortcut that takes the break window's one exit.
func End() string {
	if runtime.GOOS == "darwin" {
		return "⌘X"
	}
	return "Ctrl+X"
}
