//go:build e2e

// Package e2e drives a running e2e build of Fermata over the MyGo automation
// protocol, so that the cases below run unchanged on macOS, Windows and Linux.
//
// The app has to be running with the plugin on, and its settings window open:
//
//	FERMATA_E2E_DATA=$(mktemp -d) MYGO_AUTOMATION=tcp::9100 \
//	  build/darwin-universal/Fermata.app/Contents/MacOS/Fermata preferences
//
// Then, from the repository:
//
//	go test -tags e2e ./internal/e2e
//
// A test skips when nothing answers at FERMATA_E2E_ADDR, so the suite stays
// green where no app runs. FERMATA_E2E_APP names the platform under test when
// the app runs on another machine than the test.
package e2e

import (
	"os"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/gythialy/fermata/internal/automation"
)

const settle = 3 * time.Second

// countdown is the break window's remaining time, as "4:51".
var countdown = regexp.MustCompile(`^\d+:\d\d$`)

func addr() string {
	if v := os.Getenv("FERMATA_E2E_ADDR"); v != "" {
		return v
	}
	return "127.0.0.1:9100"
}

// platform is the platform under test, which is not the test's own when the
// app runs on another machine.
func platform() string {
	if v := os.Getenv("FERMATA_E2E_APP"); v != "" {
		return v
	}
	return runtime.GOOS
}

func dial(t *testing.T) *automation.Client {
	t.Helper()
	c, err := automation.Dial(addr(), 2*time.Second)
	if err != nil {
		t.Skipf("no app answering at %s: %v", addr(), err)
	}
	// A window that animates answers idle late, after its own timeout.
	c.SetTimeout(20 * time.Second)
	t.Cleanup(func() { c.Close() })
	return c
}

// settings returns the one settings window, by either of its titles: the app
// speaks the system's language unless a test changed it.
func settings(t *testing.T, c *automation.Client) automation.Window {
	t.Helper()
	windows, err := c.Windows()
	if err != nil {
		t.Fatalf("list windows: %v", err)
	}
	for _, title := range []string{"Settings", "设置"} {
		if w, ok := automation.FindWindow(windows, title); ok {
			return w
		}
	}
	t.Skipf("no settings window among %v: start the app with `preferences`", windows)
	return automation.Window{}
}

func nodes(t *testing.T, c *automation.Client, win int) []automation.Node {
	t.Helper()
	if err := c.Idle(win, settle); err != nil {
		t.Logf("window %d did not settle: %v", win, err)
	}
	list, _, err := c.Tree(win)
	if err != nil {
		t.Fatalf("tree of window %d: %v", win, err)
	}
	return list
}

// nodesNow reads the window without waiting for it to settle, which a break
// window never does while its countdown runs.
func nodesNow(t *testing.T, c *automation.Client, win int) []automation.Node {
	t.Helper()
	list, _, err := c.Tree(win)
	if err != nil {
		t.Fatalf("tree of window %d: %v", win, err)
	}
	return list
}

// hasText reports whether any element of the window says the text.
func hasText(list []automation.Node, want string) bool {
	for _, n := range list {
		if strings.Contains(n.Text(), want) {
			return true
		}
	}
	return false
}

// press acts on the one element the label names, and fails the test when the
// label is missing or ambiguous.
func press(t *testing.T, c *automation.Client, win int, label string) automation.Node {
	t.Helper()
	n, err := automation.Find(nodes(t, c, win), "", label, "press")
	if err != nil {
		t.Fatalf("%v", err)
	}
	if err := c.Act(win, n.ID, "", "press", ""); err != nil {
		t.Fatalf("press %q: %v", label, err)
	}
	return n
}

// tabs opens the settings page the label names.
func tab(t *testing.T, c *automation.Client, win int, page string) {
	t.Helper()
	press(t, c, win, page)
	if err := c.Idle(win, settle); err != nil {
		t.Logf("window %d did not settle: %v", win, err)
	}
}

