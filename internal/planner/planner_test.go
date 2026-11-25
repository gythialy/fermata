package planner

import (
	"math/rand/v2"
	"testing"
	"time"
)

var t0 = time.Date(2026, 11, 25, 9, 0, 0, 0, time.UTC)

func defaultConfig() Config {
	return Config{
		Mini: BreakConfig{
			Enabled:  true,
			Duration: 20 * time.Second,
			Interval: 10 * time.Minute,
			Notify:   true,
			Lead:     10 * time.Second,
			Postpone: PostponeConfig{Enabled: true, Time: 2 * time.Minute, Limit: 1, Percent: 30},
		},
		Long: BreakConfig{
			Enabled:  true,
			Duration: 5 * time.Minute,
			Interval: 30 * time.Minute,
			Notify:   true,
			Lead:     30 * time.Second,
			Postpone: PostponeConfig{Enabled: true, Time: 5 * time.Minute, Limit: 1, Percent: 30},
		},
		EveryNBreaks:   2,
		PauseDurations: []time.Duration{30 * time.Minute, time.Hour},
		MorningHour:    6,
	}
}

func started(t *testing.T) *Planner {
	t.Helper()
	p := New(defaultConfig())
	p.Step(Reset{Now: t0})
	if got := p.State(); got.Phase != Waiting || got.N != 1 || got.Kind != Mini {
		t.Fatalf("after reset: %+v", got)
	}
	return p
}

func effects[T Effect](eff []Effect) []T {
	var out []T
	for _, e := range eff {
		if v, ok := e.(T); ok {
			out = append(out, v)
		}
	}
	return out
}

func armAt(t *testing.T, eff []Effect) time.Time {
	t.Helper()
	arms := effects[Arm](eff)
	if len(arms) != 1 {
		t.Fatalf("arms: %v", eff)
	}
	return arms[0].At
}

func closeReasons(eff []Effect) []CloseReason {
	var out []CloseReason
	for _, c := range effects[CloseBreak](eff) {
		out = append(out, c.Reason)
	}
	return out
}

func TestDefaultSequence(t *testing.T) {
	p := started(t)
	st := p.State()
	if want := t0.Add(10 * time.Minute); !st.BreakAt.Equal(want) {
		t.Errorf("break at %v, want %v", st.BreakAt, want)
	}
	if want := t0.Add(9*time.Minute + 50*time.Second); !st.NotifyAt.Equal(want) {
		t.Errorf("notice at %v, want %v", st.NotifyAt, want)
	}

	eff := p.Step(Tick{Now: t0.Add(9*time.Minute + 50*time.Second)})
	notices := effects[Notify](eff)
	if len(notices) != 1 || notices[0].Kind != Mini || notices[0].Lead != 10*time.Second {
		t.Fatalf("notice effects: %v", eff)
	}
	if p.State().Phase != Notified {
		t.Fatalf("phase %v", p.State().Phase)
	}

	eff = p.Step(Tick{Now: t0.Add(10 * time.Minute)})
	breaks := effects[StartBreak](eff)
	if len(breaks) != 1 {
		t.Fatalf("start effects: %v", eff)
	}
	spec := breaks[0].Spec
	if spec.Kind != Mini || spec.Duration != 20*time.Second || !spec.StartAt.Equal(t0.Add(10*time.Minute)) {
		t.Errorf("spec %+v", spec)
	}
	if got := armAt(t, eff); !got.Equal(t0.Add(10*time.Minute + 20*time.Second)) {
		t.Errorf("armed at %v", got)
	}

	eff = p.Step(Tick{Now: t0.Add(10*time.Minute + 20*time.Second)})
	if got := closeReasons(eff); len(got) != 1 || got[0] != Completed {
		t.Fatalf("close: %v", eff)
	}
	st = p.State()
	if st.Phase != Waiting || st.N != 2 || st.Kind != Mini {
		t.Fatalf("after break: %+v", st)
	}
	if want := t0.Add(20*time.Minute + 20*time.Second); !st.BreakAt.Equal(want) {
		t.Errorf("next break at %v, want %v", st.BreakAt, want)
	}
}

