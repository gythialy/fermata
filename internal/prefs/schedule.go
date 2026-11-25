package prefs

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/gythialy/fermata/internal/icons"
	"github.com/gythialy/fermata/internal/settings"
	"github.com/gythialy/fermata/internal/theme"
	"github.com/gythialy/fermata/internal/units"
	"github.com/gythialy/fermata/locales"
)

// schedulePage draws the two rhythms, strict mode, the health switch and the
// summary.
func (w *Window) schedulePage(c *ui.Context, t *ui.Theme) {
	s := w.opts.Settings()
	pageHeader(c, t, locales.SettingsSchedule(), locales.SettingsScheduleSubtitle())

	ui.Row(c).Gap(24).AlignItems(ui.Start).Children(func() {
		ui.Column(c).Grow(1).Gap(0).Children(func() { w.breakColumn(c, t, s, true) })
		ui.Column(c).Grow(1).Gap(0).Children(func() { w.breakColumn(c, t, s, false) })
	})

	w.strictSection(c, t, s)
	w.healthRow(c, t, s)
	w.summary(c, t, s)

	if w.error != "" {
		ui.Text(c, w.error).FontSize(theme.Px(c, 12)).FixedLineHeight(theme.Px(c, 17)).TextColor(t.Danger).Margin(8, 0, 0, 0)
	}
}

// breakColumn draws the short or long breaks' settings.
func (w *Window) breakColumn(c *ui.Context, t *ui.Theme, s *settings.Settings, mini bool) {
	enabled := s.Schedule.Mini.Enabled
	label := locales.SettingsMiniBreaks()
	if !mini {
		enabled = s.Schedule.Long.Enabled
		label = locales.SettingsLongBreaks()
	}
	ui.Row(c).MinHeight(36).AlignItems(ui.Center).Children(func() {
		ui.Text(c, label).FontSize(theme.Px(c, 13)).FixedLineHeight(theme.Px(c, 19)).Bold().Grow(1)
		on := enabled
		sw := ui.Switch(c, &on)
		if sw.Changed() {
			w.setBreakEnabled(s, mini, on)
		}
	})

	if mini {
		w.durationRow(c, t, s, enabled)
		w.intervalRow(c, t, s, enabled)
		w.notifyRow(c, t, s, enabled, true)
		w.postponeRow(c, t, s, enabled, true)
		w.waitFinishRow(c, t, s, enabled, true)
		return
	}
	w.longDurationRow(c, t, s, enabled)
	w.longEveryRow(c, t, s, enabled)
	w.notifyRow(c, t, s, enabled, false)
	w.postponeRow(c, t, s, enabled, false)
	w.waitFinishRow(c, t, s, enabled, false)
}

// setBreakEnabled turns one rhythm on or off, keeping at least one on.
func (w *Window) setBreakEnabled(s *settings.Settings, mini, on bool) {
	if mini {
		if !on && !s.Schedule.Long.Enabled {
			w.setError(locales.SettingsKeepOne())
			return
		}
		s.Schedule.Mini.Enabled = on
	} else {
		if !on && !s.Schedule.Mini.Enabled {
			w.setError(locales.SettingsKeepOne())
			return
		}
		s.Schedule.Long.Enabled = on
	}
	w.clearError()
	w.opts.Touched()
}

func (w *Window) durationRow(c *ui.Context, t *ui.Theme, s *settings.Settings, enabled bool) {
	columnRow(c, t, enabled, locales.SettingsBreakFor(), "", func() {
		w.editableMinutes(c, t, "mini-duration", float64(s.Schedule.Mini.DurationMs)/60_000, 1, 10, 1,
			!enabled, func(v float64) {
				s.Schedule.Mini.DurationMs = int(v) * 60_000
				w.opts.Touched()
			})
	})
}

