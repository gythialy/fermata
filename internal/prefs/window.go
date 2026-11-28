// Package prefs is Fermata's settings window: general, schedule, appearance
// and about, with the design system's rows and dividers.
package prefs

import (
	"log/slog"
	"runtime"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/gythialy/fermata/internal/settings"
	"github.com/gythialy/fermata/internal/theme"
	"github.com/gythialy/fermata/locales"
)

// The settings pages.
const (
	SectionGeneral    = "general"
	SectionSchedule   = "schedule"
	SectionAppearance = "appearance"
	SectionAbout      = "about"
)

// Options are the app's services for the settings window.
type Options struct {
	// Settings returns the live settings.
	Settings func() *settings.Settings
	// Touched validates and saves a change, and applies what it needs.
	Touched func()
	// Reset restores the defaults after the user confirmed it.
	Reset func()
	// PreviewSound plays a sound for the appearance page.
	PreviewSound func(name string)
	// Dark reports whether the app is in a dark appearance.
	Dark func() bool
	// Clock formats a time of day.
	Clock func(time.Time) string
	// Now returns the local time used by the skip period status.
	Now func() time.Time
	// Displays returns the connected displays, for the content choices.
	Displays func() []mygo.Display
	// FocusAvailable reports whether the system exposes Focus without extra
	// access, which decides whether the Focus row is shown at all.
	FocusAvailable func() bool
	// ProcessesReadable reports whether the system lists running processes,
	// which decides whether app exclusion rules are offered at all.
	ProcessesReadable func() bool
	Language          func(string)
	// Version returns the app's version and build.
	Version func() (version, build string)
	// Update returns a newer version and its page, or empty strings.
	Update func() (version, url string)
	// Website, Source and Issues are the about page's links.
	Website, Source, Issues string
	// PreviewText is the sentence the appearance preview shows.
	PreviewText func() string
	Log         *slog.Logger
}

// Window is the one settings window of the app.
type Window struct {
	texts     map[string]*string
	times     map[string]*time.Time
	opts      Options
	win       *mygo.Window
	section   string
	banner    banner
	error     string
	localeOff func()
}

type banner struct {
	text   string
	action string
	run    func()
}

// New returns the settings window, not yet created.
func New(opts Options) *Window {
	if opts.Log == nil {
		opts.Log = slog.Default()
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	return &Window{opts: opts, section: SectionGeneral}
}

// now returns the time the views use, so tests can hold it still.
func (w *Window) now() time.Time {
	if w.opts.Now == nil {
		return time.Now()
	}
	return w.opts.Now()
}

// Show opens the window, or brings the open one forward.
func (w *Window) Show() {
	if w.win == nil || w.win.IsDestroyed() {
		w.create()
		return
	}
	w.win.Show()
	w.win.Focus()
}

func (w *Window) create() {
	top := 0
	if runtime.GOOS == "darwin" {
		top = -1 // the title bar is hidden; the view leaves room for the lights
	}
	var style mygo.TitleBarStyle
	if top != 0 {
		style = mygo.TitleBarHidden
	}
	win := mygo.NewWindow(mygo.WindowOptions{
		Title:         locales.SettingsTitle(),
		Width:         800,
		Height:        720,
		MinWidth:      760,
		MinHeight:     560,
		TitleBarStyle: style,
		StateKey:      "preferences",
		Content:       ui.View(w.view),
	})
	w.win = win
	w.localeOff = mygo.App.OnLocaleChanged(func(string) {
		win.SetTitle(locales.SettingsTitle())
	})
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

// Refresh draws the window again, after a change it did not make itself.
func (w *Window) Refresh() {
	if w.win != nil && !w.win.IsDestroyed() {
		w.win.Invalidate()
	}
}

// SetBanner shows a one-line message with one action.
func (w *Window) SetBanner(text, action string, run func()) {
	w.banner = banner{text: text, action: action, run: run}
	w.Refresh()
}

// ClearBanner removes the message.
func (w *Window) ClearBanner() {
	w.banner = banner{}
	w.Refresh()
}

// setError shows a message under the control that failed.
func (w *Window) setError(text string) {
	w.error = text
	w.Refresh()
}

func (w *Window) clearError() {
	if w.error != "" {
		w.error = ""
	}
}

// view builds the window.
func (w *Window) view(c *ui.Context) {
	t := theme.Apply(c)
	ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
		w.sidebar(c, t)
		ui.Scroll(c).Grow(1).Children(func() {
			ui.Column(c).Padding(w.topInset(), 32, 24, 32).MaxWidth(592).Children(func() {
				if w.banner.text != "" {
					w.bannerView(c, t)
				}
				switch w.section {
				case SectionSchedule:
					w.schedulePage(c, t)
				case SectionAppearance:
					w.appearancePage(c, t)
				case SectionAbout:
					w.aboutPage(c, t)
				default:
					w.generalPage(c, t)
				}
			})
		})
	})
	// The title bar is hidden: a strip at the top drags the window, under
	// the window controls.
	width, _ := c.Size()
	bar := c.TitleBar()
	ui.Box(c).Absolute().Left(0).Top(0).Size(width, max(bar.Height, 32)).DragWindow()
}

