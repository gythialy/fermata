package settings

import (
	"bytes"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func load(t *testing.T, path string) *Store {
	t.Helper()
	st, warnings, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	for _, w := range warnings {
		t.Logf("warning: %s", w)
	}
	return st
}

func TestLoadWritesDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), FileName)
	st := load(t, path)
	if !st.Get().FirstRun {
		t.Error("a first run is not marked as one")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte(`"version": 2`)) || !bytes.Contains(data, []byte(`"isFirstRun": true`)) {
		t.Errorf("defaults not written:\n%s", data)
	}

	// A second load finds the file it wrote and needs no corrections.
	st2, warnings, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	defer st2.Close()
	if len(warnings) != 0 {
		t.Errorf("a written file needs corrections: %v", warnings)
	}
}

func TestUnknownFieldsSurviveUpdates(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	seed := `{"version":1,"isFirstRun":false,"general":{"language":"en"},"shiny":{"kept":true}}`
	if err := os.WriteFile(path, []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}
	st := load(t, path)
	st.Update(func(s *Settings) { s.General.MenuBarStyle = StyleTime })
	if err := st.Flush(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte(`"shiny"`)) || !bytes.Contains(data, []byte(`"kept"`)) {
		t.Errorf("unknown member lost:\n%s", data)
	}
	if !bytes.Contains(data, []byte(`"menuBarStyle": "time"`)) {
		t.Errorf("change not written:\n%s", data)
	}
}

func TestCorruptConfigIsBackedUp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	if err := os.WriteFile(path, []byte(`{"general":`), 0o644); err != nil {
		t.Fatal(err)
	}
	st, warnings, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if len(warnings) == 0 {
		t.Error("no warning for a corrupt file")
	}
	if st.Get().FirstRun {
		t.Error("a read error should not show the welcome window again")
	}
	backups, err := filepath.Glob(filepath.Join(dir, FileName+".corrupt.*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(backups) != 1 {
		t.Fatalf("backups: %v", backups)
	}
	again, _, err := Load(path)
	if err != nil {
		t.Fatalf("the rewritten file does not load: %v", err)
	}
	_ = again.Close()
}

func TestValidateRepairs(t *testing.T) {
	tests := []struct {
		name   string
		change func(*Settings)
		check  func(*Settings) error
	}{
		{"both off", func(s *Settings) { s.Schedule.Mini.Enabled, s.Schedule.Long.Enabled = false, false },
			func(s *Settings) error {
				if !s.Schedule.Mini.Enabled {
					return errors.New("mini breaks stayed off")
				}
				return nil
			}},
		{"mini duration", func(s *Settings) { s.Schedule.Mini.DurationMs = 1 },
			func(s *Settings) error {
				if s.Schedule.Mini.DurationMs != 10_000 {
					return errors.New("duration not clamped")
				}
				return nil
			}},
		{"every n", func(s *Settings) { s.Schedule.Long.EveryNBreaks = 99 },
			func(s *Settings) error {
				if s.Schedule.Long.EveryNBreaks != 30 {
					return errors.New("frequency not clamped")
				}
				return nil
			}},
		{"lead too long", func(s *Settings) { s.Schedule.Mini.NotifyLeadMs = 600_000 },
			func(s *Settings) error {
				if s.Schedule.Mini.NotifyLeadMs != 60_000 {
					return errors.New("lead not clamped")
				}
				return nil
			}},
		{"language", func(s *Settings) { s.General.Language = "fr" },
			func(s *Settings) error {
				if s.General.Language != LanguageSystem {
					return errors.New("language not reset")
				}
				return nil
			}},
		{"field", func(s *Settings) { s.BreakWindow.Field = "purple" },
			func(s *Settings) error {
				if s.BreakWindow.Field != FieldPine {
					return errors.New("field not reset")
				}
				return nil
			}},
		{"custom color", func(s *Settings) { s.BreakWindow.CustomField = "not a color" },
			func(s *Settings) error {
				if s.BreakWindow.CustomField != DefaultCustomField {
					return errors.New("color not reset")
				}
				return nil
			}},
		{"sounds", func(s *Settings) {
			s.Sound.MiniStart = "crickets"
			s.Sound.LongEnd = "crickets"
		}, func(s *Settings) error {
			if s.Sound.MiniStart != SoundSilence || s.Sound.LongEnd != SoundCrystalGlass {
				return errors.New("sounds not reset")
			}
			return nil
		}},
		{"volume", func(s *Settings) { s.Sound.Volume = 3 },
			func(s *Settings) error {
				if s.Sound.Volume != 1 {
					return errors.New("volume not clamped")
				}
				return nil
			}},
		{"pause durations", func(s *Settings) { s.Schedule.PauseDurations = nil },
			func(s *Settings) error {
				if len(s.Schedule.PauseDurations) == 0 {
					return errors.New("pause durations not restored")
				}
				return nil
			}},
		{"shortcut", func(s *Settings) { s.Shortcuts.EndBreak = "" },
			func(s *Settings) error {
				if s.Shortcuts.EndBreak == "" {
					return errors.New("shortcut not restored")
				}
				return nil
			}},
		{"screen", func(s *Settings) { s.BreakWindow.Screen = "third"; s.BreakWindow.ContentScreen = "third" },
			func(s *Settings) error {
				if s.BreakWindow.Screen != ScreenPrimary || s.BreakWindow.ContentScreen != ScreenAll {
					return errors.New("screens not reset")
				}
				return nil
			}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := Default()
			tc.change(&s)
			s.Validate()
			if err := tc.check(&s); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestValidateExclusions(t *testing.T) {
	s := Default()
	s.Presence.AppExclusions = []AppExclusion{
		{Name: "  ", Commands: []string{"x"}, Rule: "pause", Active: true},
		{Name: "Focus", Commands: []string{" ", "slack"}, Rule: "sometimes", Active: true},
	}
	s.Validate()
	if len(s.Presence.AppExclusions) != 1 {
		t.Fatalf("rules: %+v", s.Presence.AppExclusions)
	}
	rule := s.Presence.AppExclusions[0]
	if rule.Name != "Focus" || len(rule.Commands) != 1 || rule.Commands[0] != "slack" {
		t.Errorf("rule cleaned wrong: %+v", rule)
	}
	if rule.Rule != RulePause {
		t.Errorf("rule kind %q", rule.Rule)
	}
}

func TestNewerVersionIsReadNotMigrated(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	if err := os.WriteFile(path, []byte(`{"version":99,"isFirstRun":false}`), 0o644); err != nil {
		t.Fatal(err)
	}
	st, warnings, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if st.Get().Version != 99 {
		t.Errorf("version %d", st.Get().Version)
	}
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "newer") {
			found = true
		}
	}
	if !found {
		t.Errorf("warnings: %v", warnings)
	}
}

func TestWriteFailureIsReported(t *testing.T) {
	// A directory where the settings go: every system refuses to write that,
	// where a read-only directory only does on Unix.
	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	st := load(t, path)
	var reported error
	st.OnWriteError(func(err error) { reported = err })
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatal(err)
	}
	st.Update(func(s *Settings) { s.General.MenuBarStyle = StyleTime })
	if err := st.Flush(); err == nil {
		t.Error("flush succeeded although the settings path is a directory")
	}
	if reported == nil {
		t.Error("the write error listener did not hear about it")
	}
	if st.LastWriteError() == nil {
		t.Error("the write error was not kept")
	}
}