// popUp returns the pop-up button nearest the top of the page, which on the
// General page is the language choice.
func popUp(t *testing.T, c *automation.Client, win int) automation.Node {
	t.Helper()
	var best automation.Node
	found := false
	for _, n := range nodes(t, c, win) {
		if n.Role != "popUpButton" {
			continue
		}
		if !found || n.Bounds.Y < best.Bounds.Y {
			best, found = n, true
		}
	}
	if !found {
		t.Fatalf("no pop-up button on window %d", win)
	}
	return best
}

// setLanguage picks the language the item names, from the General page's
// pop-up: "English", "简体中文", or the entry that follows the system.
func setLanguage(t *testing.T, c *automation.Client, win int, item string) {
	t.Helper()
	button := popUp(t, c, win)
	if err := c.Act(win, button.ID, "", "press", ""); err != nil {
		t.Fatalf("open the language pop-up: %v", err)
	}
	press(t, c, win, item)
	if err := c.Idle(win, settle); err != nil {
		t.Logf("window %d did not settle: %v", win, err)
	}
}

// currentLanguage reads the language the pop-up shows.
func currentLanguage(t *testing.T, c *automation.Client, win int) string {
	t.Helper()
	return popUp(t, c, win).Value
}

// TestMain leaves the app in the language it was found in, whichever tests
// ran, and asks for English first so that the labels the cases press are the
// ones this file names.
func TestMain(m *testing.M) {
	c, err := automation.Dial(addr(), 2*time.Second)
	if err != nil {
		os.Exit(m.Run())
	}
	windows, _ := c.Windows()
	var settingsWin automation.Window
	ok := false
	for _, title := range []string{"Settings", "设置"} {
		if w, found := automation.FindWindow(windows, title); found {
			settingsWin, ok = w, true
			break
		}
	}
	restore := ""
	if !ok {
		// A fresh data directory opens the welcome window first; its button
		// both finishes it and opens the settings this suite drives.
		if settingsWin, ok = openSettings(c); ok {
			windows = nil
		}
	}
	if ok {
		restore = valueAt(c, settingsWin.ID)
		setLanguageQuietly(c, settingsWin.ID, "English")
	}
	code := m.Run()
	if ok && restore != "" {
		setLanguageQuietly(c, settingsWin.ID, restore)
	}
	c.Close()
	os.Exit(code)
}

// openSettings presses the welcome window's way into the settings, and
// returns the window it opened.
func openSettings(c *automation.Client) (automation.Window, bool) {
	windows, err := c.Windows()
	if err != nil {
		return automation.Window{}, false
	}
	for _, w := range windows {
		list, _, err := c.Tree(w.ID)
		if err != nil {
			continue
		}
		for _, want := range []string{"Open Settings", "打开设置"} {
			target, err := automation.Find(list, "", want, "press")
			if err != nil {
				continue
			}
			if err := c.Act(w.ID, target.ID, "", "press", ""); err != nil {
				continue
			}
			time.Sleep(time.Second)
			after, err := c.Windows()
			if err != nil {
				return automation.Window{}, false
			}
			for _, title := range []string{"Settings", "设置"} {
				if found, ok := automation.FindWindow(after, title); ok {
					return found, true
				}
			}
		}
	}
	return automation.Window{}, false
}

// valueAt reads the language pop-up's value without a *testing.T, for TestMain.
func valueAt(c *automation.Client, win int) string {
	list, _, err := c.Tree(win)
	if err != nil {
		return ""
	}
	best, found := automation.Node{}, false
	for _, n := range list {
		if n.Role == "popUpButton" && (!found || n.Bounds.Y < best.Bounds.Y) {
			best, found = n, true
		}
	}
	if !found {
		return ""
	}
	return best.Value
}

