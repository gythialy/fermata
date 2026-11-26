// Package welcome is Fermata's first run: one page, one decision, and a real
// preview of the break window.
package welcome

import (
	"log/slog"
	"runtime"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/gythialy/fermata/internal/ideas"
	"github.com/gythialy/fermata/internal/settings"
	"github.com/gythialy/fermata/internal/shortcut"
	"github.com/gythialy/fermata/internal/theme"
	"github.com/gythialy/fermata/internal/units"
	"github.com/gythialy/fermata/locales"
)

// Options are the app's services for the welcome window.
type Options struct {
	// Settings returns the live settings.
	Settings func() *settings.Settings
	// Touched validates and saves a change.
	Touched func()
	// Start begins the schedule; openSettings also opens the settings window.
	Start func(openSettings bool)
	// Dark reports whether the app is in a dark appearance.
	Dark func() bool
	// Clock formats a time of day.
	Clock func(time.Time) string
	// Idea is the sentence the preview shows.
	Idea func() ideas.Idea
	Log  *slog.Logger
}

// Window is the welcome window.
type Window struct {
	opts      Options
	win       *mygo.Window
	starting  bool
	localeOff func()
}

// New returns the welcome window, not yet created.
func New(opts Options) *Window {
	if opts.Log == nil {
		opts.Log = slog.Default()
	}
	return &Window{opts: opts}
}

// Show creates and shows the window.
func (w *Window) Show() {
	if w.win != nil && !w.win.IsDestroyed() {
		w.win.Show()
		w.win.Focus()
		return
	}
	win := mygo.NewWindow(mygo.WindowOptions{
		Title:             locales.AppName(),
		Width:             800,
		Height:            540,
		TitleBarStyle:     mygo.TitleBarHidden,
		DisableResize:     true,
		DisableMinimize:   true,
		DisableMaximize:   true,
		DisableFullScreen: true,
		Hidden:            true,
		Content:           ui.View(w.view),
	})
	w.win = win
	w.localeOff = mygo.App.OnLocaleChanged(func(string) { win.SetTitle(locales.AppName()) })
	// Closing the window is the same decision as Start.
	win.OnClose(func(e *mygo.CloseEvent) {
		if !w.starting {
			w.finish(false)
		}
	})
	win.Show()
	win.Focus()
}

// Refresh draws the window again, after a language or settings change.
func (w *Window) Refresh() {
	if w.win != nil && !w.win.IsDestroyed() {
		w.win.Invalidate()
	}
}

// Close destroys the window.
func (w *Window) Close() {
	if w.localeOff != nil {
		w.localeOff()
		w.localeOff = nil
	}
	if w.win == nil {
		return
	}
	w.win.Close()
	w.win = nil
}

// finish records the one decision and lets the app act on it.
func (w *Window) finish(openSettings bool) {
	if w.starting {
		return
	}
	w.starting = true
	w.opts.Start(openSettings)
}

// view draws the welcome.
func (w *Window) view(c *ui.Context) {
	t := theme.Apply(c)
	s := w.opts.Settings()
	bg, fg := theme.Field(s.BreakWindow, t.Dark)
	ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
		w.preview(c, t, s, bg, fg)
		w.content(c, t, s)
	})
}

