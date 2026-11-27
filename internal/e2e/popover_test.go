//go:build e2e

package e2e

import (
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/gythialy/fermata/internal/automation"
	"github.com/gythialy/fermata/locales"
)

// bin runs the app's command line, which opens the popover: the tray icon
// that would cannot be clicked over the protocol.
func bin(t *testing.T, args ...string) {
	t.Helper()
	path := os.Getenv("FERMATA_E2E_BIN")
	if path == "" {
		t.Skip("FERMATA_E2E_BIN is not set: no command line to open the popover with")
	}
	cmd := exec.Command(path, args...)
	cmd.Env = os.Environ()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s %v: %v\n%s", path, args, err, out)
	}
}

// popover returns the popover's window, whose content is the width the
// design fixes, once it shows. The window itself is wider by its arrow.
func popover(t *testing.T, c *automation.Client) automation.Window {
	t.Helper()
	deadline := time.Now().Add(settle)
	var last []automation.Window
	for time.Now().Before(deadline) {
		windows, err := c.Windows()
		if err != nil {
			t.Fatalf("list windows: %v", err)
		}
		last = windows
		for _, w := range windows {
			if w.Visible && w.Contents.Width == 320 {
				return w
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("no popover among %v", last)
	return automation.Window{}
}

// TestPopover opens the popover from the command line, reads it and closes it
// with its close button. The tray click that opens it in use is out of the
// protocol's reach, and so is the click outside it that dismisses it.
func TestPopover(t *testing.T) {
	c := dial(t)
	bin(t, "popover", "show")
	win := popover(t, c)

	if n, err := c.Capture(win.ID, t.TempDir()+"/popover.png"); err != nil {
		t.Errorf("capture the popover: %v", err)
	} else if n == 0 {
		t.Error("the popover captured no pixels")
	}

	// The header's close button, named as the catalog names it.
	press(t, c, win.ID, locales.ActionClose())

	deadline := time.Now().Add(settle)
	for time.Now().Before(deadline) {
		windows, err := c.Windows()
		if err != nil {
			t.Fatalf("list windows: %v", err)
		}
		found := false
		for _, w := range windows {
			if w.ID == win.ID && w.Visible {
				found = true
			}
		}
		if !found {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Error("the popover stayed open after its close button")
}
