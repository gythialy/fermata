package savedstate

import (
	"path/filepath"
	"testing"
	"time"
)

func TestRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "schedule.json")
	now := time.Date(2026, 11, 25, 10, 0, 0, 0, time.UTC)
	want := State{SavedAt: now, N: 2, BreakAt: now.Add(8 * time.Minute), NotifyAt: now.Add(8*time.Minute - 10*time.Second)}
	if err := Save(path, want); err != nil {
		t.Fatal(err)
	}
	got, ok := Load(path, now.Add(time.Minute))
	if !ok {
		t.Fatal("saved schedule was not loaded")
	}
	if got.N != want.N || !got.BreakAt.Equal(want.BreakAt) || !got.NotifyAt.Equal(want.NotifyAt) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestLoadRejects(t *testing.T) {
	now := time.Date(2026, 11, 25, 10, 0, 0, 0, time.UTC)
	good := State{SavedAt: now, N: 1, BreakAt: now.Add(5 * time.Minute)}
	tests := []struct {
		name string
		mut  func(*State)
	}{
		{"stale", func(s *State) { s.SavedAt = now.Add(-MaxAge - time.Second) }},
		{"past break", func(s *State) { s.BreakAt = now.Add(-time.Second) }},
		{"no number", func(s *State) { s.N = 0 }},
		{"future save", func(s *State) { s.SavedAt = now.Add(time.Minute) }},
		{"notice after break", func(s *State) { s.NotifyAt = s.BreakAt.Add(time.Second) }},
	}
	for _, tc := range tests {
		path := filepath.Join(t.TempDir(), "state.json")
		st := good
		tc.mut(&st)
		if err := Save(path, st); err != nil {
			t.Fatal(err)
		}
		if _, ok := Load(path, now); ok {
			t.Errorf("%s: unexpected resume", tc.name)
		}
	}
	if _, ok := Load(filepath.Join(t.TempDir(), "missing.json"), now); ok {
		t.Error("missing file resumed")
	}
}

func TestClear(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if err := Save(path, State{SavedAt: time.Now(), N: 1, BreakAt: time.Now().Add(time.Minute)}); err != nil {
		t.Fatal(err)
	}
	if err := Clear(path); err != nil {
		t.Fatal(err)
	}
	if _, ok := Load(path, time.Now()); ok {
		t.Error("cleared state resumed")
	}
	if err := Clear(path); err != nil {
		t.Fatalf("clearing a missing file: %v", err)
	}
}
