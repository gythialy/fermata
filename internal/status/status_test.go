package status

import (
	"fmt"
	"testing"
	"time"

	"github.com/gythialy/fermata/internal/planner"
)

// fakeMessages marks each message with its key and arguments, so a test reads
// which message the status chose.
type fakeMessages struct{}

func (fakeMessages) Preparing() string               { return "preparing" }
func (fakeMessages) NextMini(n int) string           { return fmt.Sprintf("next-mini:%d", n) }
func (fakeMessages) NextLong(n int) string           { return fmt.Sprintf("next-long:%d", n) }
func (fakeMessages) AfterMini(n int) string          { return fmt.Sprintf("after-mini:%d", n) }
func (fakeMessages) SoonMini() string                { return "soon-mini" }
func (fakeMessages) SoonLong() string                { return "soon-long" }
func (fakeMessages) NotifiedMini(s int) string       { return fmt.Sprintf("notified-mini:%d", s) }
func (fakeMessages) NotifiedLong(s int) string       { return fmt.Sprintf("notified-long:%d", s) }
func (fakeMessages) BreakingMini(left string) string { return fmt.Sprintf("breaking-mini:%s", left) }
func (fakeMessages) BreakingLong(left string) string { return fmt.Sprintf("breaking-long:%s", left) }
func (fakeMessages) StrictMini(left string) string   { return fmt.Sprintf("strict-mini:%s", left) }
func (fakeMessages) StrictLong(left string) string   { return fmt.Sprintf("strict-long:%s", left) }
func (fakeMessages) BreakOver() string               { return "break-over" }
func (fakeMessages) PausedTimed(minutes int) string  { return fmt.Sprintf("paused-timed:%d", minutes) }
func (fakeMessages) PausedForever() string           { return "paused-forever" }
func (fakeMessages) PausedSkipPeriod() string        { return "paused-skip-period" }
func (fakeMessages) PausedDnd() string               { return "paused-dnd" }
func (fakeMessages) PausedIdle() string              { return "paused-idle" }
func (fakeMessages) PausedLocked() string            { return "paused-locked" }
func (fakeMessages) PausedExclusion() string         { return "paused-exclusion" }
func (fakeMessages) TooltipTitle() string            { return "tooltip-title" }

var now = time.Date(2026, 11, 25, 9, 0, 0, 0, time.UTC)

func TestFormat(t *testing.T) {
	tests := []struct {
		name   string
		state  planner.State
		extra  Extra
		status string
		detail string
	}{
		{
			name:   "preparing",
			state:  planner.State{Phase: planner.Preparing},
			status: "preparing",
		},
		{
			name:   "waiting for a mini break",
			state:  planner.State{Phase: planner.Waiting, Kind: planner.Mini, BreakAt: now.Add(9*time.Minute + 40*time.Second)},
			status: "next-mini:10",
		},
		{
			name:   "waiting for a long break",
			state:  planner.State{Phase: planner.Waiting, Kind: planner.Long, BreakAt: now.Add(4 * time.Minute)},
			status: "next-long:4",
		},
		{
			name:   "second line",
			state:  planner.State{Phase: planner.Waiting, Kind: planner.Mini, BreakAt: now.Add(9 * time.Minute)},
			extra:  Extra{LongAfterMinis: 2},
			status: "next-mini:9",
			detail: "after-mini:2",
		},
		{
			name:   "under a minute",
			state:  planner.State{Phase: planner.Waiting, Kind: planner.Mini, BreakAt: now.Add(37 * time.Second)},
			status: "notified-mini:37",
		},
		{
			name:   "due now",
			state:  planner.State{Phase: planner.Waiting, Kind: planner.Long, BreakAt: now},
			status: "soon-long",
		},
		{
			name:   "notified",
			state:  planner.State{Phase: planner.Notified, Kind: planner.Long, BreakAt: now.Add(30 * time.Second)},
			status: "notified-long:30",
		},
		{
			name:   "breaking",
			state:  planner.State{Phase: planner.Breaking, Kind: planner.Mini, EndAt: now.Add(14 * time.Second)},
			status: "breaking-mini:0:14",
		},
		{
			name:   "breaking long",
			state:  planner.State{Phase: planner.Breaking, Kind: planner.Long, EndAt: now.Add(4*time.Minute + 58*time.Second)},
			status: "breaking-long:4:58",
		},
		{
			name:   "strict",
			state:  planner.State{Phase: planner.Breaking, Kind: planner.Mini, Strict: true, EndAt: now.Add(14 * time.Second)},
			status: "strict-mini:0:14",
		},
		{
			name:   "manual finish",
			state:  planner.State{Phase: planner.ManualAwait, Kind: planner.Mini},
			status: "break-over",
		},
		{
			name:   "paused with a time",
			state:  planner.State{Phase: planner.Paused, Reasons: planner.Manual, PauseUntil: now.Add(12*time.Minute + 20*time.Second)},
			status: "paused-timed:12",
		},
		{
			name:   "paused forever",
			state:  planner.State{Phase: planner.Paused, Reasons: planner.Manual},
			status: "paused-forever",
		},
		{
			name:   "skip period",
			state:  planner.State{Phase: planner.Paused, Reasons: planner.SkipPeriod | planner.DND},
			status: "paused-skip-period",
		},
		{
			name:   "manual pause outranks the skip period",
			state:  planner.State{Phase: planner.Paused, Reasons: planner.Manual | planner.SkipPeriod},
			status: "paused-forever",
		},
		{
			name:   "dnd",
			state:  planner.State{Phase: planner.Paused, Reasons: planner.DND | planner.Idle},
			status: "paused-dnd",
		},
		{
			name:   "idle",
			state:  planner.State{Phase: planner.Paused, Reasons: planner.Idle},
			status: "paused-idle",
		},
		{
			name:   "locked",
			state:  planner.State{Phase: planner.Paused, Reasons: planner.Lock},
			status: "paused-locked",
		},
		{
			name:   "excluded",
			state:  planner.State{Phase: planner.Paused, Reasons: planner.Exclusion},
			status: "paused-exclusion",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Format(tc.state, now, tc.extra, fakeMessages{})
			if got.Status != tc.status {
				t.Errorf("status %q, want %q", got.Status, tc.status)
			}
			if got.Detail != tc.detail {
				t.Errorf("detail %q, want %q", got.Detail, tc.detail)
			}
			if got.Tooltip == "" {
				t.Error("no tooltip")
			}
		})
	}
}

func TestTooltipCarriesEverything(t *testing.T) {
	st := planner.State{Phase: planner.Waiting, Kind: planner.Mini, BreakAt: now.Add(9 * time.Minute)}
	got := Format(st, now, Extra{LongAfterMinis: 2}, fakeMessages{})
	want := "tooltip-title\n\nnext-mini:9\nafter-mini:2"
	if got.Tooltip != want {
		t.Errorf("tooltip %q, want %q", got.Tooltip, want)
	}
}

func TestDurationAndClock(t *testing.T) {
	if got := Duration(14 * time.Second); got != "0:14" {
		t.Errorf("duration %q", got)
	}
	if got := Duration(time.Hour + 2*time.Minute + 3*time.Second); got != "1:02:03" {
		t.Errorf("duration %q", got)
	}
	if got := Overshoot(7 * time.Second); got != "+0:07" {
		t.Errorf("overshoot %q", got)
	}
	at := time.Date(2026, 11, 25, 14, 32, 0, 0, time.UTC)
	if got := Clock(at, false); got != "14:32" {
		t.Errorf("clock %q", got)
	}
	if got := Clock(at, true); got != "2:32 PM" {
		t.Errorf("clock %q", got)
	}
}
