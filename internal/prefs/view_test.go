package prefs

import (
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/gythialy/fermata/internal/settings"
)

type fake struct {
	settings *settings.Settings
	touched  int
	resets   int
}

func newWindow(t *testing.T) (*Window, *fake) {
	t.Helper()
	f := &fake{settings: func() *settings.Settings { s := settings.Default(); s.FirstRun = false; return &s }()}
	w := New(Options{
		Settings:          func() *settings.Settings { return f.settings },
		Touched:           func() { f.touched++ },
		Reset:             func() { f.resets++ },
		PreviewSound:      func(string) {},
		Dark:              func() bool { return false },
		Clock:             func(time.Time) string { return "14:32" },
		FocusAvailable:    func() bool { return true },
		ProcessesReadable: func() bool { return true },
		Version:           func() (string, string) { return "0.1.0", "1" },
		Update:            func() (string, string) { return "", "" },
		Website:           "https://example.com",
		Source:            "https://example.com",
		Issues:            "https://example.com",
		PreviewText:       func() string { return "Look out a window until your eyes relax." },
	})
	return w, f
}

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

func TestGeneralPage(t *testing.T) {
	w, f := newWindow(t)
	tt := ui.NewTester(w.view, 800, 720)
	tt.Frame()
	want := []string{
		"General", "Start at login", "Language",
		"Pause when away", "Wait while typing",
		"Show on all displays", "Fill the screen", "App exclusions",
	}
	// The tray offers no title on Windows, and the row says so.
	if runtime.GOOS == "windows" {
		want = append(want, "Tray icon")
	} else {
		want = append(want, "Menu bar")
	}
	if runtime.GOOS == "darwin" {
		// Only macOS reads the foreground window's bounds.
		want = append(want, "Wait for fullscreen apps")
	}
	for _, text := range want {
		if !tt.HasText(text) {
			t.Errorf("missing %q in %v", text, tt.Texts())
		}
	}
	_ = f
}

// TestEverySwitchIsNamed: assistive technology announces a switch by its
// name; unnamed, it reads "switch" with no subject, and automation has
// nothing to address but an ID that changes with the frame.
func TestEverySwitchIsNamed(t *testing.T) {
	for _, section := range []string{SectionGeneral, SectionSchedule, SectionAppearance, SectionAbout} {
		w, _ := newWindow(t)
		w.section = section
		tt := ui.NewTester(w.view, 800, 720)
		tt.Frame()
		for _, a := range tt.AccessAll() {
			if a.Role == ui.RoleSwitch && a.Name == "" {
				t.Errorf("%s: a switch at %v has no name", section, a.Bounds)
			}
		}
	}
}

// TestExclusionsHiddenWhenProcessesUnreadable: a rule that can never match is
// worse than no rule, so the section goes away where the app cannot list
// processes.
func TestExclusionsHiddenWhenProcessesUnreadable(t *testing.T) {
	w, _ := newWindow(t)
	w.opts.ProcessesReadable = func() bool { return false }
	tt := ui.NewTester(w.view, 800, 720)
	tt.Frame()
	if tt.HasText("App exclusions") {
		t.Errorf("exclusions are offered although processes cannot be listed: %v", tt.Texts())
	}
}

func TestFocusRowIsHiddenWhenUnreadable(t *testing.T) {
	w, _ := newWindow(t)
	w.opts.FocusAvailable = func() bool { return false }
	tt := ui.NewTester(w.view, 800, 720)
	tt.Frame()
	if tt.HasText("Pause during Focus") {
		t.Errorf("Focus is offered although it cannot be read: %v", tt.Texts())
	}
}

func TestSchedulePage(t *testing.T) {
	w, f := newWindow(t)
	tt := ui.NewTester(w.view, 800, 720)
	tt.Frame()
	click(t, tt, "Schedule")
	for _, text := range []string{
		"Mini breaks", "Long breaks", "Break for", "Every",
		"Notify before", "Allow postpone", "Wait for me to finish",
		"Strict mode", "Break health",
	} {
		if !tt.HasText(text) {
			t.Errorf("missing %q in %v", text, tt.Texts())
		}
	}
	if !tt.HasText("min") {
		t.Errorf("the time fields do not show their unit: %v", tt.Texts())
	}
	_ = f
}