func TestLongBreakAfterTwoMinis(t *testing.T) {
	p := started(t)
	now := t0
	for range 2 {
		now = p.State().BreakAt
		p.Step(Tick{Now: now})
		now = p.State().EndAt
		p.Step(Tick{Now: now})
	}
	st := p.State()
	if st.N != 3 || st.Kind != Long {
		t.Fatalf("third break: %+v", st)
	}
	if want := now.Add(10 * time.Minute); !st.BreakAt.Equal(want) {
		t.Errorf("long break at %v, want %v", st.BreakAt, want)
	}
	// The long break's notice is 30 seconds before.
	eff := p.Step(Tick{Now: st.BreakAt.Add(-30 * time.Second)})
	notices := effects[Notify](eff)
	if len(notices) != 1 || notices[0].Kind != Long || notices[0].Lead != 30*time.Second {
		t.Fatalf("notice: %v", eff)
	}
}

func TestMissedBreakIsNotShown(t *testing.T) {
	p := started(t)
	late := t0.Add(10*time.Minute + 3*time.Second)
	eff := p.Step(Tick{Now: late})
	if len(effects[StartBreak](eff)) != 0 {
		t.Fatalf("a late break started: %v", eff)
	}
	st := p.State()
	if st.N != 2 || !st.Missed {
		t.Fatalf("after miss: %+v", st)
	}
	if want := late.Add(10 * time.Minute); !st.BreakAt.Equal(want) {
		t.Errorf("next break at %v, want %v", st.BreakAt, want)
	}
}

func TestMissedNoticeCloses(t *testing.T) {
	p := started(t)
	p.Step(Tick{Now: t0.Add(9*time.Minute + 50*time.Second)})
	eff := p.Step(Tick{Now: t0.Add(10*time.Minute + 3*time.Second)})
	if len(effects[CloseNotice](eff)) != 1 {
		t.Fatalf("notice not closed: %v", eff)
	}
	if len(effects[StartBreak](eff)) != 0 {
		t.Fatalf("a late break started: %v", eff)
	}
	if st := p.State(); st.N != 2 {
		t.Errorf("n %d", st.N)
	}
}

func TestPostponeWithinWindow(t *testing.T) {
	p := started(t)
	p.Step(Tick{Now: t0.Add(10 * time.Minute)})
	now := t0.Add(10*time.Minute + 5*time.Second)
	if !p.CanPostpone(now) {
		t.Fatal("postpone not offered")
	}
	eff := p.Step(Postpone{Now: now})
	if got := closeReasons(eff); len(got) != 1 || got[0] != Postponed {
		t.Fatalf("close: %v", eff)
	}
	st := p.State()
	if st.N != 1 || st.Kind != Mini {
		t.Fatalf("sequence moved: %+v", st)
	}
	if want := now.Add(2 * time.Minute); !st.BreakAt.Equal(want) {
		t.Errorf("break at %v, want %v", st.BreakAt, want)
	}
	if want := now.Add(2*time.Minute - 10*time.Second); !st.NotifyAt.Equal(want) {
		t.Errorf("notice at %v, want %v", st.NotifyAt, want)
	}
	// The same break starts again, and cannot be postponed twice.
	p.Step(Tick{Now: st.BreakAt})
	again := p.State()
	if again.Phase != Breaking || again.Duration != 20*time.Second || again.N != 1 {
		t.Fatalf("restarted break: %+v", again)
	}
	if p.CanPostpone(again.StartAt) {
		t.Error("postponed twice")
	}
	if !p.CanSkip(again.StartAt) {
		t.Error("skip not offered once postpone is used")
	}
}

func TestPostponeWindowClosed(t *testing.T) {
	p := started(t)
	p.Step(Tick{Now: t0.Add(10 * time.Minute)})
	now := t0.Add(10*time.Minute + 7*time.Second) // 35% of 20s
	if p.CanPostpone(now) {
		t.Fatal("postpone offered after its window")
	}
	if eff := p.Step(Postpone{Now: now}); len(effects[CloseBreak](eff)) != 0 {
		t.Fatalf("postponed after its window: %v", eff)
	}
	if !p.CanSkip(now) {
		t.Fatal("skip not offered")
	}
	eff := p.Step(Skip{Now: now})
	if got := closeReasons(eff); len(got) != 1 || got[0] != Skipped {
		t.Fatalf("close: %v", eff)
	}
	st := p.State()
	if st.N != 2 || st.Phase != Waiting {
		t.Fatalf("after skip: %+v", st)
	}
	if want := now.Add(10 * time.Minute); !st.BreakAt.Equal(want) {
		t.Errorf("next break at %v, want %v", st.BreakAt, want)
	}
}

