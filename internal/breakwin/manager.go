// Package breakwin shows a break: one window per display, the moon on the
// field, the suggestion, the countdown and one exit.
package breakwin

import (
	"log/slog"
	"strconv"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/gythialy/fermata/internal/ideas"
	"github.com/gythialy/fermata/internal/overlay"
	"github.com/gythialy/fermata/internal/planner"
	"github.com/gythialy/fermata/internal/settings"
	"github.com/gythialy/fermata/internal/theme"
)

// Action is what the break window asks the app to do.
type Action uint8

const (
	// Postpone moves the break later.
	Postpone Action = iota + 1
	// Skip ends the break early.
	Skip
	// Finish closes a break that waited for the user.
	Finish
)

// Options are the app's contributions to a break window.
type Options struct {
	// Settings returns the break window's settings when the break starts.
	Settings func() settings.BreakWindow
	// Idea returns the suggestion for a break, chosen once per break.
	Idea func(kind planner.Kind) ideas.Idea
	// Clock formats the current time of day.
	Clock func(time.Time) string
	// Action runs what the user asked for, on the main thread.
	Action func(Action)
	// Label names the break, as "MINI BREAK" or "STRICT LONG BREAK".
	Label func(kind planner.Kind, strict bool) string
	// ActionLabel names an exit, as "Postpone".
	ActionLabel func(Action) string
	// Dark reports whether the app follows a dark appearance.
	Dark func() bool
	// Background returns the bytes of a background image, or nil; it is
	// only called when the setting is on.
	Background func() []byte
	Log        *slog.Logger
}

// Manager owns the break's windows.
type Manager struct {
	opts Options
	spec planner.BreakSpec
	bw   settings.BreakWindow
	idea ideas.Idea
	bg   *ui.Bitmap

	phase     planner.Phase
	postpones int
	health    int

	// shownAt and animate drive the entrance: the panel fades in and the
	// content rises when a break starts, but not on a mid-break rebuild or
	// when the desktop asks for less motion.
	shownAt time.Time
	animate bool

	windows map[int64]*window
}

type window struct {
	win     *mygo.Window
	content bool
}

// breakSurface is the window surface for one break, after the desktop's
// accessibility preferences have been applied over the user's settings.
type breakSurface struct {
	transparent bool
	background  string
	opacity     float64
	fade        bool
}

// breakWindowSurface resolves the user's settings against the desktop's
// preferences without changing the settings themselves.
func breakWindowSurface(bw settings.BreakWindow, field ui.Color, animate bool, prefs mygo.ThemePreferences) breakSurface {
	surface := breakSurface{
		transparent: bw.Transparent || bw.Opacity < 1,
		background:  hex(field),
		opacity:     bw.Opacity,
		fade:        animate && !prefs.ReduceMotion,
	}
	if surface.transparent {
		surface.background = ""
	}
	if prefs.ReduceTransparency {
		surface.transparent = false
		surface.background = hex(field)
		surface.opacity = 1
		surface.fade = false
	}
	return surface
}

// New returns a manager for the app.
func New(opts Options) *Manager {
	return &Manager{opts: opts, windows: map[int64]*window{}}
}

// Visible reports whether break windows are open.
func (m *Manager) Visible() bool { return len(m.windows) > 0 }

// Spec returns the running break's snapshot.
func (m *Manager) Spec() planner.BreakSpec { return m.spec }

// Idea returns the suggestion the break shows.
func (m *Manager) Idea() ideas.Idea { return m.idea }

// Invalidate draws the windows again, after a language change.
func (m *Manager) Invalidate() {
	for _, w := range m.windows {
		w.win.Invalidate()
	}
}

// Show opens the break's windows. It is called once, when the break starts.
func (m *Manager) Show(spec planner.BreakSpec, phase planner.Phase) {
	m.Close()
	m.spec = spec
	m.phase = phase
	m.shownAt = time.Now()
	m.animate = !mygo.Theme.Preferences().ReduceMotion
	m.postpones = 0
	m.bw = m.opts.Settings()
	m.idea = m.opts.Idea(spec.Kind)
	m.bg = nil
	if m.bw.BackgroundImage && m.opts.Background != nil {
		if data := m.opts.Background(); len(data) > 0 {
			bitmap, err := ui.DecodeBitmap(data)
			if err != nil {
				m.opts.Log.Warn("background image could not be decoded", "error", err)
			} else {
				m.bg = bitmap
			}
		}
	}
	m.create(true)
}