func TestAppearancePage(t *testing.T) {
	w, _ := newWindow(t)
	tt := ui.NewTester(w.view, 800, 720)
	tt.Frame()
	click(t, tt, "Appearance")
	for _, text := range []string{
		"BREAK FIELD", "Pine", "Ink", "Oxblood", "Slate", "Sand", "Paper",
		"Custom…", "SOUNDS", "Start", "End", "Volume", "Mute all sounds",
	} {
		if !tt.HasText(text) {
			t.Errorf("missing %q in %v", text, tt.Texts())
		}
	}
	if !tt.HasText("Look out a window until your eyes relax.") {
		t.Errorf("the preview is missing: %v", tt.Texts())
	}
}

func TestAboutPage(t *testing.T) {
	w, f := newWindow(t)
	tt := ui.NewTester(w.view, 800, 720)
	tt.Frame()
	click(t, tt, "About")
	for _, text := range []string{
		"Fermata", "A held note in the workday.", "Version 0.1.0 (build 1)",
		"Inspired by Stretchly. Fermata is MIT licensed.", "Website", "Source code",
		"Report an issue", "Restore Defaults…",
	} {
		if !tt.HasText(text) {
			t.Errorf("missing %q in %v", text, tt.Texts())
		}
	}
	_ = f
}

func TestPagesFitTheSmallestWindow(t *testing.T) {
	w, _ := newWindow(t)
	tt := ui.NewTester(w.view, 760, 560)
	tt.Frame()
	for _, page := range []string{"General", "Schedule", "Appearance", "About"} {
		click(t, tt, page)
		tt.Frame()
		if tt.FramePending() {
			t.Errorf("%s keeps asking for frames", page)
		}
	}
}

func TestMinimizeColumns(t *testing.T) {
	// The page must not clip its title even at the minimum size.
	w, _ := newWindow(t)
	tt := ui.NewTester(w.view, 760, 560)
	tt.Frame()
	box, ok := tt.Find("General")
	if !ok {
		t.Fatal("the title is missing")
	}
	if box.Y+box.H > 560 {
		t.Errorf("the title is below the window: %+v", box)
	}
}

// TestScheduleSteppersAlign guards the long break's "Every N mini breaks"
// row: its stepper must end at the same right edge as the row above, while
// the words "mini breaks" sit under the label.
func TestScheduleSteppersAlign(t *testing.T) {
	w, _ := newWindow(t)
	w.section = SectionSchedule
	tt := ui.NewTester(w.view, 800, 720)
	tt.Frame()
	var right []float32
	for _, a := range tt.AccessAll() {
		if a.Role == ui.RoleButton && a.Description == "Increase" {
			right = append(right, a.Bounds.X+a.Bounds.W)
		}

	}
	if len(right) != 4 {
		t.Fatalf("found %d steppers, want 4: %v", len(right), right)
	}
	sort.Slice(right, func(i, j int) bool { return right[i] < right[j] })
	for i := 0; i < len(right); i += 2 {
		if d := right[i+1] - right[i]; d > 0.5 {
			t.Errorf("stepper right edges differ by %.1f pt: %v", d, right)
		}
	}
}

func TestStrictModeExplainsWhyPostponeIsOff(t *testing.T) {
	w, f := newWindow(t)
	f.settings.Schedule.Mini.Strict = true
	tt := ui.NewTester(w.view, 800, 720)
	tt.Frame()
	click(t, tt, "Schedule")
	if text := strings.Join(tt.Texts(), "\n"); !strings.Contains(text, "Strict mode cannot postpone.") {
		t.Errorf("the postpone row does not explain strict mode: %v", tt.Texts())
	}
}
