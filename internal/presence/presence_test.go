package presence

import (
	"errors"
	"testing"
	"time"
)

var at = time.Date(2026, 11, 25, 9, 0, 0, 0, time.UTC)

func newManager(t *testing.T, cfg Config) (*Manager, *[]Event) {
	t.Helper()
	events := []Event{}
	m := New(Source{
		Dispatch: func(f func()) { f() },
		Now:      func() time.Time { return at },
	}, func(e Event) { events = append(events, e) })
	m.Configure(cfg)
	t.Cleanup(m.Stop)
	return m, &events
}

func kinds(events []Event) []Kind {
	out := make([]Kind, 0, len(events))
	for _, e := range events {
		out = append(out, e.Kind)
	}
	return out
}

func TestIdlePausesOnce(t *testing.T) {
	m, events := newManager(t, Config{IdleEnabled: true, IdleReset: 5 * time.Minute})
	m.PollIdle(at, 10*time.Second)
	if len(*events) != 0 {
		t.Fatalf("events while typing: %v", kinds(*events))
	}
	m.PollIdle(at, 4*time.Minute)
	if len(*events) != 0 {
		t.Fatalf("events while away but not long enough: %v", kinds(*events))
	}
	m.PollIdle(at, 5*time.Minute)
	m.PollIdle(at, 6*time.Minute)
	got := kinds(*events)
	if len(got) != 1 || got[0] != IdleEnter {
		t.Fatalf("events: %v", got)
	}
	m.PollIdle(at, time.Second)
	got = kinds(*events)
	if len(got) != 2 || got[1] != IdleLeave {
		t.Fatalf("events: %v", got)
	}
}

func TestIdleDisabled(t *testing.T) {
	m, events := newManager(t, Config{IdleEnabled: false})
	m.PollIdle(at, time.Hour)
	if len(*events) != 0 {
		t.Fatalf("events while idle is off: %v", kinds(*events))
	}
}

func TestDND(t *testing.T) {
	m, events := newManager(t, Config{DNDEnabled: true})
	m.PollDND(true, nil)
	m.PollDND(true, nil)
	m.PollDND(false, nil)
	got := kinds(*events)
	if len(got) != 2 || got[0] != DNDEnter || got[1] != DNDLeave {
		t.Fatalf("events: %v", got)
	}
}

func TestDNDReadFailureIsReportedOnce(t *testing.T) {
	m, events := newManager(t, Config{DNDEnabled: true})
	err := errors.New("no permission")
	m.PollDND(true, err)
	m.PollDND(true, err)
	got := kinds(*events)
	if len(got) != 1 || got[0] != Unavailable {
		t.Fatalf("events: %v", got)
	}
	if m.dndOn {
		t.Error("an unreadable Focus counts as on")
	}
	if !m.dndBroken.Load() {
		t.Error("the Focus probe keeps running after it failed")
	}
	// A settings change tries once more.
	m.Configure(Config{DNDEnabled: true})
	if m.dndBroken.Load() {
		t.Error("the Focus probe was not reset for a new setting")
	}
}

func TestExclusionRule(t *testing.T) {
	rule := &Rule{Name: "Meeting", Processes: []string{"zoom"}, Pause: true}
	m, events := newManager(t, Config{Rule: rule})
	m.PollExclusion(rule, true, nil)
	m.PollExclusion(rule, true, nil)
	m.PollExclusion(rule, false, nil)
	got := kinds(*events)
	if len(got) != 2 || got[0] != ExclusionEnter || got[1] != ExclusionLeave {
		t.Fatalf("events: %v", got)
	}
}

func TestResumeRuleAllowsOnlyWhileRunning(t *testing.T) {
	rule := &Rule{Name: "Focus app", Processes: []string{"focus"}, Pause: false}
	m, events := newManager(t, Config{Rule: rule})
	m.PollExclusion(rule, false, nil)
	if got := kinds(*events); len(got) != 1 || got[0] != ExclusionEnter {
		t.Fatalf("not running: %v", got)
	}
	m.PollExclusion(rule, true, nil)
	if got := kinds(*events); len(got) != 2 || got[1] != ExclusionLeave {
		t.Fatalf("running: %v", got)
	}
}

func TestDisablingReleasesThePause(t *testing.T) {
	m, events := newManager(t, Config{IdleEnabled: true, IdleReset: time.Minute})
	m.PollIdle(at, time.Hour)
	if got := kinds(*events); len(got) != 1 || got[0] != IdleEnter {
		t.Fatalf("events: %v", got)
	}
	m.Configure(Config{IdleEnabled: false, IdleReset: time.Minute})
	if got := kinds(*events); len(got) != 2 || got[1] != IdleLeave {
		t.Fatalf("events after disabling: %v", got)
	}
}

func TestRuleChangeReleasesThePause(t *testing.T) {
	rule := &Rule{Name: "Meeting", Processes: []string{"zoom"}, Pause: true}
	m, events := newManager(t, Config{Rule: rule})
	m.PollExclusion(rule, true, nil)
	m.Configure(Config{Rule: nil})
	if got := kinds(*events); len(got) != 2 || got[1] != ExclusionLeave {
		t.Fatalf("events: %v", got)
	}
}