func TestSkipRefusedWhilePostponeOffered(t *testing.T) {
	p := started(t)
	p.Step(Tick{Now: t0.Add(10 * time.Minute)})
	now := t0.Add(10*time.Minute + time.Second)
	if eff := p.Step(Skip{Now: now}); len(effects[CloseBreak](eff)) != 0 {
		t.Fatalf("skipped while postpone was offered: %v", eff)
	}
	if p.State().Phase != Breaking {
		t.Fatalf("phase %v", p.State().Phase)
	}
}

func TestStrictBreak(t *testing.T) {
	cfg := defaultConfig()
	cfg.Mini.Strict = true
	p := New(cfg)
	p.Step(Reset{Now: t0})
	p.Step(Tick{Now: t0.Add(10 * time.Minute)})
	now := t0.Add(10*time.Minute + time.Second)
	if p.CanPostpone(now) || p.CanSkip(now) || p.CanPause() {
		t.Fatal("strict break offers an exit")
	}
	if eff := p.Step(Postpone{Now: now}); len(effects[CloseBreak](eff)) != 0 {
		t.Fatalf("postponed a strict break: %v", eff)
	}
	if eff := p.Step(Skip{Now: now}); len(effects[CloseBreak](eff)) != 0 {
		t.Fatalf("skipped a strict break: %v", eff)
	}
	if eff := p.Step(Pause{Now: now, Reason: Manual}); len(effects[CloseBreak](eff)) != 0 {
		t.Fatalf("paused a strict break: %v", eff)
	}
	// The system's lock ends it.
	eff := p.Step(Pause{Now: now, Reason: Lock})
	if got := closeReasons(eff); len(got) != 1 || got[0] != Interrupted {
		t.Fatalf("lock: %v", eff)
	}
	if st := p.State(); st.Phase != Paused || !st.Reasons.Has(Lock) {
		t.Fatalf("after lock: %+v", st)
	}
	eff = p.Step(Resume{Now: now.Add(time.Minute), Reason: Lock})
	if st := p.State(); st.Phase != Waiting || st.N != 1 {
		t.Fatalf("after unlock: %+v", st)
	}
	if len(effects[NotifyResume](eff)) != 0 {
		t.Errorf("a lock resume notified: %v", eff)
	}
}

func TestManualFinish(t *testing.T) {
	cfg := defaultConfig()
	cfg.Mini.ManualFinish = true
	p := New(cfg)
	p.Step(Reset{Now: t0})
	p.Step(Tick{Now: t0.Add(10 * time.Minute)})
	end := p.State().EndAt
	eff := p.Step(Tick{Now: end})
	if len(effects[CloseBreak](eff)) != 0 {
		t.Fatalf("closed a manual finish break: %v", eff)
	}
	if got := effects[PlaySound](eff); len(got) != 1 || got[0].Name != SoundEnd {
		t.Fatalf("sound: %v", eff)
	}
	if st := p.State(); st.Phase != ManualAwait || !st.ManualFinish {
		t.Fatalf("phase %v", st.Phase)
	}
	// It stays until finished.
	if eff := p.Step(Tick{Now: end.Add(5 * time.Second)}); len(effects[CloseBreak](eff)) != 0 {
		t.Fatalf("closed on its own: %v", eff)
	}
	if !p.CanFinish() {
		t.Error("finish not offered")
	}
	eff = p.Step(Finish{Now: end.Add(7 * time.Second)})
	if got := closeReasons(eff); len(got) != 1 || got[0] != Completed {
		t.Fatalf("finish: %v", eff)
	}
	st := p.State()
	if st.Phase != Waiting || st.N != 2 {
		t.Fatalf("after finish: %+v", st)
	}
	if want := end.Add(7*time.Second + 10*time.Minute); !st.BreakAt.Equal(want) {
		t.Errorf("next break at %v, want %v", st.BreakAt, want)
	}
}

