package prefs

import (
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/gythialy/fermata/internal/settings"
	"github.com/gythialy/fermata/internal/shortcut"
	"github.com/gythialy/fermata/internal/theme"
	"github.com/gythialy/fermata/locales"
)

// appearancePage draws the preview, the break field and the sounds.
func (w *Window) appearancePage(c *ui.Context, t *ui.Theme) {
	s := w.opts.Settings()
	pageHeader(c, t, locales.SettingsAppearance(), locales.SettingsAppearanceSubtitle())

	row(c, t, locales.SettingsTheme(), locales.SettingsThemeHelp(), func() {
		tags := []string{settings.ThemeSystem, settings.ThemeLight, settings.ThemeDark}
		labels := []string{locales.ThemeSystem(), locales.ThemeLight(), locales.ThemeDark()}
		idx := 0
		for i, tag := range tags {
			if tag == s.General.Theme {
				idx = i
			}
		}
		seg := ui.Segmented(c, &idx, labels...)
		if seg.Changed() && idx < len(tags) {
			s.General.Theme = tags[idx]
			w.opts.Touched()
		}
	})

	divider(c, t)

	row(c, t, locales.SettingsOpacity(), locales.SettingsOpacityHelp(), func() {
		v := s.BreakWindow.Opacity
		sl := ui.Slider(c, &v, 0.3, 1)
		if sl.Changed() {
			s.BreakWindow.Opacity = v
			w.opts.Touched()
			w.Refresh()
		}
		ui.Text(c, fmt.Sprintf("%d%%", int(v*100+0.5))).Width(42).FontSize(theme.Px(c, 11)).
			FixedLineHeight(16).TextColor(t.TextMuted)
	})

	divider(c, t)

	w.preview(c, t, s)
	ui.Text(c, locales.SettingsPreviewScale()).FontSize(theme.Px(c, 10)).FixedLineHeight(theme.Px(c, 15)).
		TextColor(t.TextMuted).Margin(4, 0, 14, 0)

	ui.Row(c).Gap(16).AlignItems(ui.Start).Children(func() {
		ui.Column(c).Grow(1).Children(func() {
			ui.Text(c, strings.ToUpper(locales.SettingsBreakField())).FontSize(theme.Px(c, 11)).
				FixedLineHeight(theme.Px(c, 15)).FontWeight(600).LetterSpacing(theme.Px(c, 1.54)).TextColor(t.TextMuted)
			ui.Row(c).Gap(9).Margin(8, 0, 0, 0).Children(func() {
				for _, name := range settings.FieldPresets {
					w.swatch(c, t, s, name)
				}
				w.customSwatch(c, t, s)
			})
		})
	})

	divider(c, t)

	row(c, t, locales.SettingsShowTime(), "", func() {
		on := s.BreakWindow.ShowCurrentTime
		sw := ui.Switch(c, &on)
		if sw.Changed() {
			s.BreakWindow.ShowCurrentTime = on
			w.opts.Touched()
		}
	})

	divider(c, t)

	ui.Text(c, strings.ToUpper(locales.SettingsSounds())).FontSize(theme.Px(c, 11)).FixedLineHeight(theme.Px(c, 14)).
		FontWeight(600).LetterSpacing(theme.Px(c, 1.0)).TextColor(t.TextMuted).Margin(10, 0, 0, 4)

	muted := s.Sound.Muted
	row(c, t, locales.SettingsStartSound(), "", func() {
		soundSelect(c, w, s.Sound.MiniStart, muted, func(name string) {
			s.Sound.MiniStart = name
			s.Sound.LongStart = name
			w.opts.Touched()
		})
	})
	row(c, t, locales.SettingsEndSound(), "", func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			soundSelect(c, w, s.Sound.MiniEnd, muted, func(name string) {
				s.Sound.MiniEnd = name
				s.Sound.LongEnd = name
				w.opts.Touched()
			})
			preview := textButton(c, t, locales.SettingsPreviewSound(), false)
			preview.Disabled(muted)
			if preview.Clicked() {
				w.opts.PreviewSound(s.Sound.MiniEnd)
			}
		})
	})
	row(c, t, locales.SettingsVolume(), "", func() {
		volume := s.Sound.Volume
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			slider := ui.Slider(c, &volume, 0, 1).Width(controlWidth - 54)
			slider.Disabled(muted)
			if slider.Changed() {
				s.Sound.Volume = volume
				w.opts.Touched()
			}
			ui.Text(c, fmt.Sprintf("%d%%", int(volume*100+0.5))).Width(44).
				FontSize(12).FixedLineHeight(theme.Px(c, 18)).TextAlign(ui.End).
				TextColor(t.Text).FontFeatures("tnum")
		})
	})
	row(c, t, locales.SettingsMute(), "", func() {
		on := s.Sound.Muted
		sw := ui.Switch(c, &on)
		if sw.Changed() {
			s.Sound.Muted = on
			w.opts.Touched()
		}
	})
}