// preview shows the break field as it will be, cropped for the left side.
func (w *Window) preview(c *ui.Context, t *ui.Theme, s *settings.Settings, bg, fg ui.Color) {
	ui.Column(c).Width(320).FillHeight().Background(bg).Padding(26, 28, 28, 28).Children(func() {
		ui.Row(c).Justify(ui.SpaceBetween).Children(func() {
			ui.Text(c, strings.ToUpper(locales.BreakLabelMini())).FontSize(theme.Px(c, 8)).FixedLineHeight(theme.Px(c, 12)).
				FontWeight(500).LetterSpacing(theme.Px(c, 0.9)).TextColor(fg.Alpha(0.55))
			if s.BreakWindow.ShowCurrentTime {
				ui.Text(c, w.opts.Clock(time.Now())).Font(theme.Mono).
					FontSize(8).FixedLineHeight(theme.Px(c, 12)).TextColor(fg.Alpha(0.55))
			}
		})
		ui.Column(c).Grow(1).Center().Children(func() {
			previewMark(c, bg)
			ui.Text(c, w.opts.Idea().Text).Font(theme.Serif).
				FontSize(24).FixedLineHeight(theme.Px(c, 32)).FontWeight(300).TextAlign(ui.Center).
				MaxWidth(250).MaxLines(3).TextColor(fg).Margin(16, 0, 0, 0)
			track := ui.Box(c).Width(130).Height(1).Background(fg.Alpha(0.25)).Margin(20, 0, 0, 0)
			track.Children(func() {
				ui.Box(c).Width(39).Height(1).Background(fg.Alpha(0.85))
			})
			ui.Text(c, "0:14").Font(theme.Mono).FontSize(theme.Px(c, 8)).
				FixedLineHeight(12).TextColor(fg.Alpha(0.65)).Margin(8, 0, 0, 0)
		})
		ui.Row(c).Justify(ui.Center).Children(func() {
			ui.Text(c, locales.BreakPostpone()+"  "+shortcut.End()).FontSize(theme.Px(c, 10)).FixedLineHeight(theme.Px(c, 14)).
				TextColor(fg.Alpha(0.82))
		})
	})
}

// content is the right half: the product, the rhythm, and the decision.
func (w *Window) content(c *ui.Context, t *ui.Theme, s *settings.Settings) {
	ui.Column(c).Grow(1).FillHeight().Padding(30, 34, 34, 34).Children(func() {
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			previewMarkLarge(c)
			ui.Text(c, locales.AppName()).FontSize(theme.Px(c, 20)).FixedLineHeight(theme.Px(c, 26)).Bold().LetterSpacing(theme.Px(c, -0.2))
		})
		ui.Text(c, locales.AppTagline()).Font(theme.Serif).
			FontSize(14).FixedLineHeight(theme.Px(c, 21)).TextColor(t.TextMuted).Margin(10, 0, 0, 0)

		ui.Column(c).Margin(22, 0, 0, 0).Padding(10, 0, 0, 0).
			BorderWidth(1, 0, 0, 0).BorderColor(t.Border).Children(func() {
			w.rhythmRows(c, t, s)
		})

		ui.Row(c).MinHeight(30).Gap(10).AlignItems(ui.Center).Margin(16, 0, 0, 0).Children(func() {
			ui.Text(c, locales.SettingsLanguage()).FontSize(theme.Px(c, 12)).FixedLineHeight(theme.Px(c, 18)).
				TextColor(t.TextMuted).Grow(1)
			tags := []string{settings.LanguageSystem, settings.LanguageEnglish, settings.LanguageChinese}
			labels := []string{locales.LanguageSystem(), locales.LanguageEnglish(), locales.LanguageChineseSimplified()}
			index := 0
			for i, tag := range tags {
				if tag == s.General.Language {
					index = i
				}
			}
			pick := labels[index]
			sel := ui.Select(c, &pick, labels)
			if sel.Changed() {
				for i, label := range labels {
					if label == pick {
						s.General.Language = tags[i]
						w.opts.Touched()
					}
				}
			}
		})

		ui.Row(c).MinHeight(30).Gap(10).AlignItems(ui.Center).Margin(8, 0, 0, 0).Children(func() {
			ui.Text(c, locales.SettingsOpenAtLogin()).FontSize(theme.Px(c, 12)).FixedLineHeight(theme.Px(c, 18)).
				TextColor(t.TextMuted).Grow(1)
			on := s.General.OpenAtLogin
			sw := ui.Switch(c, &on).Label(locales.SettingsOpenAtLogin())
			if sw.Changed() {
				s.General.OpenAtLogin = on
				w.opts.Touched()
			}
		})

		ui.Spacer(c)

		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			if ui.PrimaryButton(c, locales.WelcomeStart()).Clicked() {
				w.finish(false)
			}
			if textButton(c, t, locales.WelcomeOpenSettings()).Clicked() {
				w.finish(true)
			}
			ui.Text(c, locales.WelcomeClosingHint()).FontSize(theme.Px(c, 10)).FixedLineHeight(theme.Px(c, 15)).
				TextColor(t.TextMuted).MarginStart(ui.Auto)
		})
		ui.Text(c, w.livesText()).FontSize(theme.Px(c, 10)).FixedLineHeight(theme.Px(c, 15)).
			TextColor(t.TextMuted).Margin(12, 0, 0, 0)
	})
}

