package prefs

import (
	"runtime"
	"strconv"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/gythialy/fermata/internal/settings"
	"github.com/gythialy/fermata/internal/theme"
	"github.com/gythialy/fermata/locales"
)

// generalPage draws what decides when the app goes quiet.
func (w *Window) generalPage(c *ui.Context, t *ui.Theme) {
	s := w.opts.Settings()
	pageHeader(c, t, locales.SettingsGeneral(), locales.SettingsGeneralSubtitle())

	row(c, t, locales.SettingsOpenAtLogin(), "", func() {
		on := s.General.OpenAtLogin
		sw := ui.Switch(c, &on)
		if sw.Changed() {
			s.General.OpenAtLogin = on
			w.opts.Touched()
		}
	})

	row(c, t, locales.SettingsLanguage(), "", func() {
		tags := []string{settings.LanguageSystem, settings.LanguageEnglish, settings.LanguageChinese}
		labels := []string{locales.LanguageSystem(), locales.LanguageEnglish(), locales.LanguageChineseSimplified()}
		if tag, changed := choiceSelect(c, s.General.Language, tags, labels); changed {
			s.General.Language = tag
			if w.opts.Language != nil {
				w.opts.Language(tag)
			}
			w.opts.Touched()
		}
	})

	divider(c, t)

	row(c, t, barStyleLabel(), "", func() {
		tags := []string{settings.StyleMark, settings.StyleTime, settings.StyleProgress}
		labels := []string{locales.StyleMark(), locales.StyleTime(), locales.StyleRing()}
		if runtime.GOOS == "windows" {
			// Windows tray icons have no title: the time style is not offered.
			tags = []string{settings.StyleMark, settings.StyleProgress}
			labels = []string{locales.StyleMark(), locales.StyleRing()}
		}
		idx := 0
		for i, tag := range tags {
			if tag == s.General.MenuBarStyle {
				idx = i
			}
		}
		seg := ui.Segmented(c, &idx, labels...)
		if seg.Changed() && idx < len(tags) {
			s.General.MenuBarStyle = tags[idx]
			w.opts.Touched()
		}
	}).MinHeight(52)

	divider(c, t)

	row(c, t, locales.SettingsPauseAway(), "", func() {
		on := s.Presence.Idle.Enabled
		sw := ui.Switch(c, &on)
		if sw.Changed() {
			s.Presence.Idle.Enabled = on
			w.opts.Touched()
		}
	})
	subRow(c, t, locales.SettingsAfterMinutes(s.Presence.Idle.ResetAfterMs/60_000), func() {
		minutes := float64(s.Presence.Idle.ResetAfterMs / 60_000)
		out, changed := listStepper(c, t, minutes, []float64{5, 10, 15, 30}, func(v float64) string {
			return locales.UnitMinutes(int(v))
		}, false)
		if changed {
			s.Presence.Idle.ResetAfterMs = int(out) * 60_000
			w.opts.Touched()
		}
	})

	if runtime.GOOS == "darwin" && w.focusAvailable() {
		row(c, t, locales.SettingsPauseFocus(), "", func() {
			on := s.Presence.DND.Enabled
			sw := ui.Switch(c, &on)
			if sw.Changed() {
				s.Presence.DND.Enabled = on
				w.opts.Touched()
			}
		})
	}

	row(c, t, locales.SettingsPauseLock(), "", func() {
		on := s.Presence.PauseOnLock
		sw := ui.Switch(c, &on)
		if sw.Changed() {
			s.Presence.PauseOnLock = on
			w.opts.Touched()
		}
	})

	row(c, t, locales.SettingsWaitTyping(), locales.SettingsWaitTypingHelp(), func() {
		on := s.Presence.WaitWhileTyping
		sw := ui.Switch(c, &on)
		if sw.Changed() {
			s.Presence.WaitWhileTyping = on
			w.opts.Touched()
		}
	})

	if runtime.GOOS == "darwin" {
		row(c, t, locales.SettingsWaitFullscreen(), locales.SettingsWaitFullscreenHelp(), func() {
			on := s.Presence.Fullscreen
			sw := ui.Switch(c, &on)
			if sw.Changed() {
				s.Presence.Fullscreen = on
				w.opts.Touched()
			}
		})
	}

	divider(c, t)

	row(c, t, locales.SettingsAllDisplays(), "", func() {
		on := s.BreakWindow.AllScreens
		sw := ui.Switch(c, &on)
		if sw.Changed() {
			s.BreakWindow.AllScreens = on
			w.opts.Touched()
		}
	})

	if s.BreakWindow.AllScreens {
		subRow(c, t, locales.SettingsContentOn(), func() {
			tags, labels := w.displayChoices(true)
			if tag, changed := choiceSelect(c, s.BreakWindow.ContentScreen, tags, labels); changed {
				s.BreakWindow.ContentScreen = tag
				w.opts.Touched()
			}
		})
	} else {
		subRow(c, t, locales.SettingsScreen(), func() {
			tags, labels := w.displayChoices(false)
			if tag, changed := choiceSelect(c, s.BreakWindow.Screen, tags, labels); changed {
				s.BreakWindow.Screen = tag
				w.opts.Touched()
			}
		})
	}

	row(c, t, locales.SettingsFillScreen(), "", func() {
		on := s.BreakWindow.FillScreen
		sw := ui.Switch(c, &on)
		if sw.Changed() {
			s.BreakWindow.FillScreen = on
			w.opts.Touched()
		}
	})

	w.exclusionsSection(c, t, s)
}

