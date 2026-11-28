package app

import (
	"time"

	"github.com/egoist/mygo"
	"github.com/gythialy/fermata/internal/cli"
	"github.com/gythialy/fermata/internal/planner"
	"github.com/gythialy/fermata/internal/settings"
	"github.com/gythialy/fermata/internal/skiptime"
	"github.com/gythialy/fermata/internal/status"
)

// configFromSettings turns the settings into the planner's configuration.
func configFromSettings(s *settings.Settings) planner.Config {
	breakConfig := func(ms int, intervalMs int, notify bool, leadMs int, p settings.Postpone, strict, manual bool) planner.BreakConfig {
		return planner.BreakConfig{
			Enabled:      true,
			Duration:     time.Duration(ms) * time.Millisecond,
			Interval:     time.Duration(intervalMs) * time.Millisecond,
			Notify:       notify,
			Lead:         time.Duration(leadMs) * time.Millisecond,
			Postpone:     planner.PostponeConfig{Enabled: p.Enabled, Time: time.Duration(p.TimeMs) * time.Millisecond, Limit: p.Limit, Percent: p.Percent},
			Strict:       strict,
			ManualFinish: manual,
		}
	}
	mini := breakConfig(s.Schedule.Mini.DurationMs, s.Schedule.Mini.IntervalMs, s.Schedule.Mini.Notify,
		s.Schedule.Mini.NotifyLeadMs, s.Schedule.Mini.Postpone, s.Schedule.Mini.Strict, s.Schedule.Mini.ManualFinish)
	mini.Enabled = s.Schedule.Mini.Enabled
	long := breakConfig(s.Schedule.Long.DurationMs, s.Schedule.Long.IntervalMs, s.Schedule.Long.Notify,
		s.Schedule.Long.NotifyLeadMs, s.Schedule.Long.Postpone, s.Schedule.Long.Strict, s.Schedule.Long.ManualFinish)
	long.Enabled = s.Schedule.Long.Enabled
	pauses := make([]time.Duration, 0, len(s.Schedule.PauseDurations))
	for _, ms := range s.Schedule.PauseDurations {
		pauses = append(pauses, time.Duration(ms)*time.Millisecond)
	}
	periods := make([]skiptime.Period, 0, len(s.Schedule.SkipPeriods))
	for _, p := range s.Schedule.SkipPeriods {
		if !p.Enabled {
			continue
		}
		start, ok := clockMinutes(p.Start)
		if !ok {
			continue
		}
		end, ok := clockMinutes(p.End)
		if !ok || start == end {
			continue
		}
		days := skiptime.AllDays
		if len(p.Days) > 0 {
			days = 0
			for _, d := range p.Days {
				if d >= 1 && d <= 7 {
					days |= 1 << (d - 1)
				}
			}
			if days == 0 {
				// Only invalid weekdays: settings validation drops these.
				continue
			}
		}
		periods = append(periods, skiptime.Period{StartMin: start, EndMin: end, Days: days})
	}
	return planner.Config{
		Mini:               mini,
		Long:               long,
		EveryNBreaks:       s.Schedule.Long.EveryNBreaks,
		PauseDurations:     pauses,
		MorningHour:        s.Schedule.MorningHour,
		SkipPeriodsEnabled: s.Schedule.SkipPeriodsEnabled,
		SkipPeriods:        periods,
	}
}

// clockMinutes reads a validated "HH:MM" clock as minutes from midnight.
func clockMinutes(v string) (int, bool) {
	if len(v) != 5 || v[2] != ':' {
		return 0, false
	}
	for _, i := range []int{0, 1, 3, 4} {
		if v[i] < '0' || v[i] > '9' {
			return 0, false
		}
	}
	hour := int(v[0]-'0')*10 + int(v[1]-'0')
	minute := int(v[3]-'0')*10 + int(v[4]-'0')
	if hour > 23 || minute > 59 {
		return 0, false
	}
	return hour*60 + minute, true
}

// pauseDurations are the menu's timed pauses.
func (a *App) pauseDurations() []time.Duration {
	s := a.store.Get()
	out := make([]time.Duration, 0, len(s.Schedule.PauseDurations))
	for _, ms := range s.Schedule.PauseDurations {
		out = append(out, time.Duration(ms)*time.Millisecond)
	}
	return out
}

// pauseFor pauses breaks, timed or indefinitely.
func (a *App) pauseFor(d time.Duration, indefinite bool) {
	now := time.Now()
	until := time.Time{}
	if !indefinite && d > 0 {
		until = now.Add(d)
	}
	a.step(planner.Pause{Now: now, Reason: planner.Manual, Until: until})
}

// pauseUntilMorning pauses until the next morning hour.
func (a *App) pauseUntilMorning() {
	now := time.Now()
	until := cli.UntilMorning(now, a.store.Get().Schedule.MorningHour)
	a.step(planner.Pause{Now: now, Reason: planner.Manual, Until: until})
}

// resume removes a manual pause.
func (a *App) resume() {
	a.step(planner.Resume{Now: time.Now(), Reason: planner.Manual})
}

// reset restarts the rhythm, keeping a pause.
func (a *App) reset() {
	a.danger = 0
	a.step(planner.Reset{Now: time.Now()})
}

// skipTo starts a break of the requested kind soon.
func (a *App) skipTo(kind planner.Kind) {
	cfg := a.pl.Config()
	if !cfg.For(kind).Enabled {
		return
	}
	a.step(planner.SkipTo{Now: time.Now(), Kind: kind})
	// A manual skip has no notice of its own: show the countdown and its
	// options right away.
	a.showPopover()
}

// clock writes the time of day as the region does.
func (a *App) clock(t time.Time) string {
	return status.Clock(t, mygo.App.Region().HourCycle == 12)
}

// applyTheme follows the theme setting: the system's, or a forced light or
// dark appearance for the app's windows.
func (a *App) applyTheme() {
	tag := a.store.Get().General.Theme
	mygo.Theme.SetSource(mygo.ThemeSource(tag))
}

// onConfigChanged applies a settings change to everything that follows it.
func (a *App) onConfigChanged() {
	if a.pl == nil {
		return
	}
	s := a.store.Get()
	a.applyLocale()
	a.applyTheme()
	a.applyLogin()
	a.configurePresence()
	if !s.Health.Mode {
		a.danger = 0
	}
	a.step(planner.SetConfig{Now: time.Now(), Config: configFromSettings(s)})
	if a.tray != nil {
		a.tray.Relabel(trayLabels())
	}
	if a.prefs != nil {
		a.prefs.Refresh()
	}
	if a.welcome != nil {
		a.welcome.Refresh()
	}
	a.refreshTray()
}
