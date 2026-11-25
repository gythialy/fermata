package app

import (
	"testing"
	"time"

	"github.com/gythialy/fermata/internal/planner"
	"github.com/gythialy/fermata/internal/settings"
)

func TestHoldDecision(t *testing.T) {
	now := time.Date(2026, 11, 25, 10, 0, 0, 0, time.UTC)
	due := now.Add(-time.Second)
	soon := now.Add(time.Minute)
	on := settings.Presence{Fullscreen: true, WaitWhileTyping: true}
	off := settings.Presence{}

	tests := []struct {
		name       string
		phase      planner.Phase
		breakAt    time.Time
		notifyAt   time.Time
		presence   settings.Presence
		fullscreen bool
		typing     bool
		want       string
	}{
		{"waiting, not due", planner.Waiting, soon, soon, on, true, true, ""},
		{"paused", planner.Paused, due, due, on, true, true, ""},
		{"fullscreen at the notice", planner.Waiting, soon, due, on, true, false, holdFullscreen},
		{"fullscreen at the break", planner.Waiting, due, due, on, true, false, holdFullscreen},
		{"typing at the break", planner.Waiting, due, due, on, false, true, holdTyping},
		{"typing only waits for the break", planner.Waiting, soon, due, on, false, true, ""},
		{"fullscreen wins over typing", planner.Waiting, due, due, on, true, true, holdFullscreen},
		{"settings off", planner.Waiting, due, due, off, true, true, ""},
		{"notified, fullscreen", planner.Notified, due, due, on, true, false, holdFullscreen},
		{"notified, typing", planner.Notified, due, due, on, false, true, holdTyping},
	}
	for _, tc := range tests {
		got := holdDecision(tc.phase, tc.breakAt, tc.notifyAt, now, tc.presence, tc.fullscreen, tc.typing)
		if got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestHoldStill(t *testing.T) {
	on := settings.Presence{Fullscreen: true, WaitWhileTyping: true}
	if !holdStill(holdFullscreen, on, true, false) {
		t.Error("fullscreen should still hold")
	}
	if holdStill(holdFullscreen, on, false, false) {
		t.Error("fullscreen should not hold after leaving fullscreen")
	}
	if !holdStill(holdTyping, on, false, true) {
		t.Error("typing should still hold")
	}
	if holdStill(holdTyping, on, false, false) {
		t.Error("typing should not hold after a pause")
	}
	if holdStill(holdTyping, settings.Presence{}, false, true) {
		t.Error("typing should not hold when the setting is off")
	}
	if holdStill("", on, true, true) {
		t.Error("no reason should hold")
	}
}