func TestManualPauseClosesBreak(t *testing.T) {
	p := started(t)
	p.Step(Tick{Now: t0.Add(10 * time.Minute)})
	now := t0.Add(10*time.Minute + 5*time.Second)
	eff := p.Step(Pause{Now: now, Reason: Manual})
	if got := closeReasons(eff); len(got) != 1 || got[0] != Interrupted {
		t.Fatalf("pause: %v", eff)
	}
	st := p.State()
	if st.Phase != Paused || st.N != 0 || !st.Reasons.Has(Manual) {
		t.Fatalf("after pause: %+v", st)
	}
	if !p.CanResume() {
		t.Error("resume not offered")
	}
	eff = p.Step(Resume{Now: now.Add(2 * time.Minute), Reason: Manual})
	if len(effects[NotifyResume](eff)) != 1 {
		t.Fatalf("resume did not notify: %v", eff)
	}
	st = p.State()
	if st.Phase != Waiting || st.N != 1 {
		t.Fatalf("after resume: %+v", st)
	}
	if want := now.Add(2*time.Minute + 10*time.Minute); !st.BreakAt.Equal(want) {
		t.Errorf("next break at %v, want %v", st.BreakAt, want)
	}
}

func TestTimedPauseExpires(t *testing.T) {
	p := started(t)
	p.Step(Pause{Now: t0.Add(time.Minute), Reason: Manual, Until: t0.Add(31 * time.Minute)})
	if st := p.State(); st.Phase != Paused || !st.PauseUntil.Equal(t0.Add(31*time.Minute)) {
		t.Fatalf("after pause: %+v", st)
	}
	eff := p.Step(Tick{Now: t0.Add(30 * time.Minute)})
	if len(effects[Arm](eff)) != 1 || len(effects[NotifyResume](eff)) != 0 {
		t.Fatalf("early tick: %v", eff)
	}
	eff = p.Step(Tick{Now: t0.Add(31 * time.Minute)})
	if len(effects[NotifyResume](eff)) != 1 {
		t.Fatalf("expiry did not notify: %v", eff)
	}
	st := p.State()
	if st.Phase != Waiting || st.N != 1 {
		t.Fatalf("after expiry: %+v", st)
	}
	if want := t0.Add(31*time.Minute + 10*time.Minute); !st.BreakAt.Equal(want) {
		t.Errorf("next break at %v, want %v", st.BreakAt, want)
	}
}

func TestDndDuringBreakHolds(t *testing.T) {
	p := started(t)
	p.Step(Tick{Now: t0.Add(10 * time.Minute)})
	now := t0.Add(10*time.Minute + 2*time.Second)
	eff := p.Step(Pause{Now: now, Reason: DND})
	if len(effects[CloseBreak](eff)) != 0 {
		t.Fatalf("dnd closed the break: %v", eff)
	}
	if st := p.State(); st.Phase != Breaking || !st.Reasons.Has(DND) {
		t.Fatalf("during dnd: %+v", st)
	}
	end := p.State().EndAt
	p.Step(Tick{Now: end})
	st := p.State()
	if st.Phase != Paused || st.N != 0 || !st.Reasons.Has(DND) {
		t.Fatalf("after break: %+v", st)
	}
	eff = p.Step(Resume{Now: end.Add(time.Minute), Reason: DND})
	if len(effects[NotifyResume](eff)) != 0 {
		t.Errorf("a dnd resume notified: %v", eff)
	}
	if st := p.State(); st.Phase != Waiting || st.N != 1 {
		t.Fatalf("after dnd: %+v", st)
	}
}

func TestPauseWhileWaiting(t *testing.T) {
	p := started(t)
	eff := p.Step(Pause{Now: t0.Add(time.Minute), Reason: Idle})
	if st := p.State(); st.Phase != Paused || st.N != 0 || !st.BreakAt.IsZero() {
		t.Fatalf("after idle: %+v", st)
	}
	if len(effects[Disarm](eff)) != 1 {
		t.Fatalf("no disarm: %v", eff)
	}
	p.Step(Resume{Now: t0.Add(2 * time.Minute), Reason: Idle})
	st := p.State()
	if st.Phase != Waiting || st.N != 1 {
		t.Fatalf("after return: %+v", st)
	}
	if want := t0.Add(2*time.Minute + 10*time.Minute); !st.BreakAt.Equal(want) {
		t.Errorf("next break at %v, want %v", st.BreakAt, want)
	}
}

func TestResetKeepsPause(t *testing.T) {
	p := started(t)
	p.Step(Pause{Now: t0, Reason: Manual})
	eff := p.Step(Reset{Now: t0.Add(time.Minute)})
	if st := p.State(); st.Phase != Paused || st.N != 0 {
		t.Fatalf("reset resumed a paused app: %+v", st)
	}
	if len(effects[Arm](eff)) != 0 {
		t.Errorf("a paused reset armed a timer: %v", eff)
	}
}