// Sync updates what the view shows, as the app's state moves on.
func (m *Manager) Sync(phase planner.Phase, postpones, health int) {
	m.phase = phase
	m.postpones = postpones
	m.health = health
	for _, w := range m.windows {
		w.win.Invalidate()
	}
}

// Close takes the windows down.
func (m *Manager) Close() {
	for id, w := range m.windows {
		w.win.Hide()
		w.win.Destroy()
		delete(m.windows, id)
	}
	m.bg = nil
}

// Rebuild recreates the windows for the displays as they are now, keeping the
// running break's clock.
func (m *Manager) Rebuild() {
	if len(m.windows) == 0 {
		return
	}
	bg := m.bg
	m.Close()
	m.bg = bg
	m.animate = false
	m.create(false)
}

// ActionAt returns what the exit does at now, if any.
func (m *Manager) ActionAt(now time.Time) (Action, bool) {
	switch m.phase {
	case planner.ManualAwait:
		return Finish, true
	case planner.Breaking:
		if m.spec.Strict {
			return 0, false
		}
		if m.spec.CanPostpone(now, m.postpones) {
			return Postpone, true
		}
		return Skip, true
	}
	return 0, false
}

// create opens one window per target display. animate is true only when the
// break starts, not when a display change rebuilds the group.
func (m *Manager) create(animate bool) {
	displays := mygo.Screen.Displays()
	if len(displays) == 0 {
		m.opts.Log.Warn("no displays: the break has no window")
		return
	}
	for _, target := range targets(displays, m.bw, m.opts.Log) {
		m.windows[target.display.ID] = m.open(target.display, target.content, animate)
	}
}

// target is one display and whether it shows the content.
type target struct {
	display mygo.Display
	content bool
}

func (m *Manager) open(d mygo.Display, content bool, animate bool) *window {
	b := d.Bounds
	width, height := b.Width, b.Height
	if !m.bw.FillScreen {
		width = max(int(float64(b.Width)*m.bw.WidthRatio), 1)
		height = max(int(float64(b.Height)*m.bw.HeightRatio), 1)
	}
	x := b.X + (b.Width-width)/2
	y := b.Y + (b.Height-height)/2
	corner := 20
	if m.bw.FillScreen {
		corner = 0
	}
	bg, _ := theme.Field(m.bw, m.opts.Dark())
	if m.spec.Kind == planner.Long {
		bg = theme.Deeper(m.bw.Field, bg)
	}
	surface := breakWindowSurface(m.bw, bg, animate, mygo.Theme.Preferences())
	opts := mygo.WindowOptions{
		Title:             "Fermata",
		Width:             width,
		Height:            height,
		X:                 x,
		Y:                 y,
		Panel:             true,
		Frameless:         true,
		AlwaysOnTop:       true,
		Hidden:            true,
		DisableResize:     true,
		DisableMove:       true,
		DisableMinimize:   true,
		DisableMaximize:   true,
		DisableClose:      true,
		DisableFullScreen: true,
		SkipTaskbar:       true,
		CornerRadius:      corner,
		BackgroundColor:   surface.background,
		Transparent:       surface.transparent,
		Opacity:           surface.opacity,
		Content:           ui.View(func(c *ui.Context) { m.view(c, content) }),
	}
	if surface.fade {
		// The window itself fades in; the view moves the content up over a
		// slightly longer time, so the field arrives before the words.
		opts.Opacity = 0
	}
	win := mygo.NewWindow(opts)
	win.OnClose(func(e *mygo.CloseEvent) { e.PreventDefault() })
	win.ShowInactive()
	if coversDisplay(d, width, height) {
		// A field the size of the display reaches its edges: the desktop's own
		// bars must not sit on top of it. It is placed again after showing,
		// when AppKit has already pinned it below the menu bar.
		r := mygo.Rectangle{X: x, Y: y, Width: width, Height: height}
		if err := overlay.Cover(win, r); err != nil {
			m.opts.Log.Warn("break window could not cover the screen's own bars", "error", err)
		}
	}
	if surface.fade {
		go fadeIn(win, surface.opacity)
	}
	return &window{win: win, content: content}
}