// topInset clears the window's controls: the traffic lights on macOS, the
// title bar elsewhere.
func (w *Window) topInset() float32 {
	if runtime.GOOS == "darwin" {
		return 54
	}
	return 24
}

// sidebar builds the source list: the mark, the name and four pages, without
// icons.
func (w *Window) sidebar(c *ui.Context, t *ui.Theme) {
	bg := ui.Hex("#F0EEE6")
	if t.Dark {
		bg = ui.Hex("#171B18")
	}
	ui.Column(c).Width(208).FillHeight().Background(bg).
		BorderWidth(0, 1, 0, 0).BorderColor(t.Border).
		Padding(44, 16, 16, 20).Children(func() {
		ui.Row(c).Gap(7).AlignItems(ui.Center).Children(func() {
			_, mark, dot := theme.AppIcon(t.Dark)
			ui.Box(c).Size(22, 22).Draw(func(p *ui.Painter, r ui.Rect) {
				theme.PaintMark(p, r, mark, dot)
			})
			ui.Text(c, locales.AppName()).FontSize(theme.Px(c, 13)).FixedLineHeight(theme.Px(c, 18)).Bold().LetterSpacing(theme.Px(c, 0.06))
		}).Margin(0, 0, 16, 0)
		for _, page := range []string{SectionGeneral, SectionSchedule, SectionAppearance, SectionAbout} {
			w.navItem(c, t, page)
		}
	})
}

// navItem is one row of the source list.
func (w *Window) navItem(c *ui.Context, t *ui.Theme, page string) {
	selected := w.section == page
	row := ui.ButtonBase(c).Height(32).MinHeight(32).Padding(0, 10).Radius(6).Gap(0).Justify(ui.Start)
	if selected {
		row.Background(t.Surface).Shadow(0, 1, 1, 0, ui.RGBA(0, 0, 0, 0.04))
	} else if row.Hovered() {
		row.Background(t.Surface.Alpha(0.55))
	}
	row.Children(func() {
		label := ui.Text(c, pageName(page)).FontSize(theme.Px(c, 13)).FixedLineHeight(theme.Px(c, 19))
		if selected {
			label.Bold().TextColor(t.Text)
		} else {
			label.TextColor(t.TextMuted)
		}
	})
	if row.Clicked() {
		w.section = page
	}
}

func pageName(page string) string {
	switch page {
	case SectionSchedule:
		return locales.SettingsSchedule()
	case SectionAppearance:
		return locales.SettingsAppearance()
	case SectionAbout:
		return locales.SettingsAbout()
	default:
		return locales.SettingsGeneral()
	}
}

// bannerView shows the one-line message.
func (w *Window) bannerView(c *ui.Context, t *ui.Theme) {
	bg, border, text := ui.Hex("#F7E8C8"), ui.Hex("#E6CF9F"), ui.Hex("#4A3600")
	if t.Dark {
		bg, border, text = ui.Hex("#3A2E12"), ui.Hex("#5A4A22"), ui.Hex("#F0C36A")
	}
	ui.Row(c).Radius(8).Padding(11, 12).Gap(12).Background(bg).Border(1, border).
		AlignItems(ui.Center).Margin(0, 0, 16, 0).Children(func() {
		ui.Text(c, w.banner.text).FontSize(theme.Px(c, 12)).FixedLineHeight(theme.Px(c, 17)).TextColor(text).Grow(1)
		if w.banner.action != "" {
			action := textButton(c, t, w.banner.action, false)
			if w.banner.run != nil && action.Clicked() {
				w.banner.run()
			}
		}
	})
}

// pageHeader draws the page's title and subtitle.
func pageHeader(c *ui.Context, t *ui.Theme, title, subtitle string) {
	ui.Text(c, title).FontSize(theme.Px(c, 22)).FixedLineHeight(theme.Px(c, 28)).Bold().LetterSpacing(theme.Px(c, -0.22))
	if subtitle != "" {
		ui.Text(c, subtitle).FontSize(theme.Px(c, 12)).FixedLineHeight(theme.Px(c, 17)).TextColor(t.TextMuted).Margin(5, 0, 16, 0)
	} else {
		ui.Box(c).Height(16)
	}
}

// markIcon draws the moon at a small size, in the app icon's colors: ink
// on paper in the light appearance, paper on ink in the dark one.
func markIcon(c *ui.Context, size float32) {
	_, mark, dot := theme.AppIcon(c.Theme().Dark)
	box := ui.Box(c).Size(size, size)
	box.Draw(func(p *ui.Painter, r ui.Rect) {
		theme.PaintAppIconMark(p, r, mark, dot)
	})
}
