package prefs

import (
	"github.com/egoist/mygo/ui"
	"github.com/gythialy/fermata/internal/icons"
	"github.com/gythialy/fermata/internal/theme"
	"github.com/gythialy/fermata/locales"
)

// row draws a label, an optional description under it, and a control at the
// right edge.
func row(c *ui.Context, t *ui.Theme, label, desc string, control func()) ui.Element {
	e := ui.Row(c).FillWidth().MinHeight(minRow).Gap(18).AlignItems(ui.Center)
	e.Children(func() {
		ui.Column(c).Grow(1).Children(func() {
			ui.Text(c, label).FontSize(theme.Px(c, 13)).FixedLineHeight(theme.Px(c, 19)).TextColor(t.Text)
			if desc != "" {
				ui.Text(c, desc).FontSize(theme.Px(c, 11)).FixedLineHeight(theme.Px(c, 16)).TextColor(t.TextMuted).Margin(2, 0, 0, 0)
			}
		})
		control()
	})
	return e
}

// subRow is the indented row under a switch it depends on.
func subRow(c *ui.Context, t *ui.Theme, label string, control func()) ui.Element {
	e := ui.Row(c).FillWidth().MinHeight(subRowHeight).Gap(18).AlignItems(ui.Center).PaddingStart(18)
	e.Children(func() {
		ui.Text(c, label).FontSize(theme.Px(c, 12)).FixedLineHeight(theme.Px(c, 17)).TextColor(t.TextMuted).Grow(1)
		control()
	})
	return e
}

// columnRow is a row inside the schedule page's two columns. A disabled
// column grays out without hiding.
func columnRow(c *ui.Context, t *ui.Theme, enabled bool, label, desc string, control func()) ui.Element {
	e := ui.Row(c).FillWidth().MinHeight(44).Gap(10).AlignItems(ui.Center).
		BorderWidth(0, 0, 1, 0).BorderColor(t.Border)
	if !enabled {
		e.Disabled(true)
	}
	text := t.Text
	if !enabled {
		text = t.TextMuted
	}
	e.Children(func() {
		ui.Column(c).Grow(1).Children(func() {
			ui.Text(c, label).FontSize(theme.Px(c, 13)).FixedLineHeight(theme.Px(c, 19)).TextColor(text)
			if desc != "" {
				ui.Text(c, desc).FontSize(theme.Px(c, 11)).FixedLineHeight(theme.Px(c, 16)).TextColor(t.TextMuted).Margin(2, 0, 0, 0)
			}
		})
		control()
	})
	return e
}

// divider draws the hairline between groups.
func divider(c *ui.Context, t *ui.Theme) {
	ui.Box(c).Height(1).Background(t.Border).Margin(15, 0, 16, 0)
}

const (
	minRow       = 44
	subRowHeight = 38
	controlWidth = 180
)

// stepper draws the design's stepper: two arrow buttons around the value,
// which the caller formats with its unit.
func stepper(c *ui.Context, t *ui.Theme, value float64, lo, hi, step float64, format func(float64) string, disabled bool) (float64, bool) {
	changed := false
	out := value
	box := ui.Row(c).Height(32).MinWidth(132).Radius(8).Border(1, t.Border).Background(t.Surface)
	box.Children(func() {
		minus := arrowButton(c, t, icons.Minus, locales.ActionDecrease())
		minus.Disabled(disabled || value <= lo)
		if minus.Clicked() {
			out = max(value-step, lo)
			changed = true
		}
		ui.Text(c, format(value)).Grow(1).MinWidth(52).FontSize(theme.Px(c, 13)).FixedLineHeight(theme.Px(c, 19)).
			TextAlign(ui.Center).TextColor(t.Text).FontFeatures("tnum")
		plus := arrowButton(c, t, icons.Plus, locales.ActionIncrease())
		plus.Disabled(disabled || value >= hi)
		if plus.Clicked() {
			out = min(value+step, hi)
			changed = true
		}
	})
	return out, changed
}

// listStepper steps through a fixed list of values, as the away threshold
// does.
func listStepper(c *ui.Context, t *ui.Theme, value float64, values []float64, format func(float64) string, disabled bool) (float64, bool) {
	index := 0
	for i, v := range values {
		if v <= value {
			index = i
		}
	}
	out := value
	changed := false
	box := ui.Row(c).Height(32).MinWidth(132).Radius(8).Border(1, t.Border).Background(t.Surface)
	if disabled {
		box.Disabled(true)
	}
	box.Children(func() {
		minus := arrowButton(c, t, icons.Minus, locales.ActionDecrease())
		minus.Disabled(disabled || index == 0)
		if minus.Clicked() {
			out = values[max(index-1, 0)]
			changed = true
		}
		ui.Text(c, format(values[index])).Grow(1).MinWidth(56).FontSize(theme.Px(c, 12)).FixedLineHeight(theme.Px(c, 18)).
			TextAlign(ui.Center).TextColor(t.Text).FontFeatures("tnum")
		plus := arrowButton(c, t, icons.Plus, locales.ActionIncrease())
		plus.Disabled(disabled || index == len(values)-1)
		if plus.Clicked() {
			out = values[min(index+1, len(values)-1)]
			changed = true
		}
	})
	return out, changed
}

// arrowButton is one button of a stepper. The glyph is drawn, not typed: a
// text minus and plus carry their own font's weight and sit on its baseline.
func arrowButton(c *ui.Context, t *ui.Theme, draw func(*ui.Painter, ui.Rect, ui.Color), desc string) ui.Element {
	button := ui.ButtonBase(c).Width(30).Height(30).Gap(0).Radius(6).Description(desc)
	if button.Hovered() {
		button.Background(t.SurfaceHover)
	}
	button.Children(func() {
		ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) {
			draw(p, icons.Box(r, 12), t.Text)
		})
	})
	return button
}

// switchControl is a plain switch bound to on, telling whether it changed.
func switchControl(c *ui.Context, on *bool) (ui.Element, bool) {
	sw := ui.Switch(c, on)
	return sw, sw.Changed()
}
