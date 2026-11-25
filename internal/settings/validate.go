package settings

import (
	"fmt"
	"strconv"
	"strings"
)

// Validate clamps out-of-range values, fixes mutually exclusive choices and
// returns what it changed, for the log. It never rejects a configuration:
// the app starts with the values it can use.
func (s *Settings) Validate() []string {
	var w []string
	note := func(format string, args ...any) { w = append(w, fmt.Sprintf(format, args...)) }

	if s.Version < 1 {
		s.Version = CurrentVersion
	}
	if s.Version > CurrentVersion {
		note("config version %d is newer than %d; reading the known settings", s.Version, CurrentVersion)
	}

	// General.
	if !contains(Languages, s.General.Language) {
		note("general.language %q is not supported; using %q", s.General.Language, LanguageSystem)
		s.General.Language = LanguageSystem
	}
	if !contains([]string{StyleMark, StyleTime, StyleProgress}, s.General.MenuBarStyle) {
		note("general.menuBarStyle %q is not supported; using %q", s.General.MenuBarStyle, StyleMark)
		s.General.MenuBarStyle = StyleMark
	}

	// The rhythm.
	if !s.Schedule.Mini.Enabled && !s.Schedule.Long.Enabled {
		note("both break types were off; turning mini breaks on")
		s.Schedule.Mini.Enabled = true
	}
	m := &s.Schedule.Mini
	clampInt(&m.DurationMs, 10_000, 600_000, "schedule.mini.durationMs", note)
	clampInt(&m.IntervalMs, 300_000, 3_900_000, "schedule.mini.intervalMs", note)
	clampInt(&m.NotifyLeadMs, 1_000, 60_000, "schedule.mini.notifyLeadMs", note)
	clampPostpone(&m.Postpone, "schedule.mini.postpone", note)
	l := &s.Schedule.Long
	clampInt(&l.DurationMs, 60_000, 3_600_000, "schedule.long.durationMs", note)
	clampInt(&l.EveryNBreaks, 1, 30, "schedule.long.everyNBreaks", note)
	clampInt(&l.IntervalMs, 600_000, 7_200_000, "schedule.long.intervalMs", note)
	clampInt(&l.NotifyLeadMs, 1_000, 60_000, "schedule.long.notifyLeadMs", note)
	clampPostpone(&l.Postpone, "schedule.long.postpone", note)
	gap := m.IntervalMs
	if !m.Enabled && l.Enabled {
		gap = l.IntervalMs
	}
	if m.NotifyLeadMs >= gap {
		note("schedule.mini.notifyLeadMs %d does not fit the %d ms wait; using half", m.NotifyLeadMs, gap)
		m.NotifyLeadMs = gap / 2
	}
	if l.NotifyLeadMs >= gap {
		note("schedule.long.notifyLeadMs %d does not fit the %d ms wait; using half", l.NotifyLeadMs, gap)
		l.NotifyLeadMs = gap / 2
	}
	if len(s.Schedule.PauseDurations) == 0 {
		note("schedule.pauseDurationsMs was empty; using the defaults")
		s.Schedule.PauseDurations = Default().Schedule.PauseDurations
	}
	for i, d := range s.Schedule.PauseDurations {
		if d < 60_000 || d > 86_400_000 {
			note("schedule.pauseDurationsMs[%d] %d is out of range; clamping", i, d)
			s.Schedule.PauseDurations[i] = min(max(d, 60_000), 86_400_000)
		}
	}
	clampInt(&s.Schedule.MorningHour, 0, 23, "schedule.morningHour", note)

	// Presence.
	clampInt(&s.Presence.Idle.ResetAfterMs, 60_000, 3_600_000, "presence.idle.resetAfterMs", note)
	s.Presence.Idle.CheckIntervalMs = 2_000
	s.Presence.DND.CheckIntervalMs = 2_000
	s.Presence.AppExclusions = validateExclusions(s.Presence.AppExclusions, note)

	// The break window.
	bw := &s.BreakWindow
	bw.Screen = validScreen(bw.Screen, ScreenPrimary, note, "breakWindow.screen")
	bw.ContentScreen = validContentScreen(bw.ContentScreen, note)
	clampFloat(&bw.WidthRatio, 0.5, 1, "breakWindow.widthRatio", note)
	clampFloat(&bw.HeightRatio, 0.5, 1, "breakWindow.heightRatio", note)
	if bw.Field != FieldCustom && !contains(FieldPresets, bw.Field) {
		note("breakWindow.field %q is not a preset; using %q", bw.Field, FieldPine)
		bw.Field = FieldPine
	}
	if !validColor(bw.CustomField) {
		note("breakWindow.customField %q is not #rrggbb; using %q", bw.CustomField, DefaultCustomField)
		bw.CustomField = DefaultCustomField
	}
	clampFloat(&bw.Opacity, 0.3, 1, "breakWindow.opacity", note)

	// Sound.
	clampFloat(&s.Sound.Volume, 0, 1, "sound.volume", note)
	validateSound(&s.Sound.MiniStart, SoundSilence, "sound.miniStart", note)
	validateSound(&s.Sound.LongStart, SoundSilence, "sound.longStart", note)
	validateSound(&s.Sound.MiniEnd, SoundCrystalGlass, "sound.miniEnd", note)
	validateSound(&s.Sound.LongEnd, SoundCrystalGlass, "sound.longEnd", note)

	// Shortcuts.
	if strings.TrimSpace(s.Shortcuts.EndBreak) == "" {
		note("shortcuts.endBreak was empty; using CmdOrCtrl+X")
		s.Shortcuts.EndBreak = "CmdOrCtrl+X"
	}
	return w
}