// coversDisplay reports whether a break window of width × height fills its
// display: only then do the menu bar and the Dock have to make way for it.
func coversDisplay(d mygo.Display, width, height int) bool {
	return width >= d.Bounds.Width && height >= d.Bounds.Height
}

// entranceFade is how long the panel takes to arrive. The content rises for a
// little longer (see view.go), as the design asks.
const entranceFade = 180 * time.Millisecond

// contentRise is how long the content takes to settle; the panel's fade is
// shorter, so the field arrives before the words.
const contentRise = 220 * time.Millisecond

// fadeAt is the panel's opacity at elapsed, eased out over entranceFade.
func fadeAt(target float64, elapsed time.Duration) float64 {
	if elapsed >= entranceFade {
		return target
	}
	if elapsed < 0 {
		return 0
	}
	p := float32(elapsed) / float32(entranceFade)
	return target * float64(ui.EaseOut(p))
}

// fadeIn takes the window from transparent to its setting, without blocking
// the main thread. SetOpacity marshals onto the main thread itself.
func fadeIn(win *mygo.Window, target float64) {
	start := time.Now()
	for {
		elapsed := time.Since(start)
		if elapsed >= entranceFade {
			win.SetOpacity(target)
			return
		}
		win.SetOpacity(fadeAt(target, elapsed))
		time.Sleep(16 * time.Millisecond)
	}
}

// targets decides which displays show the field, and which of them show its
// content.
func targets(displays []mygo.Display, bw settings.BreakWindow, log *slog.Logger) []target {
	if !bw.AllScreens {
		d := resolveDisplay(displays, bw.Screen, log)
		return []target{{display: d, content: true}}
	}
	if bw.ContentScreen == settings.ScreenAll {
		out := make([]target, 0, len(displays))
		for _, d := range displays {
			out = append(out, target{display: d, content: true})
		}
		return out
	}
	content := resolveDisplay(displays, bw.ContentScreen, log)
	out := make([]target, 0, len(displays))
	for _, d := range displays {
		out = append(out, target{display: d, content: d.ID == content.ID})
	}
	return out
}

// resolveDisplay turns "primary", "cursor" or an index into a display, falling
// back to the pointer's display and then the primary one.
func resolveDisplay(displays []mygo.Display, name string, log *slog.Logger) mygo.Display {
	switch name {
	case settings.ScreenPrimary:
		return primary(displays)
	case settings.ScreenCursor:
		return cursor(displays)
	}
	if n, err := strconv.Atoi(name); err == nil && n >= 0 && n < len(displays) {
		return displays[n]
	}
	log.Warn("break window display does not exist; using the pointer's", "display", name)
	return cursor(displays)
}

func primary(displays []mygo.Display) mygo.Display {
	for _, d := range displays {
		if d.Primary {
			return d
		}
	}
	return displays[0]
}

func cursor(displays []mygo.Display) mygo.Display {
	if d, ok := nearest(displays); ok {
		return d
	}
	return primary(displays)
}

func nearest(displays []mygo.Display) (mygo.Display, bool) {
	if len(displays) == 0 {
		return mygo.Display{}, false
	}
	p := mygo.Screen.CursorScreenPoint()
	return mygo.Screen.DisplayNearestPoint(p), true
}

// hex turns a color into the CSS form a window option takes.
func hex(c ui.Color) string {
	const digits = "0123456789abcdef"
	out := []byte{'#', 0, 0, 0, 0, 0, 0}
	for i, v := range []uint8{c.R, c.G, c.B} {
		out[1+2*i] = digits[v>>4]
		out[2+2*i] = digits[v&0xf]
	}
	return string(out)
}