// preview shows the break field at about 0.43, as it will look. Its text keeps the
// design's own sizes: it is a picture of the break, not interface text, so the
// system's text scale does not reflow the miniature.
func (w *Window) preview(c *ui.Context, t *ui.Theme, s *settings.Settings) {
	bg, fg := theme.Field(s.BreakWindow, t.Dark)
	op := float32(s.BreakWindow.Opacity)
	if op <= 0 {
		op = 1
	}
	fade := func(col ui.Color, a float32) ui.Color { return col.Alpha(a * op) }
	box := ui.Box(c).Width(528).Height(216).Radius(12).Padding(18, 20, 20, 20)
	box.Draw(func(p *ui.Painter, r ui.Rect) {
		// A checkerboard shows how much of the desktop the field lets through.
		light, dark := t.Surface, t.Border
		const cell = 12
		for y := 0; float32(y) < r.H; y += cell {
			for x := 0; float32(x) < r.W; x += cell {
				col := light
				if (x/cell+y/cell)%2 == 0 {
					col = dark
				}
				p.Fill(ui.Rect{X: r.X + float32(x), Y: r.Y + float32(y), W: cell, H: cell}, col, 0)
			}
		}
		p.Fill(r, bg.Alpha(op), 12)
	})
	box.Children(func() {
		ui.Row(c).Justify(ui.SpaceBetween).Children(func() {
			ui.Text(c, strings.ToUpper(locales.BreakLabelMini())).FontSize(8).FixedLineHeight(11).
				FontWeight(500).LetterSpacing(theme.Px(c, 0.8)).TextColor(fade(fg, 0.55))
			if s.BreakWindow.ShowCurrentTime {
				ui.Text(c, w.opts.Clock(time.Now())).Font(theme.Mono).FontSize(8).FixedLineHeight(11).
					LetterSpacing(0.28).TextColor(fade(fg, 0.55))
			}
		})
		ui.Column(c).Grow(1).Center().Children(func() {
			arc, dot := theme.Mark(bg)
			previewMark(c, fade(arc, 1), fade(dot, 1))
			ui.Text(c, w.opts.PreviewText()).Font(theme.Serif).FontSize(20).FixedLineHeight(26).
				FontWeight(300).TextAlign(ui.Center).MaxWidth(410).MaxLines(2).
				TextColor(fade(fg, 1)).Margin(9, 0, 0, 0)
			track := ui.Box(c).Width(124).Height(1).Background(fade(fg, 0.25)).Margin(12, 0, 0, 0)
			track.Children(func() {
				ui.Box(c).Width(37).Height(1).Background(fade(fg, 0.85))
			})
			ui.Text(c, "0:14").Font(theme.Mono).FontSize(8).FixedLineHeight(11).
				LetterSpacing(0.28).TextColor(fade(fg, 0.65)).Margin(6, 0, 0, 0)
		})
		ui.Row(c).Justify(ui.Center).Children(func() {
			ui.Text(c, locales.BreakPostpone()+"  "+shortcut.End()).FontSize(10).FixedLineHeight(14).
				TextColor(fade(fg, 0.82))
		})
	})
}

