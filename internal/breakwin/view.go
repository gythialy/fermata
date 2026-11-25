package breakwin

import (
	"time"
	"unicode"

	"github.com/egoist/mygo/ui"
	"github.com/gythialy/fermata/internal/planner"
	"github.com/gythialy/fermata/internal/shortcut"
	"github.com/gythialy/fermata/internal/status"
	"github.com/gythialy/fermata/internal/theme"
)

// view draws one break window. It reads the manager's snapshot and never
// starts or ends a break itself.
func (m *Manager) view(c *ui.Context, content bool) {
	bg, fg := theme.Field(m.bw, c.Theme().Dark)
	long := m.spec.Kind == planner.Long
	if long {
		bg = theme.Deeper(m.bw.Field, bg)
	}
	root := ui.Box(c).Fill().Background(bg)
	if !m.bw.FillScreen {
		root.Radius(20)
	}
	if m.health > 2 && content && (m.phase == planner.Breaking || m.phase == planner.ManualAwait) {
		// The break health counter's warning: a vermilion edge, deeper as the
		// count grows, that changes no layout.
		strength := min(float32(m.health-2)/8, 1)
		_, dot := theme.Mark(bg)
		root.InsetShadow(0, 0, 24+36*strength, 4+10*strength, dot.Alpha(0.14+0.24*strength))
	}
	if !content {
		return
	}
	now := c.Now()
	// The content rises while the panel fades in; the window's own fade
	// lives in manager.go.
	raise := float32(0)
	if m.animate && !c.Preferences().ReduceMotion {
		if elapsed := now.Sub(m.shownAt); elapsed >= 0 && elapsed < contentRise {
			raise = riseAt(elapsed)
			c.After(16 * time.Millisecond)
		}
	}
	root.Children(func() {
		if m.bg != nil {
			ui.Image(c, m.bg).Absolute().Top(0).Left(0).Right(0).Bottom(0).Fit(ui.Cover)
			ui.Box(c).Absolute().Top(0).Left(0).Right(0).Bottom(0).
				Background(ui.RGBA(0, 0, 0, 0.6))
		}
		ui.Column(c).Fill().Padding(48, 48, 64, 48).Translate(0, raise).Children(func() {
			m.header(c, now, fg)
			ui.Column(c).Grow(1).Center().Children(func() {
				arc, dot := theme.Mark(bg)
				m.mark(c, arc, dot)
				m.suggestion(c, fg, long)
				m.progress(c, fg, now, long)
				m.countdown(c, fg, dot, now)
			})
			m.exit(c, now, fg)
		})
	})
	// Keep the countdown moving while the window is on screen.
	next := now.Truncate(time.Second).Add(time.Second)
	c.After(next.Sub(now))
}

// riseAt is how far the content sits below its resting place at elapsed,
// easing out to zero over contentRise.
func riseAt(elapsed time.Duration) float32 {
	if elapsed <= 0 {
		return 6
	}
	if elapsed >= contentRise {
		return 0
	}
	p := float32(elapsed) / float32(contentRise)
	return 6 * (1 - ui.EaseOut(p))
}

// header draws the break's label and, when asked, the time of day.
func (m *Manager) header(c *ui.Context, now time.Time, fg ui.Color) {
	ui.Row(c).Justify(ui.SpaceBetween).Children(func() {
		ui.Text(c, m.opts.Label(m.spec.Kind, m.spec.Strict)).
			FontSize(11).FixedLineHeight(theme.Px(c, 15)).FontWeight(500).
			LetterSpacing(1.5).TextColor(fg.Alpha(0.55))
		if m.bw.ShowCurrentTime {
			ui.Text(c, m.opts.Clock(now)).Font(theme.Mono).
				FontSize(11).FixedLineHeight(theme.Px(c, 15)).LetterSpacing(theme.Px(c, 0.44)).
				TextColor(fg.Alpha(0.55))
		}
	})
}

// mark draws the fermata, at any field size.
func (m *Manager) mark(c *ui.Context, mark, dot ui.Color) {
	box := ui.Box(c).Size(48, 48)
	box.Draw(func(p *ui.Painter, r ui.Rect) {
		theme.PaintMark(p, r, mark, dot)
	})
}