func setLanguageQuietly(c *automation.Client, win int, item string) {
	list, _, err := c.Tree(win)
	if err != nil {
		return
	}
	best, found := automation.Node{}, false
	for _, n := range list {
		if n.Role == "popUpButton" && (!found || n.Bounds.Y < best.Bounds.Y) {
			best, found = n, true
		}
	}
	if !found {
		return
	}
	if best.Value == item {
		return
	}
	if err := c.Act(win, best.ID, "", "press", ""); err != nil {
		return
	}
	list, _, err = c.Tree(win)
	if err != nil {
		return
	}
	target, err := automation.Find(list, "", item, "press")
	if err != nil {
		return
	}
	_ = c.Act(win, target.ID, "", "press", "")
	_ = c.Idle(win, settle)
}

// TestSettingsPages opens each of the four pages and reads its title.
func TestSettingsPages(t *testing.T) {
	c := dial(t)
	w := settings(t, c)
	for _, page := range []string{"General", "Schedule", "Appearance", "About"} {
		tab(t, c, w.ID, page)
		if list := nodes(t, c, w.ID); !hasText(list, page) {
			t.Errorf("the %s page does not name itself", page)
		}
	}
	tab(t, c, w.ID, "General")
}

// TestEverySwitchIsNamed: assistive technology announces a switch by its name,
// and a named switch is also the only one automation can address without its
// ID, which changes with every frame.
func TestEverySwitchIsNamed(t *testing.T) {
	c := dial(t)
	w := settings(t, c)
	for _, page := range []string{"General", "Schedule", "Appearance"} {
		tab(t, c, w.ID, page)
		for _, n := range nodes(t, c, w.ID) {
			if n.Role == "switch" && n.Label == "" {
				t.Errorf("%s: the switch at %v has no name", page, n.Bounds)
			}
		}
	}
	tab(t, c, w.ID, "General")
}

// TestToggleAndRestore turns the one harmless switch on the General page on
// and off again.
func TestToggleAndRestore(t *testing.T) {
	c := dial(t)
	w := settings(t, c)
	tab(t, c, w.ID, "General")

	before, err := automation.Find(nodes(t, c, w.ID), "", "Wait while typing", "press")
	if err != nil {
		t.Fatalf("%v", err)
	}
	if err := c.Act(w.ID, before.ID, "", "press", ""); err != nil {
		t.Fatalf("press: %v", err)
	}
	after, err := automation.Find(nodes(t, c, w.ID), "", "Wait while typing", "press")
	if err != nil {
		t.Fatalf("%v", err)
	}
	if before.Has("checked") == after.Has("checked") {
		t.Errorf("the switch did not change: %v then %v", before.States, after.States)
	}
	// Leave it as it was found.
	if err := c.Act(w.ID, after.ID, "", "press", ""); err != nil {
		t.Fatalf("press again: %v", err)
	}
	restored, err := automation.Find(nodes(t, c, w.ID), "", "Wait while typing", "press")
	if err != nil {
		t.Fatalf("%v", err)
	}
	if restored.Has("checked") != before.Has("checked") {
		t.Errorf("the switch was not restored: %v, want %v", restored.States, before.States)
	}
}

