package app

import (
	"testing"
	"time"

	"github.com/gythialy/fermata/internal/planner"
)

func TestRemainingLabel(t *testing.T) {
	now := time.Date(2026, 11, 26, 9, 0, 0, 0, time.Local)
	cases := []struct {
		name string
		st   planner.State
		want string
	}{
		{"waiting rounds up", planner.State{Phase: planner.Waiting, BreakAt: now.Add(20*time.Minute + 25*time.Second)}, "21m"},
		{"waiting never goes negative", planner.State{Phase: planner.Waiting, BreakAt: now.Add(-time.Minute)}, "0m"},
		{"the notice counts seconds", planner.State{Phase: planner.Notified, BreakAt: now.Add(8 * time.Second)}, "8s"},
		{"a break counts down", planner.State{Phase: planner.Breaking, EndAt: now.Add(4*time.Minute + 12*time.Second)}, "4:12"},
		{"a pause without an end", planner.State{Phase: planner.Paused}, "—"},
		{"a timed pause", planner.State{Phase: planner.Paused, PauseUntil: now.Add(28*time.Minute + 12*time.Second)}, "28:12"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := remainingLabel(tc.st, now); got != tc.want {
				t.Errorf("remainingLabel = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestPopoverState(t *testing.T) {
	cases := []struct {
		name string
		st   planner.State
		want string
	}{
		{"waiting for a mini break", planner.State{Phase: planner.Waiting, Kind: planner.Mini}, "Next mini break"},
		{"waiting for a long break", planner.State{Phase: planner.Waiting, Kind: planner.Long}, "Next long break"},
		{"the notice", planner.State{Phase: planner.Notified, Kind: planner.Mini}, "Mini break · starting"},
		{"a break in progress", planner.State{Phase: planner.Breaking, Kind: planner.Mini}, "Mini break · in progress"},
		{"a long break in progress", planner.State{Phase: planner.Breaking, Kind: planner.Long}, "Long break · in progress"},
		{"a strict break", planner.State{Phase: planner.Breaking, Kind: planner.Mini, Strict: true}, "Strict · Mini break"},
		{"paused", planner.State{Phase: planner.Paused}, "Paused"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := popoverState(tc.st); got != tc.want {
				t.Errorf("popoverState = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestPopoverActionsFollowTheState(t *testing.T) {
	a := &App{}
	cases := []struct {
		name      string
		st        planner.State
		want      int
		wantFirst bool
	}{
		{"waiting offers two", planner.State{Phase: planner.Waiting}, 2, true},
		{"the notice offers three", planner.State{Phase: planner.Notified}, 3, true},
		{"pausing offers only resume", planner.State{Phase: planner.Paused}, 1, true},
		{"a break offers none", planner.State{Phase: planner.Breaking}, 0, false},
		{"strict offers none", planner.State{Phase: planner.Breaking, Strict: true}, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := a.popoverActions(tc.st)
			if len(got) != tc.want {
				t.Fatalf("got %d actions, want %d", len(got), tc.want)
			}
			if tc.want > 0 && got[0].Primary != tc.wantFirst {
				t.Errorf("the first action's primary = %v, want %v", got[0].Primary, tc.wantFirst)
			}
		})
	}
}