// livesText says where the app lives, with each platform's own name for the
// place: the menu bar on macOS, the system tray on Windows and Linux.
func (w *Window) livesText() string {
	if runtime.GOOS == "darwin" {
		return locales.WelcomeLivesInMenuBar()
	}
	return locales.WelcomeLivesInTray()
}

// rhythmRows shows the two rhythms the settings make.
func (w *Window) rhythmRows(c *ui.Context, t *ui.Theme, s *settings.Settings) {
	rows := w.rhythm(s)
	for i, row := range rows {
		ui.Row(c).Justify(ui.SpaceBetween).AlignItems(ui.Baseline).Padding(6, 0).Children(func() {
			if i > 0 {
				// A hairline above each row after the first.
				_ = i
			}
			ui.Text(c, row.label).FontSize(theme.Px(c, 11)).FixedLineHeight(theme.Px(c, 16)).TextColor(t.TextMuted)
			ui.Text(c, row.value).FontSize(theme.Px(c, 11)).FixedLineHeight(theme.Px(c, 16)).TextColor(t.Text).
				FontFeatures("tnum")
		})
	}
}

type rhythmRow struct{ label, value string }

func (w *Window) rhythm(s *settings.Settings) []rhythmRow {
	miniDur := units.Short(s.Schedule.Mini.DurationMs)
	interval := locales.UnitMinutes(s.Schedule.Mini.IntervalMs / 60_000)
	longDur := locales.UnitMinutes(s.Schedule.Long.DurationMs / 60_000)
	var rows []rhythmRow
	if s.Schedule.Mini.Enabled {
		rows = append(rows, rhythmRow{
			label: locales.WelcomeMiniBreak(),
			value: miniDur + " · " + locales.WelcomeEvery(interval),
		})
	}
	switch {
	case s.Schedule.Long.Enabled && s.Schedule.Mini.Enabled:
		rows = append(rows, rhythmRow{
			label: locales.WelcomeLongBreak(),
			value: longDur + " · " + locales.WelcomeEveryNth(locales.Ordinal(s.Schedule.Long.EveryNBreaks+1)),
		})
	case s.Schedule.Long.Enabled:
		rows = append(rows, rhythmRow{
			label: locales.WelcomeLongBreak(),
			value: longDur + " · " + locales.WelcomeEvery(locales.UnitMinutes(s.Schedule.Long.IntervalMs/60_000)),
		})
	}
	return rows
}

// previewMark draws the fermata at 40 points, where the preview's smaller
// type still reads it as the same mark as the break window's.
func previewMark(c *ui.Context, bg ui.Color) {
	arc, dot := theme.Mark(bg)
	box := ui.Box(c).Size(44, 44)
	box.Draw(func(p *ui.Painter, r ui.Rect) {
		theme.PaintMark(p, r, arc, dot)
	})
}

// previewMarkLarge draws the brand row's mark in the app icon's colors, so
// that it stays visible in either appearance.
func previewMarkLarge(c *ui.Context) {
	_, mark, dot := theme.AppIcon(c.Theme().Dark)
	box := ui.Box(c).Size(30, 30)
	box.Draw(func(p *ui.Painter, r ui.Rect) {
		theme.PaintAppIconMark(p, r, mark, dot)
	})
}

// textButton draws a quiet text action.
func textButton(c *ui.Context, t *ui.Theme, label string) ui.Element {
	button := ui.ButtonBase(c).Padding(2, 6).Gap(0).Radius(6)
	button.Children(func() {
		text := ui.Text(c, label).FontSize(theme.Px(c, 12)).FixedLineHeight(theme.Px(c, 17)).TextColor(t.Accent)
		if button.Hovered() {
			text.Underline().UnderlineOffset(3)
		}
	})
	return button
}
