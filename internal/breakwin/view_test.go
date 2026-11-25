package breakwin

import (
	"log/slog"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/gythialy/fermata/internal/ideas"
	"github.com/gythialy/fermata/internal/planner"
	"github.com/gythialy/fermata/internal/settings"
)

type fixture struct {
	m       *Manager
	actions []Action
}

func newFixture(t *testing.T) (*fixture, *ui.Tester) {
	t.Helper()
	f := &fixture{}
	bw := settings.Default().BreakWindow
	m := New(Options{
		Settings: func() settings.BreakWindow { return bw },
		Idea: func(planner.Kind) ideas.Idea {
			return ideas.Idea{Text: "Look out a window until your eyes relax."}
		},
		Clock:       func(time.Time) string { return "14:32" },
		Action:      func(a Action) { f.actions = append(f.actions, a) },
		Label:       func(kind planner.Kind, strict bool) string { return "MINI BREAK" },
		ActionLabel: func(a Action) string { return actionName(a) },
		Dark:        func() bool { return false },
		Log:         slog.Default(),
	})
	m.bw = bw
	m.idea = ideas.Idea{Text: "Look out a window until your eyes relax."}
	m.spec = planner.BreakSpec{
		Kind:     planner.Mini,
		Duration: 20 * time.Second,
		Postpone: planner.PostponeConfig{Enabled: true, Limit: 1, Percent: 30, Time: 2 * time.Minute},
	}
	m.phase = planner.Breaking
	f.m = m
	tt := ui.NewTester(func(c *ui.Context) { m.view(c, true) }, 1200, 800)
	now := tt.Now()
	m.spec.StartAt = now
	m.spec.EndAt = now.Add(20 * time.Second)
	return f, tt
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

func actionName(a Action) string {
	switch a {
	case Postpone:
		return "Postpone"
	case Skip:
		return "Skip"
	default:
		return "Finish"
	}
}

func TestBreakWindowShowsTheSuggestionAndTheExit(t *testing.T) {
	f, tt := newFixture(t)
	tt.Frame()
	if !tt.HasText("Look out a window until your eyes relax.") {
		t.Fatalf("texts %v", tt.Texts())
	}
	if !tt.HasText("MINI BREAK") || !tt.HasText("0:20") {
		t.Errorf("texts %v", tt.Texts())
	}
	if !tt.HasText("Postpone") {
		t.Errorf("no postpone exit: %v", tt.Texts())
	}
	click(t, tt, "Postpone")
	if len(f.actions) != 1 || f.actions[0] != Postpone {
		t.Fatalf("actions %v", f.actions)
	}
}

func TestBreakWindowAfterThePostponeWindow(t *testing.T) {
	_, tt := newFixture(t)
	tt.Frame()
	tt.Advance(7 * time.Second)
	tt.Frame()
	if tt.HasText("Postpone") {
		t.Errorf("postpone is still offered: %v", tt.Texts())
	}
	if !tt.HasText("Skip") {
		t.Errorf("skip is not offered: %v", tt.Texts())
	}
}

func TestBreakWindowManualFinish(t *testing.T) {
	f, tt := newFixture(t)
	f.m.phase = planner.ManualAwait
	f.m.spec.ManualFinish = true
	tt.Frame()
	now := tt.Now()
	f.m.spec.EndAt = now
	f.m.spec.StartAt = now.Add(-20 * time.Second)
	tt.Advance(7 * time.Second)
	tt.Frame()
	if !tt.HasText("+0:07") {
		t.Errorf("no overshoot: %v", tt.Texts())
	}
	click(t, tt, "Finish")
	if len(f.actions) != 1 || f.actions[0] != Finish {
		t.Fatalf("actions %v", f.actions)
	}
}

func TestStrictBreakHasNoExit(t *testing.T) {
	f, tt := newFixture(t)
	f.m.spec.Strict = true
	tt.Frame()
	for _, text := range []string{"Postpone", "Skip", "Finish"} {
		if tt.HasText(text) {
			t.Errorf("strict break offers %q: %v", text, tt.Texts())
		}
	}
}

func TestBreakWindowSettings(t *testing.T) {
	f, tt := newFixture(t)
	f.m.bw.ShowIdeas = false
	f.m.bw.ShowCurrentTime = true
	tt.Frame()
	if tt.HasText("Look out a window until your eyes relax.") {
		t.Error("ideas are off but shown")
	}
	if !tt.HasText("14:32") {
		t.Errorf("no clock: %v", tt.Texts())
	}
}

func TestBreakWindowFitsItsSmallestWindow(t *testing.T) {
	_, tt := newFixture(t)
	tt.SetSize(640, 480)
	tt.Frame()
	box, ok := tt.Find("Look out a window until your eyes relax.")
	if !ok {
		t.Fatalf("texts %v", tt.Texts())
	}
	if box.X < 0 || box.Y < 0 || box.X+box.W > 640 || box.Y+box.H > 480 {
		t.Errorf("the suggestion is outside the window: %+v", box)
	}
}
