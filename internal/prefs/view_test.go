package prefs

import (
	"runtime"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/i18n"
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
		Now:               func() time.Time { return time.Date(2025, 6, 2, 10, 7, 0, 0, time.Local) },
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

// namedAccess finds one control by its accessible role and name.
func namedAccess(t *testing.T, tt *ui.Tester, role ui.Role, name string) ui.AccessInfo {
	t.Helper()
	a, ok := tt.AccessByRole(role, name)
	if !ok {
		t.Fatalf("%v %q is missing from %v", role, name, tt.Texts())
	}
	return a
}

// press clicks the center of a control and settles the frame it rebuilds.
func press(t *testing.T, tt *ui.Tester, a ui.AccessInfo) {
	t.Helper()
	tt.ClickAt(a.Bounds.X+a.Bounds.W/2, a.Bounds.Y+a.Bounds.H/2)
	tt.Frame()
}

// useLocale switches both the catalogs and the tester's text shaping.
func useLocale(t *testing.T, tt *ui.Tester, tag string) {
	t.Helper()
	i18n.SetLocaleFunc(func() string { return tag })
	t.Cleanup(func() { i18n.SetLocaleFunc(nil) })
	tt.SetLocale(tag)
}

func containsText(tt *ui.Tester, parts ...string) bool {
	text := strings.Join(tt.Texts(), "\n")
	for _, part := range parts {
		if !strings.Contains(text, part) {
			return false
		}
	}
	return true
}

// TestSkipPeriodsDisabledWhenMasterOff keeps the saved data but takes every
// period control away while the master switch is off.
func TestSkipPeriodsDisabledWhenMasterOff(t *testing.T) {
	w, f := newWindow(t)
	w.section = SectionSchedule
	f.settings.Schedule.SkipPeriods = []settings.SkipPeriod{{Enabled: true, Start: "09:00", End: "10:00"}}
	tt := ui.NewTester(w.view, 800, 720)
	tt.Frame()
	if !containsText(tt, "Skip periods", "Enable", "+ Add period",
		"Skip periods are off", "Your settings are kept and apply when enabled") {
		t.Errorf("the off state is not explained: %v", tt.Texts())
	}
	for _, ctl := range []struct {
		role ui.Role
		name string
	}{
		{ui.RoleButton, "+ Add period"},
		{ui.RoleTextField, "Start time"},
		{ui.RoleTextField, "End time"},
		{ui.RoleToggleButton, "Mon"},
		{ui.RoleToggleButton, "Sun"},
		{ui.RoleSwitch, "Enable 09:00 to 10:00 period"},
		{ui.RoleButton, "Remove"},
	} {
		if a := namedAccess(t, tt, ctl.role, ctl.name); !a.Disabled {
			t.Errorf("%v %q is enabled while skip periods are off", ctl.role, ctl.name)
		}
	}
	master := namedAccess(t, tt, ui.RoleSwitch, "Enable skip periods")
	if master.Disabled || master.Checked {
		t.Errorf("the master switch should be on and enabled: %+v", master)
	}
	press(t, tt, master)
	if !f.settings.Schedule.SkipPeriodsEnabled {
		t.Error("the master switch did not turn skip periods on")
	}
	if len(f.settings.Schedule.SkipPeriods) != 1 {
		t.Errorf("the saved period was lost: %+v", f.settings.Schedule.SkipPeriods)
	}
	if a := namedAccess(t, tt, ui.RoleButton, "+ Add period"); a.Disabled {
		t.Error("Add stays disabled after enabling skip periods")
	}
}

// TestSkipPeriodsEmptyAndLimit covers the empty state and the four-period cap.
func TestSkipPeriodsEmptyAndLimit(t *testing.T) {
	w, f := newWindow(t)
	w.section = SectionSchedule
	f.settings.Schedule.SkipPeriodsEnabled = true
	tt := ui.NewTester(w.view, 800, 720)
	tt.Frame()
	if !containsText(tt, "No enabled skip periods", "Add a period to turn this on") {
		t.Errorf("the empty state is missing: %v", tt.Texts())
	}
	add := namedAccess(t, tt, ui.RoleButton, "+ Add period")
	if add.Disabled {
		t.Fatal("Add is disabled with no periods")
	}
	press(t, tt, add)
	if len(f.settings.Schedule.SkipPeriods) != 1 {
		t.Fatalf("Add did not append a period: %+v", f.settings.Schedule.SkipPeriods)
	}
	if f.touched == 0 {
		t.Error("Add did not touch the settings")
	}

	for i := 1; i < maxSkipPeriods; i++ {
		f.settings.Schedule.SkipPeriods = append(f.settings.Schedule.SkipPeriods,
			settings.SkipPeriod{Enabled: true, Start: "10:00", End: "11:00"})
	}
	tt.Frame()
	if a := namedAccess(t, tt, ui.RoleButton, "+ Add period"); !a.Disabled {
		t.Error("Add is still enabled at the four-period limit")
	}
	if got := len(tt.AccessAll()); got < 4 {
		t.Errorf("the fourth period is missing: %d controls", got)
	}
}

// TestSkipPeriodsActiveStatus uses skiptime's active window for its copy.
func TestSkipPeriodsActiveStatus(t *testing.T) {
	w, f := newWindow(t)
	w.section = SectionSchedule
	f.settings.Schedule.SkipPeriodsEnabled = true
	f.settings.Schedule.SkipPeriods = []settings.SkipPeriod{{Enabled: true, Start: "09:00", End: "10:00"}}
	w.opts.Now = func() time.Time { return time.Date(2025, 6, 2, 9, 30, 0, 0, time.Local) }
	tt := ui.NewTester(w.view, 800, 720)
	tt.Frame()
	if !containsText(tt, "Skipping now · until 10:00", "Breaks restart afterwards") {
		t.Errorf("the active status is missing: %v", tt.Texts())
	}
}

// TestSkipPeriodsNextStatus uses skiptime's next boundary for both times.
func TestSkipPeriodsNextStatus(t *testing.T) {
	w, f := newWindow(t)
	w.section = SectionSchedule
	f.settings.Schedule.SkipPeriodsEnabled = true
	f.settings.Schedule.SkipPeriods = []settings.SkipPeriod{
		{Enabled: true, Start: "12:00", End: "13:00"},
		{Enabled: true, Start: "15:00", End: "16:00"},
	}
	tt := ui.NewTester(w.view, 800, 720)
	tt.Frame()
	if !containsText(tt, "Next skip 12:00", "Until 13:00") {
		t.Errorf("the next status is missing: %v", tt.Texts())
	}
}

// TestSkipPeriodsWeekdayToggle checks the dots and the stored day numbers.
func TestSkipPeriodsWeekdayToggle(t *testing.T) {
	w, f := newWindow(t)
	w.section = SectionSchedule
	f.settings.Schedule.SkipPeriodsEnabled = true
	f.settings.Schedule.SkipPeriods = []settings.SkipPeriod{
		{Enabled: true, Start: "09:00", End: "10:00", Days: []int{1, 3}},
	}
	tt := ui.NewTester(w.view, 800, 720)
	tt.Frame()
	for _, day := range []struct {
		name    string
		checked bool
	}{
		{"Mon", true}, {"Tue", false}, {"Wed", true}, {"Thu", false},
		{"Fri", false}, {"Sat", false}, {"Sun", false},
	} {
		if a := namedAccess(t, tt, ui.RoleToggleButton, day.name); a.Checked != day.checked {
			t.Errorf("%s: checked %v, want %v", day.name, a.Checked, day.checked)
		}
	}
	press(t, tt, namedAccess(t, tt, ui.RoleToggleButton, "Tue"))
	days := f.settings.Schedule.SkipPeriods[0].Days
	if !slices.Equal(days, []int{1, 2, 3}) {
		t.Errorf("days after adding Tuesday: %v", days)
	}
	if a := namedAccess(t, tt, ui.RoleToggleButton, "Tue"); !a.Checked {
		t.Error("Tuesday did not stay selected")
	}
	press(t, tt, namedAccess(t, tt, ui.RoleToggleButton, "Mon"))
	if days := f.settings.Schedule.SkipPeriods[0].Days; !slices.Equal(days, []int{2, 3}) {
		t.Errorf("days after removing Monday: %v", days)
	}
}

// TestSkipPeriodsOvernightCopy says which day each end belongs to.
func TestSkipPeriodsOvernightCopy(t *testing.T) {
	w, f := newWindow(t)
	w.section = SectionSchedule
	f.settings.Schedule.SkipPeriodsEnabled = true
	f.settings.Schedule.SkipPeriods = []settings.SkipPeriod{
		{Enabled: true, Start: "22:00", End: "08:00", Days: []int{1, 2, 3, 4, 5}},
	}
	tt := ui.NewTester(w.view, 800, 720)
	tt.Frame()
	if !containsText(tt, "From 22:00 on the start day · until 08:00 the next day") {
		t.Errorf("the overnight span is not explained: %v", tt.Texts())
	}
	if strings.Contains(strings.Join(tt.Texts(), "\n"), "workday") {
		t.Errorf("the day label should not say workday: %v", tt.Texts())
	}
	useLocale(t, tt, "zh-Hans")
	tt.Frame()
	if !containsText(tt, "按开始日 22:00 起 · 次日 08:00 止") {
		t.Errorf("the Chinese overnight span is wrong: %v", tt.Texts())
	}
}

// TestSkipPeriodsChineseStatusCopy pins the four states in the app's Chinese.
func TestSkipPeriodsChineseStatusCopy(t *testing.T) {
	period := settings.SkipPeriod{Enabled: true, Start: "09:00", End: "10:00"}
	for _, tc := range []struct {
		name    string
		on      bool
		periods []settings.SkipPeriod
		now     time.Time
		want    []string
	}{
		{
			name: "off", periods: []settings.SkipPeriod{period},
			now:  time.Date(2025, 6, 2, 9, 30, 0, 0, time.Local),
			want: []string{"跳过时段已关闭", "设置已保留，开启后生效"},
		},
		{
			name: "on but empty", on: true,
			now:  time.Date(2025, 6, 2, 9, 30, 0, 0, time.Local),
			want: []string{"暂无启用的跳过时段", "添加时段后生效"},
		},
		{
			name: "active", on: true, periods: []settings.SkipPeriod{period},
			now:  time.Date(2025, 6, 2, 9, 30, 0, 0, time.Local),
			want: []string{"正在跳过 · 至 10:00", "结束后重新计时"},
		},
		{
			name: "next", on: true,
			periods: []settings.SkipPeriod{{Enabled: true, Start: "12:00", End: "13:00"}},
			now:     time.Date(2025, 6, 2, 10, 7, 0, 0, time.Local),
			want:    []string{"下次跳过 12:00", "持续至 13:00"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, f := newWindow(t)
			w.section = SectionSchedule
			f.settings.Schedule.SkipPeriodsEnabled = tc.on
			f.settings.Schedule.SkipPeriods = tc.periods
			w.opts.Now = func() time.Time { return tc.now }
			tt := ui.NewTester(w.view, 800, 720)
			useLocale(t, tt, "zh-Hans")
			tt.Frame()
			if !containsText(tt, tc.want...) {
				t.Errorf("status copy %v is missing: %v", tc.want, tt.Texts())
			}
		})
	}
}

// TestSkipPeriodsClockDraft saves a valid edit of a time field.
func TestSkipPeriodsClockDraft(t *testing.T) {
	w, f := newWindow(t)
	w.section = SectionSchedule
	f.settings.Schedule.SkipPeriodsEnabled = true
	f.settings.Schedule.SkipPeriods = []settings.SkipPeriod{{Enabled: true, Start: "09:00", End: "10:00"}}
	tt := ui.NewTester(w.view, 800, 720)
	tt.Frame()
	field := namedAccess(t, tt, ui.RoleTextField, "Start time")
	tt.ClickAt(field.Bounds.X+field.Bounds.W/2, field.Bounds.Y+field.Bounds.H/2)
	tt.Command("selectAll")
	tt.Type("11:30")
	tt.Frame()
	if got := f.settings.Schedule.SkipPeriods[0].Start; got != "11:30" {
		t.Errorf("start time: %q, want 11:30", got)
	}
	if f.touched == 0 {
		t.Error("editing a time did not touch the settings")
	}
}

// TestNowFallsBackToClock keeps a view working when no clock was injected.
func TestNowFallsBackToClock(t *testing.T) {
	w, _ := newWindow(t)
	w.opts.Now = nil
	before := time.Now()
	got := w.now()
	if got.Before(before) || got.After(time.Now()) {
		t.Errorf("now: %v, want a time between %v and now", got, before)
	}
}
