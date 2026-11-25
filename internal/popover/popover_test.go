package popover

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

// waiting is the flyout of the default rhythm: a mini break in 21 minutes,
// with the two actions the waiting state offers.
func waiting() Options {
	return Options{
		State:     "Next mini break",
		Remaining: "21m",
		Detail:    "Long break after 4 more mini breaks",
		Share:     0.34,
		Actions: []Action{
			{Label: "Take a break now", Primary: true},
			{Label: "Pause 30 minutes"},
		},
		Settings: func() {},
		Menu:     func() {},
		Close:    func() {},
	}
}

// draw renders opts once and returns the tester.
func draw(t *testing.T, opts *Options) *ui.Tester {
	t.Helper()
	tt := ui.NewTester(func(c *ui.Context) { Show(c, *opts) }, 320, 360)
	tt.Frame()
	return tt
}

func wantText(t *testing.T, tt *ui.Tester, texts ...string) {
	t.Helper()
	shown := strings.Join(tt.Texts(), "\n")
	for _, text := range texts {
		if !strings.Contains(shown, text) {
			t.Errorf("no %q in the flyout:\n%s", text, shown)
		}
	}
}

func TestWaitingShowsTheStateTheCountdownAndTheActions(t *testing.T) {
	opts := waiting()
	tt := draw(t, &opts)
	wantText(t, tt,
		"NEXT MINI BREAK",
		"21m",
		"Long break after 4 more mini breaks",
		"Take a break now",
		"Pause 30 minutes",
		"Settings…",
	)
}

func TestThePrimaryActionRunsAndTheOtherDoesNot(t *testing.T) {
	var ran []string
	opts := waiting()
	opts.Actions = []Action{
		{Label: "Take a break now", Primary: true, Run: func() { ran = append(ran, "take") }},
		{Label: "Pause 30 minutes", Run: func() { ran = append(ran, "pause") }},
	}
	tt := draw(t, &opts)
	click(t, tt, "Take a break now")
	if len(ran) != 1 || ran[0] != "take" {
		t.Errorf("ran %v, want [take]", ran)
	}
}

func TestTheHeaderButtonsCarryTheirNames(t *testing.T) {
	opts := waiting()
	tt := draw(t, &opts)
	for _, name := range []string{"Show menu", "Close"} {
		if _, ok := tt.Access(name); !ok {
			t.Errorf("no accessible %q", name)
		}
	}
}

func TestStrictShowsNoActionsAndNoFooter(t *testing.T) {
	opts := Options{
		State:     "Strict · Mini break",
		Remaining: "0:12",
		Detail:    "No exit until it ends.",
		Share:     0.94,
	}
	tt := draw(t, &opts)
	wantText(t, tt, "STRICT · MINI BREAK", "0:12", "No exit until it ends.")
	shown := strings.Join(tt.Texts(), "\n")
	if strings.Contains(shown, "Settings") {
		t.Errorf("a strict break shows the settings row:\n%s", shown)
	}
	if _, ok := tt.AccessByRole(ui.RoleButton, ""); ok {
		t.Errorf("a strict break shows a button: %v", tt.AccessAll())
	}
}

func TestTheRulerReportsHowMuchPassed(t *testing.T) {
	opts := waiting()
	tt := draw(t, &opts)
	if _, ok := tt.Access("34%"); !ok {
		t.Errorf("the ruler is not labelled with its share: %v", tt.AccessAll())
	}
}

func TestTheCountdownFollowsTheState(t *testing.T) {
	opts := waiting()
	tt := draw(t, &opts)
	opts.Remaining = "12:04"
	opts.Share = 0.19
	tt.Frame()
	wantText(t, tt, "12:04")
	if !tt.HasText("12:04") {
		t.Errorf("the countdown did not change: %v", tt.Texts())
	}
}

func TestDarkDrawsTheSameContent(t *testing.T) {
	opts := waiting()
	tt := draw(t, &opts)
	tt.SetDark(true)
	tt.Frame()
	wantText(t, tt, "21m", "Take a break now")
}

// click presses the button whose accessible name is name.
func click(t *testing.T, tt *ui.Tester, name string) {
	t.Helper()
	for _, a := range tt.AccessAll() {
		if a.Role == ui.RoleButton && a.Name == name {
			tt.ClickAt(a.Bounds.X+a.Bounds.W/2, a.Bounds.Y+a.Bounds.H/2)
			tt.Frame()
			return
		}
	}
	t.Fatalf("no button %q: %v", name, tt.AccessAll())
}