// focusAvailable reports whether the Focus row is offered.
func (w *Window) focusAvailable() bool {
	return w.opts.FocusAvailable != nil && w.opts.FocusAvailable()
}

func barStyleLabel() string {
	if runtime.GOOS == "windows" {
		return locales.SettingsTrayIcon()
	}
	return locales.SettingsMenuBar()
}

// choiceSelect draws a select and returns the chosen tag.
func choiceSelect(c *ui.Context, current string, tags, labels []string) (string, bool) {
	index := 0
	for i, tag := range tags {
		if tag == current {
			index = i
		}
	}
	if len(labels) == 0 {
		return current, false
	}
	pick := labels[index]
	sel := ui.Select(c, &pick, labels)
	if !sel.Changed() {
		return current, false
	}
	for i, label := range labels {
		if label == pick && i < len(tags) {
			return tags[i], true
		}
	}
	return current, false
}

// displayChoices names the displays the break window may target.
func (w *Window) displayChoices(includeAll bool) (tags, labels []string) {
	if includeAll {
		tags = append(tags, settings.ScreenAll)
		labels = append(labels, locales.DisplayAll())
	}
	tags = append(tags, settings.ScreenPrimary)
	labels = append(labels, locales.DisplayPrimary())
	tags = append(tags, settings.ScreenCursor)
	labels = append(labels, locales.DisplayCursor())
	for i := range w.displays() {
		tags = append(tags, strconv.Itoa(i))
		labels = append(labels, locales.DisplayNumbered(i+1))
	}
	return tags, labels
}

func (w *Window) displays() []mygo.Display {
	if w.opts.Displays == nil {
		return nil
	}
	return w.opts.Displays()
}