func TestResetWhileWaiting(t *testing.T) {
	p := started(t)
	now := t0.Add(4 * time.Minute)
	p.Step(Reset{Now: now})
	st := p.State()
	if st.N != 1 || st.Phase != Waiting {
		t.Fatalf("after reset: %+v", st)
	}
	if want := now.Add(10 * time.Minute); !st.BreakAt.Equal(want) {
		t.Errorf("break at %v, want %v", st.BreakAt, want)
	}
}

func TestShowStartsAHeldBreak(t *testing.T) {
	p := started(t)
	// The app does not step at the deadline: a fullscreen app holds the break.
	due := p.State().BreakAt
	held := due.Add(47 * time.Second)
	if st := p.State(); st.Phase != Waiting {
		t.Fatalf("phase %v", st.Phase)
	}
	eff := p.Step(Show{Now: held})
	breaks := effects[StartBreak](eff)
	if len(breaks) != 1 {
		t.Fatalf("show effects: %v", eff)
	}
	spec := breaks[0].Spec
	if !spec.StartAt.Equal(held) || spec.Duration != 20*time.Second {
		t.Errorf("spec %+v", spec)
	}
	if st := p.State(); st.Phase != Breaking || !st.EndAt.Equal(held.Add(20*time.Second)) {
		t.Errorf("after show: %+v", st)
	}
	// Shown while still not due changes nothing.
	p2 := started(t)
	if eff := p2.Step(Show{Now: t0.Add(time.Minute)}); len(effects[StartBreak](eff)) != 0 {
		t.Fatalf("shown before its time: %v", eff)
	}
}

func TestShowWaitsForTheNoticeToo(t *testing.T) {
	p := started(t)
	notice := p.State().NotifyAt
	eff := p.Step(Show{Now: notice})
	if len(effects[StartBreak](eff)) != 0 {
		t.Fatalf("a notice started a break: %v", eff)
	}
}

func TestSkipToMiniWhenLongDue(t *testing.T) {
	p := started(t)
	// Reach n=3, the long break.
	for range 2 {
		p.Step(Tick{Now: p.State().BreakAt})
		p.Step(Tick{Now: p.State().EndAt})
	}
	now := p.State().BreakAt.Add(-time.Minute)
	eff := p.Step(SkipTo{Now: now, Kind: Mini})
	if st := p.State(); st.N != 1 || st.Kind != Mini || st.Phase != Waiting {
		t.Fatalf("skip to mini: %+v", st)
	}
	if got := armAt(t, eff); !got.Equal(now.Add(100 * time.Millisecond)) {
		t.Errorf("armed at %v", got)
	}
	p.Step(Tick{Now: now.Add(100 * time.Millisecond)})
	if st := p.State(); st.Phase != Breaking || st.Kind != Mini {
		t.Fatalf("started %+v", st)
	}
}

func TestSkipToLong(t *testing.T) {
	p := started(t)
	now := t0.Add(2 * time.Minute)
	p.Step(SkipTo{Now: now, Kind: Long})
	if st := p.State(); st.N != 3 || st.Kind != Long {
		t.Fatalf("skip to long: %+v", st)
	}
	p.Step(Tick{Now: now.Add(100 * time.Millisecond)})
	st := p.State()
	if st.Phase != Breaking || st.Duration != 5*time.Minute {
		t.Fatalf("started %+v", st)
	}
}

func TestSkipToDisabledKindRefused(t *testing.T) {
	cfg := defaultConfig()
	cfg.Long.Enabled = false
	p := New(cfg)
	p.Step(Reset{Now: t0})
	if eff := p.Step(SkipTo{Now: t0, Kind: Long}); len(effects[Arm](eff)) != 1 {
		t.Fatalf("skip to a disabled kind: %v", eff)
	}
	if st := p.State(); st.Kind != Mini || st.N != 1 {
		t.Fatalf("after refused skip: %+v", st)
	}
}