func (w *Window) intervalRow(c *ui.Context, t *ui.Theme, s *settings.Settings, enabled bool) {
	columnRow(c, t, enabled, locales.SettingsEvery(), "", func() {
		w.editableMinutes(c, t, "mini-interval", float64(s.Schedule.Mini.IntervalMs)/60_000, 5, 65, 5,
			!enabled, func(v float64) {
				s.Schedule.Mini.IntervalMs = int(v) * 60_000
				w.opts.Touched()
			})
	})
}

func (w *Window) longDurationRow(c *ui.Context, t *ui.Theme, s *settings.Settings, enabled bool) {
	columnRow(c, t, enabled, locales.SettingsBreakFor(), "", func() {
		w.editableMinutes(c, t, "long-duration", float64(s.Schedule.Long.DurationMs)/60_000, 1, 60, 1,
			!enabled, func(v float64) {
				s.Schedule.Long.DurationMs = int(v) * 60_000
				w.opts.Touched()
			})
	})
}

func (w *Window) longEveryRow(c *ui.Context, t *ui.Theme, s *settings.Settings, enabled bool) {
	if !s.Schedule.Mini.Enabled {
		// With short breaks off, the long break has its own wait.
		columnRow(c, t, enabled, locales.SettingsEvery(), "", func() {
			w.editableMinutes(c, t, "long-interval", float64(s.Schedule.Long.IntervalMs)/60_000, 10, 120, 10,
				!enabled, func(v float64) {
					s.Schedule.Long.IntervalMs = int(v) * 60_000
					w.opts.Touched()
				})
		})
		return
	}
	every := float64(s.Schedule.Long.EveryNBreaks)
	// "mini breaks" is the row's second line, not a suffix after the stepper:
	// a suffix would push the stepper left and break the column's right edge.
	columnRow(c, t, enabled, locales.SettingsEvery(), locales.SettingsEveryMiniBreaks(), func() {
		out, changed := stepper(c, t, every, 1, 30, 1, func(v float64) string {
			return locales.UnitCount(int(v))
		}, !enabled)
		if changed {
			s.Schedule.Long.EveryNBreaks = max(int(out), 1)
			w.opts.Touched()
		}
	})
}

// editableMinutes draws a stepper whose value is also a text field, in
// whole minutes. Typing a number applies it, clamped to lo..hi.
func (w *Window) editableMinutes(c *ui.Context, t *ui.Theme, id string, value, lo, hi, step float64, disabled bool, apply func(float64)) {
	if w.texts == nil {
		w.texts = map[string]*string{}
	}
	field, ok := w.texts[id]
	if !ok {
		if value < lo {
			value = lo
		}
		text := strconv.Itoa(int(value + 0.5))
		field = &text
		w.texts[id] = field
	}
	out := value
	box := ui.Row(c).Height(32).MinWidth(132).Radius(8).Border(1, t.Border).Background(t.Surface)
	if disabled {
		box.Disabled(true)
	}
	box.Children(func() {
		minus := arrowButton(c, t, icons.Minus, locales.ActionDecrease())
		minus.Disabled(disabled || value <= lo)
		if minus.Clicked() {
			out = max(value-step, lo)
			*field = strconv.Itoa(int(out))
			apply(out)
		}
		input := ui.TextInputBase(c, field).Width(44).Height(24).Radius(5).
			Padding(0, 4).Background(t.SurfaceHover).FontSize(theme.Px(c, 12)).
			TextAlign(ui.Center).Disabled(disabled)
		if input.Changed() {
			n, err := strconv.ParseFloat(strings.TrimSpace(*field), 64)
			if err == nil && n >= lo && n <= hi {
				out = n
				apply(out)
			}
		}
		parts := strings.Fields(locales.UnitMinutes(1))
		unit := parts[len(parts)-1]
		ui.Text(c, unit).Width(30).FontSize(theme.Px(c, 11)).FixedLineHeight(theme.Px(c, 16)).TextColor(t.TextMuted)
		plus := arrowButton(c, t, icons.Plus, locales.ActionIncrease())
		plus.Disabled(disabled || value >= hi)
		if plus.Clicked() {
			out = min(value+step, hi)
			*field = strconv.Itoa(int(out))
			apply(out)
		}
	})
}
func (w *Window) notifyRow(c *ui.Context, t *ui.Theme, s *settings.Settings, enabled, mini bool) {
	lead := s.Schedule.Mini.NotifyLeadMs
	notify := s.Schedule.Mini.Notify
	if !mini {
		lead = s.Schedule.Long.NotifyLeadMs
		notify = s.Schedule.Long.Notify
	}
	columnRow(c, t, enabled, locales.SettingsNotifyBefore(), locales.DurationSeconds(lead/1000), func() {
		on := notify
		sw := ui.Switch(c, &on)
		sw.Disabled(!enabled)
		if sw.Changed() {
			if mini {
				s.Schedule.Mini.Notify = on
			} else {
				s.Schedule.Long.Notify = on
			}
			w.opts.Touched()
		}
	})
}