// exclusionsSection edits the app exclusion rules.
func (w *Window) exclusionsSection(c *ui.Context, t *ui.Theme, s *settings.Settings) {
	divider(c, t)
	ui.Row(c).Gap(18).AlignItems(ui.Center).Margin(8, 0, 0, 0).Children(func() {
		ui.Column(c).Grow(1).Children(func() {
			ui.Text(c, locales.SettingsExclusions()).FontSize(theme.Px(c, 13)).FixedLineHeight(theme.Px(c, 19)).Bold()
			ui.Text(c, locales.SettingsExclusionsHelp()).FontSize(theme.Px(c, 11)).FixedLineHeight(theme.Px(c, 16)).
				TextColor(t.TextMuted).Margin(2, 0, 0, 0)
		})
		add := textButton(c, t, locales.SettingsAddRule(), false)
		if add.Clicked() {
			s.Presence.AppExclusions = append(s.Presence.AppExclusions, settings.AppExclusion{
				Name:   "",
				Rule:   settings.RulePause,
				Active: true,
			})
			w.Refresh()
		}
	})

	rules := s.Presence.AppExclusions
	if len(rules) == 0 {
		ui.Text(c, locales.SettingsExclusionsEmpty()).FontSize(theme.Px(c, 11)).FixedLineHeight(theme.Px(c, 16)).
			TextColor(t.TextMuted).Margin(8, 0, 4, 0)
	}
	remove := -1
	for i := range rules {
		rule := &rules[i]
		ui.Column(c).Gap(6).Margin(8, 0, 8, 0).Children(func() {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				name := rule.Name
				ui.TextInput(c, &name).Placeholder(locales.SettingsRuleName()).Width(140).
					OnChange(func() {
						rule.Name = name
						w.opts.Touched()
					})
				processes := joinList(rule.Commands)
				ui.TextInput(c, &processes).Placeholder(locales.SettingsRuleProcessesHint()).Grow(1).
					OnChange(func() {
						rule.Commands = splitList(processes)
						w.opts.Touched()
					})
				if removeButton(c, t, locales.SettingsRuleRemove()).Clicked() {
					remove = i
				}
			})
			ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
				idx := 0
				if rule.Rule == settings.RuleResume {
					idx = 1
				}
				seg := ui.Segmented(c, &idx, locales.SettingsRulePause(), locales.SettingsRuleResume())
				if seg.Changed() {
					if idx == 1 {
						rule.Rule = settings.RuleResume
					} else {
						rule.Rule = settings.RulePause
					}
					w.opts.Touched()
				}
				on := rule.Active
				sw := ui.Switch(c, &on)
				if sw.Changed() {
					rule.Active = on
					w.opts.Touched()
				}
				ui.Text(c, locales.SettingsRuleActive()).FontSize(theme.Px(c, 11)).FixedLineHeight(theme.Px(c, 16)).TextColor(t.TextMuted)
				ui.Text(c, locales.SettingsRuleProcessesHint()).FontSize(theme.Px(c, 10)).FixedLineHeight(theme.Px(c, 14)).
					TextColor(t.TextMuted).Grow(1).TextAlign(ui.End)
			})
		})
	}
	if remove >= 0 {
		s.Presence.AppExclusions = append(s.Presence.AppExclusions[:remove], s.Presence.AppExclusions[remove+1:]...)
		w.opts.Touched()
	}
}

// textButton draws a quiet text action.
func textButton(c *ui.Context, t *ui.Theme, label string, danger bool) ui.Element {
	button := ui.ButtonBase(c).Padding(2, 6).Gap(0).Radius(6)
	color := t.Accent
	if danger {
		color = t.Danger
	}
	button.Children(func() {
		text := ui.Text(c, label).FontSize(theme.Px(c, 12)).FixedLineHeight(theme.Px(c, 17)).TextColor(color)
		if button.Hovered() {
			text.Underline().UnderlineOffset(3)
		}
	})
	return button
}

// removeButton is a text action at the end of a rule row.
func removeButton(c *ui.Context, t *ui.Theme, label string) ui.Element {
	return textButton(c, t, label, true)
}

// joinList and splitList convert between a slice of processes and the comma
// separated text the row edits.
func joinList(items []string) string {
	out := ""
	for i, item := range items {
		if i > 0 {
			out += ", "
		}
		out += item
	}
	return out
}

func splitList(text string) []string {
	var out []string
	for _, item := range strings.Split(text, ",") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}
