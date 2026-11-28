// Package popover draws the status flyout under the menu bar icon: the
// moment's name, the countdown, the ruler that fills while the break
// approaches, and the actions that fit the state.
//
// The window owns the frame — material, corners, shadow and the opening
// animation are the system's (see app.togglePopover), which is why this view
// draws no background where the material shows. The countdown stays one
// ui.Text with a numeric content transition, so the digits roll instead of
// the whole block fading; the ruler beneath it is drawn from the same share
// the tray icon's ring mode uses.
package popover

import (
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/gythialy/fermata/internal/icons"
	"github.com/gythialy/fermata/internal/theme"
	"github.com/gythialy/fermata/locales"
)

// ticks is how many marks the ruler draws; every fifth one is a minute mark.
// It is a scale, not a clock: the share fills it as the wait passes.
const ticks = 60

// hoverUnderline is how long the footer's underline fades in and out.
const hoverUnderline = 120 * time.Millisecond

// An Action is one button of the flyout.
type Action struct {
	Label   string
	Primary bool
	Run     func()
}

// Options is what the flyout shows in this frame.
type Options struct {
	// State names the moment, as "Next mini break"; the view uppercases it.
	State string
	// Remaining is the countdown: "21m", "8s" or "12:04".
	Remaining string
	// Detail is the second line: the rhythm, or the break's suggestion.
	Detail string
	// Share is how much of this wait or break has passed, 0–1.
	Share float64
	// Actions are the buttons of this state, in order; none hides the block.
	Actions []Action
	// Settings is the footer's text button; nil hides the footer, as a
	// strict break does.
	Settings func()
	// Menu and Close are the header's icon buttons; nil hides one.
	Menu, Close func()
}

// Show builds the flyout's content for opts in this frame.
func Show(c *ui.Context, opts Options) {
	t := theme.ApplyFlyout(c)
	root := ui.Column(c).Padding(t.Space(4)).Gap(t.Space(3))
	if !c.Vibrancy() {
		root.Background(t.Background)
	}
	root.Children(func() {
		header(c, t, opts)
		ui.Column(c).Gap(t.Space(2.5)).Children(func() {
			ui.Text(c, opts.Remaining).Font(theme.Mono).FontSize(theme.Px(c, 34)).Bold().
				FixedLineHeight(theme.Px(c, 40)).
				ContentTransition(ui.ContentTransition{Numeric: true, Down: true})
			ruler(c, t, opts.Share)
			if opts.Detail != "" {
				ui.Text(c, opts.Detail).FontSize(theme.Px(c, 11)).FixedLineHeight(theme.Px(c, 15)).
					TextColor(t.TextMuted).MaxLines(2)
			}
		})
		if len(opts.Actions) > 0 {
			actions(c, t, opts.Actions)
		}
		if opts.Settings != nil {
			ui.Divider(c)
			textButton(c, t, locales.PopoverSettings(), opts.Settings)
		}
	})
}

// header draws the mark, the state and the two icon buttons.
func header(c *ui.Context, t *ui.Theme, opts Options) {
	ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
		ui.Box(c).Size(t.Space(5), t.Space(5)).Draw(func(p *ui.Painter, r ui.Rect) {
			theme.PaintMark(p, r, t.Text, t.Accent)
		})
		ui.Text(c, strings.ToUpper(opts.State)).FontSize(theme.Px(c, 10)).Bold().
			LetterSpacing(theme.Px(c, 1.6)).FixedLineHeight(theme.Px(c, 14)).
			TextColor(t.TextMuted).MaxLines(1)
		ui.Spacer(c)
		iconButton(c, t, locales.ActionShowMenu(), icons.More, opts.Menu)
		iconButton(c, t, locales.ActionClose(), icons.Close, opts.Close)
	})
}

// ruler draws the wait's scale: a tick per sixtieth, the passed ones in the
// accent, and a cursor at the edge that has passed.
func ruler(c *ui.Context, t *ui.Theme, share float64) {
	share = min(max(share, 0), 1)
	box := ui.Box(c).Fill().Height(theme.Px(c, 12))
	box.Draw(func(p *ui.Painter, r ui.Rect) {
		step := r.W / ticks
		passed := share * ticks
		for i := range ticks {
			top, height := r.Y+r.H/2, r.H/2
			if i%5 == 0 {
				top, height = r.Y, r.H
			}
			color := t.Border
			if float64(i) < passed {
				color = t.Accent
			}
			p.Fill(ui.Rect{X: r.X + float32(i)*step, Y: top, W: 1, H: height}, color, 0)
		}
		if share > 0 {
			x := r.X + r.W*float32(share)
			p.Fill(ui.Rect{X: x - 1, Y: r.Y, W: 2, H: r.H * 0.83}, t.Accent, 1)
		}
	})
	box.Role(ui.RoleProgress).Label(fmt.Sprintf("%d%%", int(share*100+0.5)))
}

// actions draws the state's buttons; the block's height change is a
// transition, so the flyout's own FitContent follows it in two frames.
func actions(c *ui.Context, t *ui.Theme, list []Action) {
	block := ui.Column(c).Gap(t.Space(2)).Transition(ui.ElementTransition{})
	block.Children(func() {
		for _, a := range list {
			ui.Row(c).Children(func() {
				if a.Primary {
					ui.PrimaryButton(c, a.Label).Grow(1).OnClick(a.Run)
					return
				}
				ui.Button(c, a.Label).Grow(1).OnClick(a.Run)
			})
		}
	})
}

// iconButton draws a 22 DIP ghost button with a 12 DIP glyph. The glyph alone
// says nothing, so its name and tooltip are how a screen reader, and a reader
// of the window on Windows, find it.
func iconButton(c *ui.Context, t *ui.Theme, name string, glyph func(*ui.Painter, ui.Rect, ui.Color), run func()) {
	if run == nil {
		return
	}
	b := ui.ButtonBase(c).Size(t.Space(5.5), t.Space(5.5)).Gap(0).Radius(t.Space(1.5)).
		Label(name).Tooltip(name)
	if b.Hovered() {
		b.Background(t.Surface)
	}
	b.Children(func() {
		ui.Box(c).Fill().Draw(func(p *ui.Painter, r ui.Rect) {
			glyph(p, icons.Box(r, 12), t.TextMuted)
		})
	})
	b.OnClick(run)
}

// textButton draws the footer's settings row: a quiet text, underlined while
// the pointer is on it, as the break window's exit is.
func textButton(c *ui.Context, t *ui.Theme, label string, run func()) {
	b := ui.ButtonBase(c).Height(theme.Px(c, 20)).Padding(0, t.Space(0.5)).Gap(0).Radius(t.Space(1))
	hovered := b.Hovered()
	b.Children(func() {
		text := ui.Text(c, label).FontSize(theme.Px(c, 11)).FixedLineHeight(theme.Px(c, 16)).
			TextColor(t.TextMuted)
		target := float32(0)
		if hovered {
			text.TextColor(t.Text)
			target = 1
		}
		underline := text.Animate("underline", target, hoverUnderline)
		if underline > 0 {
			// Fade the decoration so zero means no line; layout stays put.
			text.Underline().UnderlineOffset(3).DecorationThickness(1).
				DecorationColor(t.Text.Alpha(underline))
		}
	})
	b.OnClick(run)
}