func clampInt(v *int, lo, hi int, name string, note func(string, ...any)) {
	if *v < lo || *v > hi {
		note("%s %d is out of range %d–%d; clamping", name, *v, lo, hi)
		*v = min(max(*v, lo), hi)
	}
}

func clampFloat(v *float64, lo, hi float64, name string, note func(string, ...any)) {
	if *v < lo || *v > hi {
		note("%s %v is out of range %v–%v; clamping", name, *v, lo, hi)
		*v = min(max(*v, lo), hi)
	}
}

func clampPostpone(p *Postpone, name string, note func(string, ...any)) {
	clampInt(&p.TimeMs, 60_000, 3_600_000, name+".timeMs", note)
	clampInt(&p.Limit, 0, 5, name+".limit", note)
	clampInt(&p.Percent, 0, 100, name+".percent", note)
}

func validateExclusions(in []AppExclusion, note func(string, ...any)) []AppExclusion {
	out := in[:0]
	for _, rule := range in {
		rule.Name = strings.TrimSpace(rule.Name)
		var commands []string
		for _, c := range rule.Commands {
			if c = strings.TrimSpace(c); c != "" {
				commands = append(commands, c)
			}
		}
		rule.Commands = commands
		if rule.Rule != RulePause && rule.Rule != RuleResume {
			note("presence.appExclusions rule %q is not supported; using %q", rule.Rule, RulePause)
			rule.Rule = RulePause
		}
		if rule.Name == "" || len(commands) == 0 {
			note("dropping an app exclusion rule without a name or processes")
			continue
		}
		out = append(out, rule)
	}
	return out
}

func validateSound(v *string, fallback, name string, note func(string, ...any)) {
	if !contains(SoundNames, *v) {
		note("%s %q is not a sound; using %q", name, *v, fallback)
		*v = fallback
	}
}

func validScreen(v, fallback string, note func(string, ...any), name string) string {
	if v == ScreenPrimary || v == ScreenCursor {
		return v
	}
	if n, err := strconv.Atoi(v); err == nil && n >= 0 {
		return strconv.Itoa(n)
	}
	note("%s %q is not a display; using %q", name, v, fallback)
	return fallback
}

func validContentScreen(v string, note func(string, ...any)) string {
	if v == ScreenAll || v == ScreenPrimary || v == ScreenCursor {
		return v
	}
	if n, err := strconv.Atoi(v); err == nil && n >= 0 {
		return strconv.Itoa(n)
	}
	note("breakWindow.contentScreen %q is not a display; using %q", v, ScreenAll)
	return ScreenAll
}

func validColor(v string) bool {
	if len(v) != 7 || v[0] != '#' {
		return false
	}
	for _, c := range v[1:] {
		if !('0' <= c && c <= '9') && !('a' <= c && c <= 'f') && !('A' <= c && c <= 'F') {
			return false
		}
	}
	return true
}

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}