func (w *Window) postponeRow(c *ui.Context, t *ui.Theme, s *settings.Settings, enabled, mini bool) {
	postpone := s.Schedule.Mini.Postpone
	strict := s.Schedule.Mini.Strict
	if !mini {
		postpone = s.Schedule.Long.Postpone
		strict = s.Schedule.Long.Strict
	}
	desc := locales.SettingsPostponeOnce(postpone.TimeMs / 60_000)
	if strict {
		// Strict mode is the absence of an exit, not an error: the switch
		// grays out and the row says why.
		desc = locales.SettingsStrictCannotPostpone()
	}
	columnRow(c, t, enabled, locales.SettingsAllowPostpone(), desc, func() {
		on := postpone.Enabled
		sw := ui.Switch(c, &on)
		sw.Disabled(!enabled || strict)
		if sw.Changed() {
			if mini {
				s.Schedule.Mini.Postpone.Enabled = on
			} else {
				s.Schedule.Long.Postpone.Enabled = on
			}
			w.opts.Touched()
		}
	})
}

func (w *Window) waitFinishRow(c *ui.Context, t *ui.Theme, s *settings.Settings, enabled, mini bool) {
	columnRow(c, t, enabled, locales.SettingsWaitFinish(), locales.SettingsManualFinish(), func() {
		on := s.Schedule.Mini.ManualFinish
		if !mini {
			on = s.Schedule.Long.ManualFinish
		}
		sw := ui.Switch(c, &on)
		sw.Disabled(!enabled)
		if sw.Changed() {
			if mini {
				s.Schedule.Mini.ManualFinish = on
			} else {
				s.Schedule.Long.ManualFinish = on
			}
			w.opts.Touched()
		}
	})
}

// strictSection draws strict mode: one help line and a switch per kind.
func (w *Window) strictSection(c *ui.Context, t *ui.Theme, s *settings.Settings) {
	ui.Column(c).Margin(18, 0, 0, 0).Padding(12, 0, 0, 0).
		BorderWidth(1, 0, 0, 0).BorderColor(t.Border).Children(func() {
		ui.Row(c).Gap(20).AlignItems(ui.Start).Children(func() {
			ui.Column(c).Grow(1).Children(func() {
				ui.Text(c, locales.SettingsStrictMode()).FontSize(theme.Px(c, 13)).FixedLineHeight(theme.Px(c, 19)).Bold()
				ui.Text(c, locales.SettingsStrictHelp()).FontSize(theme.Px(c, 10)).FixedLineHeight(theme.Px(c, 14)).
					TextColor(t.TextMuted).Margin(2, 0, 0, 0)
			})
			ui.Row(c).Gap(18).AlignItems(ui.Center).Children(func() {
				w.strictSwitch(c, t, s, true)
				w.strictSwitch(c, t, s, false)
			})
		})
	})
}