// TestStepperStopsAtItsMinimum: the stepper's decrease button is disabled at
// the smallest value and enabled above it, which is the bound the app offers
// on every platform.
func TestStepperStopsAtItsMinimum(t *testing.T) {
	c := dial(t)
	w := settings(t, c)
	tab(t, c, w.ID, "General")

	decrease := func() automation.Node {
		t.Helper()
		// The button draws "−" and names itself in its description.
		for _, n := range nodes(t, c, w.ID) {
			if n.Role == "button" && (n.Description == "Decrease" || n.Description == "减少") {
				return n
			}
		}
		t.Fatal("no decrease button on the General page")
		return automation.Node{}
	}
	increase := func() automation.Node {
		t.Helper()
		for _, n := range nodes(t, c, w.ID) {
			if n.Role == "button" && (n.Description == "Increase" || n.Description == "增加") {
				return n
			}
		}
		t.Fatal("no increase button on the General page")
		return automation.Node{}
	}

	low := decrease()
	if !low.Has("disabled") {
		t.Fatalf("the stepper starts above its minimum: %v", low.States)
	}
	up := increase()
	if err := c.Act(w.ID, up.ID, "", "press", ""); err != nil {
		t.Fatalf("increase: %v", err)
	}
	if mid := decrease(); mid.Has("disabled") {
		t.Errorf("decrease is still disabled above the minimum: %v", mid.States)
	}
	down := decrease()
	if err := c.Act(w.ID, down.ID, "", "press", ""); err != nil {
		t.Fatalf("decrease: %v", err)
	}
	if again := decrease(); !again.Has("disabled") {
		t.Errorf("decrease is enabled at the minimum again: %v", again.States)
	}
}

// TestExclusionsFollowThePlatform: the rules need the system's process list,
// which Windows does not give the app.
func TestExclusionsFollowThePlatform(t *testing.T) {
	c := dial(t)
	w := settings(t, c)
	tab(t, c, w.ID, "General")

	shown := hasText(nodes(t, c, w.ID), "App exclusions")
	switch platform() {
	case "windows":
		if shown {
			t.Error("Windows offers exclusion rules it can never match")
		}
	default:
		if !shown {
			t.Errorf("the %s app hides exclusion rules it can match", platform())
		}
	}
}

// TestLanguageRoundTrip switches to the two shipped languages and back,
// reading the window title each time.
func TestLanguageRoundTrip(t *testing.T) {
	c := dial(t)
	w := settings(t, c)
	tab(t, c, w.ID, "General")

	for _, tc := range []struct{ item, title string }{
		{"简体中文", "设置"},
		{"English", "Settings"},
	} {
		setLanguage(t, c, w.ID, tc.item)
		windows, err := c.Windows()
		if err != nil {
			t.Fatalf("list windows: %v", err)
		}
		if _, ok := automation.FindWindow(windows, tc.title); !ok {
			t.Errorf("after choosing %s no window is titled %q: %v", tc.item, tc.title, windows)
		}
		tab(t, c, w.ID, pageIn(tc.title))
	}
}

// pageIn names the General page in the language of the settings title.
func pageIn(title string) string {
	if title == "设置" {
		return "通用"
	}
	return "General"
}

// TestBreakWindowIsReadable reads whichever break shows: a strict one has no
// exit, and a normal one offers postpone.
func TestBreakWindowIsReadable(t *testing.T) {
	c := dial(t)
	win, found := breakWindow(t, c)
	if !found {
		t.Skip("no break window: start one with `Fermata mini` on the app's machine")
	}
	list := nodesNow(t, c, win.ID)
	said := false
	for _, n := range list {
		if countdown.MatchString(strings.TrimSpace(n.Text())) {
			said = true
		}
	}
	if !said {
		t.Errorf("the break window shows no countdown: %v", list)
	}
	strict := hasText(list, "Strict") || hasText(list, "严格")
	buttons := 0
	for _, n := range list {
		if n.Role == "button" {
			buttons++
		}
	}
	switch {
	case strict && buttons != 0:
		t.Errorf("a strict break offers %d button(s)", buttons)
	case !strict && buttons == 0:
		t.Error("a break with an exit offers none")
	}
}

// breakWindow returns the window whose content shows a countdown.
func breakWindow(t *testing.T, c *automation.Client) (automation.Window, bool) {
	t.Helper()
	windows, err := c.Windows()
	if err != nil {
		t.Fatalf("list windows: %v", err)
	}
	for _, w := range windows {
		list, _, err := c.Tree(w.ID)
		if err != nil {
			continue
		}
		for _, n := range list {
			if countdown.MatchString(strings.TrimSpace(n.Text())) {
				return w, true
			}
		}
	}
	return automation.Window{}, false
}