func TestFlushSavesTheLatestChange(t *testing.T) {
	path := filepath.Join(t.TempDir(), FileName)
	st := load(t, path)
	for i := range 10 {
		style := StyleTime
		if i%2 == 0 {
			style = StyleProgress
		}
		st.Update(func(s *Settings) { s.General.MenuBarStyle = style })
	}
	st.Update(func(s *Settings) {
		s.General.Language = LanguageEnglish
		s.General.MenuBarStyle = StyleProgress
	})
	if err := st.Flush(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte(`"language": "en"`)) || !bytes.Contains(data, []byte(`"menuBarStyle": "progress"`)) {
		t.Errorf("the latest change was not written:\n%s", data)
	}
}

func TestMissingDirectoryIsCreated(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "dir", FileName)
	st := load(t, path)
	st.Update(func(s *Settings) {
		s.General.Language = LanguageEnglish
		s.General.MenuBarStyle = StyleProgress
	})
	if err := st.Flush(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("%s was not written", path)
		}
		t.Fatal(err)
	}
}

func TestSkipPeriodDefaults(t *testing.T) {
	s := Default()
	if s.Version != 2 || s.Schedule.SkipPeriodsEnabled {
		t.Errorf("defaults: version %d, master switch %v", s.Version, s.Schedule.SkipPeriodsEnabled)
	}
	if len(s.Schedule.SkipPeriods) != 0 {
		t.Errorf("default skip periods: %+v", s.Schedule.SkipPeriods)
	}

	path := filepath.Join(t.TempDir(), FileName)
	load(t, path)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte(`"skipPeriodsEnabled": false`)) || !bytes.Contains(data, []byte(`"skipPeriods": []`)) {
		t.Errorf("skip period defaults not written:\n%s", data)
	}
}

func TestSkipPeriodsUseStableJSONKeys(t *testing.T) {
	data, err := json.Marshal(SkipPeriod{Enabled: true, Start: "09:00", End: "10:00", Days: []int{1}})
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatal(err)
	}
	if len(fields) != 4 {
		t.Fatalf("skip period keys: %s", data)
	}
	for _, key := range []string{"enabled", "start", "end", "days"} {
		if _, ok := fields[key]; !ok {
			t.Errorf("missing key %q in %s", key, data)
		}
	}
}

