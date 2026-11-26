package welcome

import (
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/gythialy/fermata/internal/ideas"
	"github.com/gythialy/fermata/internal/settings"
)

// click presses the control whose accessible name is name.
func click(t *testing.T, tt *ui.Tester, name string) {
	t.Helper()
	for _, a := range tt.AccessAll() {
		if a.Role == ui.RoleButton && a.Name == name {
			tt.ClickAt(a.Bounds.X+a.Bounds.W/2, a.Bounds.Y+a.Bounds.H/2)
			tt.Frame()
			return
		}
	}
	t.Fatalf("%q is not a button in %v", name, tt.Texts())
}

func newWindow(t *testing.T, started *[]bool) *Window {
	t.Helper()
	s := settings.Default()
	w := New(Options{
		Settings: func() *settings.Settings { return &s },
		Touched:  func() {},
		Start:    func(openSettings bool) { *started = append(*started, openSettings) },
		Dark:     func() bool { return false },
		Clock:    func(time.Time) string { return "14:32" },
		Idea:     func() ideas.Idea { return ideas.Idea{Text: "Look out a window until your eyes relax."} },
	})
	return w
}

// TestSwitchesAreNamed: the welcome window's one switch announces the row it
// belongs to, not "switch".
func TestSwitchesAreNamed(t *testing.T) {
	started := []bool{}
	w := newWindow(t, &started)
	tt := ui.NewTester(w.view, 720, 560)
	tt.Frame()
	switches := 0
	for _, a := range tt.AccessAll() {
		if a.Role != ui.RoleSwitch {
			continue
		}
		switches++
		if a.Name == "" {
			t.Errorf("a switch at %v has no name", a.Bounds)
		}
	}
	if switches == 0 {
		t.Error("the welcome window should offer the login switch")
	}
}

func TestWelcomeShowsTheDecision(t *testing.T) {
	var started []bool
	w := newWindow(t, &started)
	tt := ui.NewTester(w.view, 800, 540)
	tt.Frame()
	for _, text := range []string{
		"Fermata", "A held note in the workday.", "Mini break", "Long break",
		"Language", "Start at login", "Start", "Open Settings",
		"Closing this window starts Fermata too.",
	} {
		if !tt.HasText(text) {
			t.Errorf("missing %q in %v", text, tt.Texts())
		}
	}
	if !tt.HasText("Look out a window until your eyes relax.") {
		t.Errorf("the preview is missing: %v", tt.Texts())
	}
	click(t, tt, "Start")
	if len(started) != 1 || started[0] {
		t.Fatalf("started %v", started)
	}
}

func TestOpenSettingsIsTheSameDecision(t *testing.T) {
	var started []bool
	w := newWindow(t, &started)
	tt := ui.NewTester(w.view, 800, 540)
	tt.Frame()
	click(t, tt, "Open Settings")
	if len(started) != 1 || !started[0] {
		t.Fatalf("started %v", started)
	}
}

func TestWelcomeFitsItsWindow(t *testing.T) {
	var started []bool
	w := newWindow(t, &started)
	tt := ui.NewTester(w.view, 800, 540)
	tt.Frame()
	box, ok := tt.Find("Fermata")
	if !ok {
		t.Fatal("the name is missing")
	}
	if box.X+box.W > 800 || box.Y+box.H > 540 {
		t.Errorf("the name is outside the window: %+v", box)
	}
}

func TestWelcomeRhythmUsesAnOrdinal(t *testing.T) {
	var started []bool
	w := newWindow(t, &started)
	tt := ui.NewTester(w.view, 800, 540)
	tt.Frame()
	if text := strings.Join(tt.Texts(), "\n"); !strings.Contains(text, "every 5th break") {
		t.Errorf("the long break's place is not an ordinal: %v", tt.Texts())
	}
}