func TestSetConfigWhileWaiting(t *testing.T) {
	p := started(t)
	cfg := defaultConfig()
	cfg.Mini.Interval = 5 * time.Minute
	now := t0.Add(time.Minute)
	p.Step(SetConfig{Now: now, Config: cfg})
	st := p.State()
	if want := t0.Add(5 * time.Minute); !st.BreakAt.Equal(want) {
		t.Fatalf("break at %v, want %v", st.BreakAt, want)
	}
	if want := t0.Add(4*time.Minute + 50*time.Second); !st.NotifyAt.Equal(want) {
		t.Errorf("notice at %v, want %v", st.NotifyAt, want)
	}
}

func TestSetConfigDuringBreakKeepsSnapshot(t *testing.T) {
	p := started(t)
	p.Step(Tick{Now: t0.Add(10 * time.Minute)})
	before := p.State()
	cfg := defaultConfig()
	cfg.Mini.Duration = 90 * time.Second
	p.Step(SetConfig{Now: t0.Add(10*time.Minute + 2*time.Second), Config: cfg})
	after := p.State()
	if !after.EndAt.Equal(before.EndAt) || after.Duration != 20*time.Second {
		t.Fatalf("snapshot changed: %+v", after)
	}
}

func TestWakeNeverShowsAMissedBreak(t *testing.T) {
	p := started(t)
	late := t0.Add(10*time.Minute + time.Second)
	eff := p.Step(Wake{Now: late})
	if len(effects[StartBreak](eff)) != 0 {
		t.Fatalf("a wake started a break: %v", eff)
	}
	if st := p.State(); st.N != 2 {
		t.Fatalf("after wake: %+v", st)
	}
}

func TestWakeRearms(t *testing.T) {
	p := started(t)
	eff := p.Step(Wake{Now: t0.Add(time.Minute)})
	if got := armAt(t, eff); !got.Equal(t0.Add(9*time.Minute + 50*time.Second)) {
		t.Errorf("armed at %v, want the notice", got)
	}
}

func TestOnlyLongBreaks(t *testing.T) {
	cfg := defaultConfig()
	cfg.Mini.Enabled = false
	p := New(cfg)
	p.Step(Reset{Now: t0})
	st := p.State()
	if st.Kind != Long {
		t.Fatalf("kind %v", st.Kind)
	}
	if want := t0.Add(30 * time.Minute); !st.BreakAt.Equal(want) {
		t.Errorf("break at %v, want %v", st.BreakAt, want)
	}
}

func TestOnlyMiniBreaks(t *testing.T) {
	cfg := defaultConfig()
	cfg.Long.Enabled = false
	p := New(cfg)
	p.Step(Reset{Now: t0})
	for range 6 {
		p.Step(Tick{Now: p.State().BreakAt})
		if st := p.State(); st.Kind != Mini {
			t.Fatalf("break %d is %v", st.N, st.Kind)
		}
		p.Step(Tick{Now: p.State().EndAt})
	}
}

func TestLeadClampedToHalfTheGap(t *testing.T) {
	cfg := defaultConfig()
	cfg.Mini.Lead = 20 * time.Minute // longer than the interval
	p := New(cfg)
	p.Step(Reset{Now: t0})
	st := p.State()
	if want := t0.Add(5 * time.Minute); !st.NotifyAt.Equal(want) {
		t.Errorf("notice at %v, want %v", st.NotifyAt, want)
	}
}

func TestNotificationOff(t *testing.T) {
	cfg := defaultConfig()
	cfg.Mini.Notify = false
	p := New(cfg)
	p.Step(Reset{Now: t0})
	if st := p.State(); !st.NotifyAt.IsZero() {
		t.Fatalf("notice at %v with notifications off", st.NotifyAt)
	}
	eff := p.Step(Tick{Now: t0.Add(9*time.Minute + 50*time.Second)})
	if len(effects[Notify](eff)) != 0 {
		t.Fatalf("notified: %v", eff)
	}
	if p.State().Phase != Waiting {
		t.Fatalf("phase %v", p.State().Phase)
	}
}

func TestEffectOrder(t *testing.T) {
	p := started(t)
	eff := p.Step(Tick{Now: t0.Add(10 * time.Minute)})
	want := []string{"CloseNotice", "StartBreak", "PlaySound", "Arm", "TrayDirty"}
	if len(eff) != len(want) {
		t.Fatalf("effects %v", eff)
	}
	for i, name := range want {
		got := effectName(eff[i])
		if got != name {
			t.Errorf("effect %d is %s, want %s", i, got, name)
		}
	}
}