func (w *Window) strictSwitch(c *ui.Context, t *ui.Theme, s *settings.Settings, mini bool) {
	ui.Row(c).Gap(7).AlignItems(ui.Center).Children(func() {
		label := locales.ShortMini()
		on := s.Schedule.Mini.Strict
		if !mini {
			label = locales.ShortLong()
			on = s.Schedule.Long.Strict
		}
		ui.Text(c, label).FontSize(theme.Px(c, 11)).FixedLineHeight(theme.Px(c, 16)).TextColor(t.TextMuted)
		sw := ui.Switch(c, &on)
		if sw.Changed() {
			if mini {
				s.Schedule.Mini.Strict = on
			} else {
				s.Schedule.Long.Strict = on
			}
			w.opts.Touched()
		}
	})
}

// healthRow is the break health switch.
func (w *Window) healthRow(c *ui.Context, t *ui.Theme, s *settings.Settings) {
	ui.Column(c).Margin(10, 0, 0, 0).Padding(10, 0, 0, 0).
		BorderWidth(1, 0, 0, 0).BorderColor(t.Border).Children(func() {
		row(c, t, locales.SettingsHealth(), locales.SettingsHealthHelp(), func() {
			on := s.Health.Mode
			sw := ui.Switch(c, &on)
			if sw.Changed() {
				s.Health.Mode = on
				w.opts.Touched()
			}
		})
	})
}

// summary draws the rhythm in words and as chips.
func (w *Window) summary(c *ui.Context, t *ui.Theme, s *settings.Settings) {
	ui.Column(c).Margin(18, 0, 0, 0).Padding(12, 0, 0, 0).
		BorderWidth(1, 0, 0, 0).BorderColor(t.Border).Children(func() {
		ui.Row(c).Gap(20).AlignItems(ui.Center).Children(func() {
			ui.Text(c, summaryText(s)).FontSize(theme.Px(c, 11)).FixedLineHeight(theme.Px(c, 16)).TextColor(t.TextMuted).Grow(1)
			ui.Row(c).Gap(6).Children(func() {
				for _, chip := range rhythmChips(s) {
					chipBox := ui.Box(c).Radius(4).Padding(2, 6).
						Background(t.Success.Alpha(0.14)).Children(func() {
						ui.Text(c, chip).FontSize(theme.Px(c, 10)).FixedLineHeight(theme.Px(c, 14)).FontWeight(500).
							LetterSpacing(0.6).TextColor(t.Success)
					})
					_ = chipBox
				}
			})
		})
	})
}

// summaryText writes the rhythm the settings make.
func summaryText(s *settings.Settings) string {
	miniDur := units.Compact(s.Schedule.Mini.DurationMs)
	interval := locales.UnitMinutes(s.Schedule.Mini.IntervalMs / 60_000)
	longDur := locales.UnitMinutes(s.Schedule.Long.DurationMs / 60_000)
	switch {
	case !s.Schedule.Long.Enabled:
		return locales.SettingsSummaryMiniOnly(miniDur, interval)
	case !s.Schedule.Mini.Enabled:
		return locales.SettingsSummaryLongOnly(longDur, locales.UnitMinutes(s.Schedule.Long.IntervalMs/60_000))
	default:
		nth := locales.Ordinal(s.Schedule.Long.EveryNBreaks + 1)
		return locales.SettingsSummary(miniDur, interval, longDur, nth)
	}
}

// rhythmChips names one cycle of the rhythm.
func rhythmChips(s *settings.Settings) []string {
	miniDur := units.Compact(s.Schedule.Mini.DurationMs)
	longDur := locales.UnitMinutes(s.Schedule.Long.DurationMs / 60_000)
	if !s.Schedule.Mini.Enabled {
		return []string{fmt.Sprintf("%s %s", locales.RhythmLong(), longDur)}
	}
	if !s.Schedule.Long.Enabled {
		return []string{fmt.Sprintf("%s %s", locales.RhythmMini(), miniDur)}
	}
	n := min(max(s.Schedule.Long.EveryNBreaks, 1), 6)
	out := make([]string, 0, n+1)
	for range n {
		out = append(out, fmt.Sprintf("%s %s", locales.RhythmMini(), miniDur))
	}
	return append(out, fmt.Sprintf("%s %s", locales.RhythmLong(), longDur))
}