// hasHan reports whether s contains a Han character: the serif suggestion is
// set two points larger with a 1.25 line height for Chinese, as the design
// system asks, and without the negative tracking the Latin setting uses.
func hasHan(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

// suggestion draws the idea, or nothing when the setting hides it.
func (m *Manager) suggestion(c *ui.Context, fg ui.Color, long bool) {
	if !m.bw.ShowIdeas {
		return
	}
	title, text := m.idea.Title, m.idea.Text
	if m.spec.Title != "" {
		title = m.spec.Title
	}
	if m.spec.Text != "" {
		text = m.spec.Text
	}
	w, _ := c.Size()
	small := w < 1280
	han := hasHan(title + text)
	if long {
		if title != "" {
			size, line := float32(50), float32(57)
			if small {
				size, line = 46, 53
			}
			if han {
				size, line = size+2, size*1.25
			}
			el := ui.Text(c, title).Font(theme.Serif).
				FontSize(theme.Px(c, size)).FixedLineHeight(theme.Px(c, line)).
				FontWeight(300).TextAlign(ui.Center).
				MaxWidth(820).MaxLines(2).TextColor(fg).Margin(28, 0, 0, 0)
			if !han {
				el.LetterSpacing(theme.Px(c, -0.6))
			}
		}
		if text != "" {
			size, line := float32(26), float32(36)
			if han {
				size, line = 28, 35
			}
			ui.Text(c, text).Font(theme.Serif).
				FontSize(theme.Px(c, size)).FixedLineHeight(theme.Px(c, line)).
				FontWeight(300).TextAlign(ui.Center).MaxWidth(680).MaxLines(3).
				TextColor(fg.Alpha(0.72)).Margin(24, 0, 0, 0)
		}
		return
	}
	sentence := title
	if sentence == "" {
		sentence = text
	}
	if sentence == "" {
		return
	}
	size, line := float32(42), float32(51)
	if small {
		size, line = 38, 46
	}
	if han {
		size, line = size+2, size*1.25
	}
	el := ui.Text(c, sentence).Font(theme.Serif).
		FontSize(theme.Px(c, size)).FixedLineHeight(theme.Px(c, line)).
		FontWeight(300).TextAlign(ui.Center).
		MaxWidth(780).MaxLines(4).TextColor(fg).Margin(28, 0, 0, 0)
	if !han {
		el.LetterSpacing(theme.Px(c, -0.34))
	}
}

// progress draws the hairline: the share of the break that has passed.
func (m *Manager) progress(c *ui.Context, fg ui.Color, now time.Time, long bool) {
	width := float32(280)
	if long {
		width = 360
	}
	ratio := float32(0)
	if m.phase == planner.Breaking {
		ratio = float32(m.spec.Progress(now))
	}
	track := ui.Box(c).Width(width).Height(1).Background(fg.Alpha(0.25)).Margin(40, 0, 0, 0)
	filled := track.Animate("filled", ratio*width, 900*time.Millisecond)
	track.Children(func() {
		ui.Box(c).Width(filled).Height(1).Background(fg.Alpha(0.85))
	})
}

// countdown draws the time left, or how long a manual finish has waited.
func (m *Manager) countdown(c *ui.Context, fg, dot ui.Color, now time.Time) {
	label := status.Duration(m.spec.Remaining(now))
	color := fg.Alpha(0.65)
	if m.phase == planner.ManualAwait {
		label = status.Overshoot(now.Sub(m.spec.EndAt))
		color = dot
	}
	ui.Text(c, label).Font(theme.Mono).FontSize(theme.Px(c, 13)).FixedLineHeight(theme.Px(c, 16)).
		LetterSpacing(0.52).TextColor(color).Margin(16, 0, 0, 0)
}

// exit draws the break's one exit, if this moment has one.
func (m *Manager) exit(c *ui.Context, now time.Time, fg ui.Color) {
	action, ok := m.ActionAt(now)
	if !ok {
		return
	}
	ui.Row(c).Justify(ui.Center).Children(func() {
		label := m.opts.ActionLabel(action)
		button := ui.ButtonBase(c).Padding(3, 6).MinWidth(44).MinHeight(28).Gap(8).
			Label(label).Description(shortcutHint() + " " + label)
		hovered := button.Hovered()
		button.Children(func() {
			text := ui.Text(c, label).FontSize(theme.Px(c, 15)).FixedLineHeight(theme.Px(c, 20)).
				LetterSpacing(0.15).TextColor(fg.Alpha(0.82))
			if hovered {
				text.Underline().UnderlineOffset(4).DecorationThickness(1)
			}
			ui.Text(c, shortcutHint()).Font(theme.Mono).FontSize(theme.Px(c, 12)).FixedLineHeight(theme.Px(c, 16)).
				TextColor(fg.Alpha(0.55))
		})
		if button.Clicked() {
			m.opts.Action(action)
		}
	})
}

// shortcutHint writes the break's shortcut as the platform shows it.
func shortcutHint() string { return shortcut.End() }