func effectName(e Effect) string {
	switch e.(type) {
	case CloseNotice:
		return "CloseNotice"
	case StartBreak:
		return "StartBreak"
	case PlaySound:
		return "PlaySound"
	case Arm:
		return "Arm"
	case Disarm:
		return "Disarm"
	case TrayDirty:
		return "TrayDirty"
	case Notify:
		return "Notify"
	case CloseBreak:
		return "CloseBreak"
	case NotifyResume:
		return "NotifyResume"
	default:
		return "unknown"
	}
}

// TestRandomInputsKeepsInvariants drives the planner with random inputs and
// checks the state invariants after every step.
func TestRandomInputsKeepsInvariants(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	p := started(t)
	now := t0
	for range 3000 {
		now = now.Add(time.Duration(r.IntN(90)) * time.Second)
		switch r.IntN(9) {
		case 0:
			p.Step(Tick{Now: now})
		case 1:
			p.Step(Wake{Now: now})
		case 2:
			p.Step(Pause{Now: now, Reason: Reason(1 << r.IntN(5))})
		case 3:
			p.Step(Resume{Now: now, Reason: Reason(1 << r.IntN(5))})
		case 4:
			p.Step(Postpone{Now: now})
		case 5:
			p.Step(Skip{Now: now})
		case 6:
			p.Step(Finish{Now: now})
		case 7:
			if r.IntN(4) == 0 {
				p.Step(Reset{Now: now})
			}
		case 8:
			p.Step(SkipTo{Now: now, Kind: Kind(r.IntN(2) + 1)})
		}
		if err := p.State().validate(); err != nil {
			t.Fatalf("at %v: %v (%+v)", now, err, p.State())
		}
	}
}

func TestRestoreResumesTheSavedSchedule(t *testing.T) {
	p := New(defaultConfig())
	now := t0.Add(time.Minute)
	breakAt := now.Add(7 * time.Minute)
	notifyAt := breakAt.Add(-10 * time.Second)
	eff := p.Step(Restore{Now: now, N: 3, BreakAt: breakAt, NotifyAt: notifyAt})
	arms := effects[Arm](eff)
	if len(arms) != 1 || !arms[0].At.Equal(notifyAt) {
		t.Fatalf("restore effects: %#v", eff)
	}
	st := p.State()
	if st.N != 3 || st.Kind != p.cfg.KindFor(3) || !st.BreakAt.Equal(breakAt) || !st.NotifyAt.Equal(notifyAt) {
		t.Errorf("restored state: %+v", st)
	}
	// A notice whose time already passed is dropped.
	p2 := New(defaultConfig())
	future := now.Add(3 * time.Minute)
	p2.Step(Restore{Now: now, N: 1, BreakAt: future, NotifyAt: now.Add(-time.Second)})
	if got := p2.State().NotifyAt; !got.IsZero() {
		t.Errorf("stale notice kept: %v", got)
	}
	// A break in the past is not resumed.
	p3 := New(defaultConfig())
	p3.Step(Restore{Now: now, N: 2, BreakAt: now.Add(-time.Second)})
	if got := p3.State(); got.N != 0 || !got.BreakAt.IsZero() {
		t.Errorf("past break restored: %+v", got)
	}
}

func TestAdvanceStartsTheNextBreakNow(t *testing.T) {
	p := started(t)
	now := p.State().BreakAt.Add(-time.Minute)
	eff := p.Step(Advance{Now: now})
	breaks := effects[StartBreak](eff)
	if len(breaks) != 1 || !breaks[0].Spec.StartAt.Equal(now) {
		t.Fatalf("advance effects: %v", eff)
	}
	if st := p.State(); st.Phase != Breaking || st.N != 1 || st.Kind != Mini {
		t.Errorf("after advance: %+v", st)
	}
}

func TestDeferPushesTheNextBreak(t *testing.T) {
	p := started(t)
	now := p.State().BreakAt.Add(-time.Minute)
	want := p.State().BreakAt.Add(5 * time.Minute)
	eff := p.Step(Defer{Now: now, By: 5 * time.Minute})
	arms := effects[Arm](eff)
	if len(arms) != 1 || !arms[0].At.Equal(want.Add(-10*time.Second)) {
		t.Fatalf("defer effects: %v", eff)
	}
	if st := p.State(); !st.BreakAt.Equal(want) || st.N != 1 || st.Kind != Mini {
		t.Errorf("after defer: %+v", st)
	}
}