// swatch draws one break field preset.
func (w *Window) swatch(c *ui.Context, t *ui.Theme, s *settings.Settings, name string) {
	selected := s.BreakWindow.Field == name
	bw := s.BreakWindow
	bw.Field = name
	col, _ := theme.Field(bw, t.Dark)
	ui.Column(c).Width(50).AlignItems(ui.Center).Children(func() {
		button := ui.ButtonBase(c).Size(42, 42).Gap(0).Radius(10)
		if selected {
			button.Border(2, t.Accent)
		}
		button.Children(func() {
			ui.Box(c).Size(34, 34).Radius(8).Background(col).Border(1, t.Border.Alpha(0.5))
		})
		if button.Clicked() {
			s.BreakWindow.Field = name
			w.opts.Touched()
		}
		ui.Text(c, fieldName(name)).FontSize(theme.Px(c, 10)).FixedLineHeight(theme.Px(c, 14)).TextColor(t.TextMuted).Margin(4, 0, 0, 0)
	})
}

// customSwatch draws the color well that makes a seventh field.
func (w *Window) customSwatch(c *ui.Context, t *ui.Theme, s *settings.Settings) {
	ui.Column(c).Width(64).AlignItems(ui.Center).Children(func() {
		color := parseColor(s.BreakWindow.CustomField)
		well := ui.ColorWell(c, &color).Size(42, 42)
		if s.BreakWindow.Field == settings.FieldCustom {
			well.Radius(10).Border(2, t.Accent)
		}
		if well.Changed() {
			s.BreakWindow.Field = settings.FieldCustom
			s.BreakWindow.CustomField = formatColor(color)
			w.opts.Touched()
		}
		ui.Text(c, locales.SettingsCustom()).FontSize(theme.Px(c, 10)).FixedLineHeight(theme.Px(c, 14)).
			TextColor(t.TextMuted).Margin(4, 0, 0, 0)
	})
}

// soundSelect draws a select of the built-in sounds.
func soundSelect(c *ui.Context, w *Window, current string, disabled bool, set func(string)) {
	pick := soundName(current)
	labels := make([]string, len(settings.SoundNames))
	for i, name := range settings.SoundNames {
		labels[i] = soundName(name)
	}
	sel := ui.Select(c, &pick, labels)
	sel.Disabled(disabled)
	if !sel.Changed() {
		return
	}
	for _, name := range settings.SoundNames {
		if soundName(name) == pick {
			set(name)
			return
		}
	}
}

func soundName(name string) string {
	switch name {
	case settings.SoundCrystalGlass:
		return locales.SoundCrystalGlass()
	case settings.SoundWindChime:
		return locales.SoundWindChime()
	case settings.SoundTicToc:
		return locales.SoundTicToc()
	case settings.SoundReverie:
		return locales.SoundReverie()
	default:
		return locales.SoundSilence()
	}
}

func fieldName(name string) string {
	switch name {
	case settings.FieldInk:
		return locales.FieldInk()
	case settings.FieldOxblood:
		return locales.FieldOxblood()
	case settings.FieldSlate:
		return locales.FieldSlate()
	case settings.FieldSand:
		return locales.FieldSand()
	case settings.FieldPaper:
		return locales.FieldPaper()
	default:
		return locales.FieldPine()
	}
}

// previewMark draws the fermata small, as the preview shows it.
func previewMark(c *ui.Context, mark, dot ui.Color) {
	box := ui.Box(c).Size(30, 30)
	box.Draw(func(p *ui.Painter, r ui.Rect) {
		theme.PaintMark(p, r, mark, dot)
	})
}

// parseColor and formatColor convert the settings' #rrggbb form.
func parseColor(hexValue string) ui.Color {
	if len(hexValue) == 7 && hexValue[0] == '#' {
		return ui.Hex(hexValue)
	}
	return ui.Hex(settings.DefaultCustomField)
}

func formatColor(c ui.Color) string {
	const digits = "0123456789abcdef"
	out := []byte{'#', 0, 0, 0, 0, 0, 0}
	for i, v := range []uint8{c.R, c.G, c.B} {
		out[1+2*i] = digits[v>>4]
		out[2+2*i] = digits[v&0xf]
	}
	return string(out)
}
