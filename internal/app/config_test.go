package app

import (
	"slices"
	"testing"
	"time"

	"github.com/gythialy/fermata/internal/planner"
	"github.com/gythialy/fermata/internal/settings"
	"github.com/gythialy/fermata/internal/skiptime"
)

func TestConfigFromSettings(t *testing.T) {
	s := settings.Default()
	s.Schedule.Mini.DurationMs = 30_000
	s.Schedule.Mini.IntervalMs = 900_000
	s.Schedule.Mini.NotifyLeadMs = 5_000
	s.Schedule.Mini.Strict = true
	s.Schedule.Mini.ManualFinish = true
	s.Schedule.Mini.Postpone = settings.Postpone{Enabled: false, TimeMs: 60_000, Limit: 2, Percent: 20}
	s.Schedule.Long.Enabled = false
	s.Schedule.Long.EveryNBreaks = 4
	s.Schedule.MorningHour = 7
	cfg := configFromSettings(&s)
	if got := cfg.Mini.Duration; got != 30*time.Second {
		t.Errorf("mini duration %v", got)
	}
	if got := cfg.Mini.Interval; got != 15*time.Minute {
		t.Errorf("mini interval %v", got)
	}
	if got := cfg.Mini.Lead; got != 5*time.Second {
		t.Errorf("lead %v", got)
	}
	if !cfg.Mini.Strict || !cfg.Mini.ManualFinish {
		t.Error("mini flags lost")
	}
	if cfg.Mini.Postpone.Enabled || cfg.Mini.Postpone.Limit != 2 || cfg.Mini.Postpone.Time != time.Minute {
		t.Errorf("postpone %+v", cfg.Mini.Postpone)
	}
	if cfg.Long.Enabled {
		t.Error("long breaks stayed on")
	}
	if cfg.EveryN() != 4 {
		t.Errorf("every n %d", cfg.EveryN())
	}
	if cfg.MorningHour != 7 {
		t.Errorf("morning hour %d", cfg.MorningHour)
	}
	if got := cfg.PauseDurations; len(got) != len(s.Schedule.PauseDurations) {
		t.Fatalf("pauses %v", got)
	}
}

func TestConfigFromSettingsSkipPeriods(t *testing.T) {
	s := settings.Default()
	s.Schedule.SkipPeriodsEnabled = true
	s.Schedule.SkipPeriods = []settings.SkipPeriod{
		{Enabled: true, Start: "09:00", End: "10:30", Days: []int{7, 1}},
		{Enabled: true, Start: "22:30", End: "06:00"},
		{Start: "bad", End: "10:00"},
		{Start: "11:00", End: "12:00", Days: []int{0, 8}},
	}
	cfg := configFromSettings(&s)
	if !cfg.SkipPeriodsEnabled {
		t.Error("master switch lost")
	}
	want := []skiptime.Period{
		{StartMin: 9 * 60, EndMin: 10*60 + 30, Days: 1 | 1<<6},
		{StartMin: 22*60 + 30, EndMin: 6 * 60, Days: skiptime.AllDays},
	}
	if !slices.Equal(cfg.SkipPeriods, want) {
		t.Errorf("periods %+v, want %+v", cfg.SkipPeriods, want)
	}

	s.Schedule.SkipPeriodsEnabled = false
	if off := configFromSettings(&s); off.SkipPeriodsEnabled {
		t.Error("master switch stayed on")
	}
}

func TestConfigFromSettingsSkipPeriodsHonorEnabled(t *testing.T) {
	s := settings.Default()
	s.Schedule.SkipPeriodsEnabled = true
	s.Schedule.SkipPeriods = []settings.SkipPeriod{
		{Enabled: false, Start: "09:00", End: "10:00", Days: []int{1}},
		{Enabled: true, Start: "12:00", End: "13:00", Days: []int{2}},
		{Enabled: false, Start: "14:00", End: "15:00"},
		{Enabled: true, Start: "18:00", End: "19:30"},
	}
	cfg := configFromSettings(&s)
	want := []skiptime.Period{
		{StartMin: 12 * 60, EndMin: 13 * 60, Days: 1 << 1},
		{StartMin: 18 * 60, EndMin: 19*60 + 30, Days: skiptime.AllDays},
	}
	if !slices.Equal(cfg.SkipPeriods, want) {
		t.Errorf("periods %+v, want %+v", cfg.SkipPeriods, want)
	}
}

func TestLongAfterMinis(t *testing.T) {
	base := settings.Default()
	cfg := configFromSettings(&base)
	tests := []struct {
		n    int
		want int
	}{
		{1, 4}, {2, 3}, {3, 2}, {4, 1}, {5, 0}, {6, 4}, {0, 0},
	}
	for _, tc := range tests {
		st := planner.State{Phase: planner.Waiting, Kind: planner.Mini, N: tc.n}
		if got := longAfterMinis(cfg, st); got != tc.want {
			t.Errorf("n=%d: %d, want %d", tc.n, got, tc.want)
		}
	}
	longBase := settings.Default()
	longCfg := configFromSettings(&longBase)
	longCfg.EveryNBreaks = 1
	st := planner.State{Phase: planner.Waiting, Kind: planner.Mini, N: 1}
	if got := longAfterMinis(longCfg, st); got != 1 {
		t.Errorf("every other: %d", got)
	}
	oneBase := settings.Default()
	oneOff := configFromSettings(&oneBase)
	oneOff.Long.Enabled = false
	if got := longAfterMinis(oneOff, st); got != 0 {
		t.Errorf("with long breaks off: %d", got)
	}
}

func TestTrayTitle(t *testing.T) {
	now := time.Date(2026, 11, 25, 9, 0, 0, 0, time.UTC)
	waiting := planner.State{Phase: planner.Waiting, BreakAt: now.Add(9*time.Minute + 20*time.Second)}
	if got := trayTitle(waiting, now); got != "10m" {
		t.Errorf("waiting: %q", got)
	}
	notified := planner.State{Phase: planner.Notified, BreakAt: now.Add(10 * time.Second)}
	if got := trayTitle(notified, now); got != "0m" {
		t.Errorf("notified: %q", got)
	}
	paused := planner.State{Phase: planner.Paused, Reasons: planner.Manual}
	if got := trayTitle(paused, now); got != "" {
		t.Errorf("paused: %q", got)
	}
}

func TestHealthCost(t *testing.T) {
	if got := healthCost(planner.Mini); got != 1 {
		t.Errorf("mini %d", got)
	}
	if got := healthCost(planner.Long); got != 2 {
		t.Errorf("long %d", got)
	}
}

func TestStrictRunning(t *testing.T) {
	s := settings.Default()
	s.Schedule.Mini.Strict = true
	s.Schedule.Mini.ManualFinish = true
	cfg := configFromSettings(&s)
	p := planner.New(cfg)
	a := &App{pl: p}
	if a.strictRunning() {
		t.Error("strict before the first break")
	}
	now := time.Date(2026, 11, 25, 9, 0, 0, 0, time.UTC)
	p.Step(planner.Reset{Now: now})
	p.Step(planner.Tick{Now: p.State().BreakAt})
	if !a.strictRunning() {
		t.Error("strict while counting down")
	}
	end := p.State().EndAt
	p.Step(planner.Tick{Now: end})
	if !a.strictRunning() {
		t.Error("strict while waiting to finish")
	}
}