func TestSkipPeriodsKeepValidEntries(t *testing.T) {
	s := Default()
	s.Schedule.SkipPeriodsEnabled = true
	s.Schedule.SkipPeriods = []SkipPeriod{
		{Enabled: true, Start: "09:00", End: "12:30", Days: []int{1, 3, 5}},
		{Enabled: false, Start: "00:00", End: "23:59"},
	}
	if warnings := s.Validate(); len(warnings) != 0 {
		t.Errorf("valid entries corrected: %v", warnings)
	}
	got := s.Schedule.SkipPeriods
	if len(got) != 2 {
		t.Fatalf("entries: %+v", got)
	}
	if !got[0].Enabled || got[0].Start != "09:00" || got[0].End != "12:30" || !slices.Equal(got[0].Days, []int{1, 3, 5}) {
		t.Errorf("first entry changed: %+v", got[0])
	}
	if got[1].Start != "00:00" || got[1].End != "23:59" || len(got[1].Days) != 0 {
		t.Errorf("second entry changed: %+v", got[1])
	}
}

func TestSkipPeriodsDropInvalidTimes(t *testing.T) {
	tests := []struct {
		name  string
		start string
		end   string
	}{
		{"short hour", "9:00", "10:00"},
		{"hour out of range", "24:00", "23:59"},
		{"minute out of range", "09:60", "10:00"},
		{"leading space", " 09:00", "10:00"},
		{"seconds", "09:00:00", "10:00"},
		{"invalid end", "09:00", "25:00"},
		{"same start and end", "09:00", "09:00"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := Default()
			s.Schedule.SkipPeriods = []SkipPeriod{{Enabled: true, Start: tc.start, End: tc.end, Days: []int{1}}}
			warnings := s.Validate()
			if len(s.Schedule.SkipPeriods) != 0 {
				t.Errorf("invalid entry kept: %+v", s.Schedule.SkipPeriods)
			}
			if len(warnings) == 0 {
				t.Error("no note for the dropped entry")
			}
		})
	}
}

func TestSkipPeriodsDropInvalidDays(t *testing.T) {
	s := Default()
	s.Schedule.SkipPeriods = []SkipPeriod{
		{Start: "09:00", End: "10:00", Days: []int{-1, 0, 8, 2}},
		{Start: "10:00", End: "11:00", Days: []int{0, 9}},
	}
	s.Validate()
	got := s.Schedule.SkipPeriods
	if len(got) != 2 {
		t.Fatalf("entries: %+v", got)
	}
	if !slices.Equal(got[0].Days, []int{2}) {
		t.Errorf("weekdays not filtered: %v", got[0].Days)
	}
	if len(got[1].Days) != 0 {
		t.Errorf("no weekdays should mean every day: %v", got[1].Days)
	}
}

func TestSkipPeriodsLimit(t *testing.T) {
	s := Default()
	for i := range 6 {
		s.Schedule.SkipPeriods = append(s.Schedule.SkipPeriods, SkipPeriod{
			Start: fmt.Sprintf("%02d:00", i),
			End:   fmt.Sprintf("%02d:30", i),
		})
	}
	warnings := s.Validate()
	if len(s.Schedule.SkipPeriods) != 4 {
		t.Fatalf("entries: %+v", s.Schedule.SkipPeriods)
	}
	if got := s.Schedule.SkipPeriods[3].Start; got != "03:00" {
		t.Errorf("fifth entry replaced the fourth: %q", got)
	}
	if len(warnings) == 0 {
		t.Error("no note for the dropped entries")
	}
}

func TestSkipPeriodsSortAndDeduplicateDays(t *testing.T) {
	s := Default()
	s.Schedule.SkipPeriods = []SkipPeriod{{Start: "09:00", End: "10:00", Days: []int{7, 3, 3, 1, 7, 2}}}
	s.Validate()
	if len(s.Schedule.SkipPeriods) != 1 {
		t.Fatalf("entries: %+v", s.Schedule.SkipPeriods)
	}
	if got := s.Schedule.SkipPeriods[0].Days; !slices.Equal(got, []int{1, 2, 3, 7}) {
		t.Errorf("weekdays not sorted and deduplicated: %v", got)
	}
}

func TestSkipPeriodsKeepMasterSwitch(t *testing.T) {
	off := Default()
	off.Schedule.SkipPeriods = []SkipPeriod{{Start: "09:00", End: "10:00"}}
	off.Validate()
	if off.Schedule.SkipPeriodsEnabled {
		t.Error("the master switch was turned on")
	}

	on := Default()
	on.Schedule.SkipPeriodsEnabled = true
	on.Schedule.SkipPeriods = []SkipPeriod{{Start: "bad", End: "10:00"}}
	on.Validate()
	if !on.Schedule.SkipPeriodsEnabled {
		t.Error("the master switch was turned off")
	}
}
