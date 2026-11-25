// Package tray owns the menu bar icon and its menu: the status line, the
// commands, and the icon styles the settings choose.
package tray

import (
	"runtime"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/gythialy/fermata/internal/planner"
	"github.com/gythialy/fermata/internal/status"
)

// Actions are what choosing a menu item does. They run on the main thread.
type Actions struct {
	// TogglePopover opens or closes the status popover; HidePopover closes
	// it before the context menu opens. Linux keeps the plain menu.
	TogglePopover func()
	HidePopover   func()
	SkipTo        func(planner.Kind)
	Pause         func(d time.Duration, indefinite bool)
	PauseMorning  func()
	Resume        func()
	Reset         func()
	Settings      func()
	Quit          func()
}

// Labels are the menu's words, in the app's language.
type Labels struct {
	SkipTo       string
	Mini         string
	Long         string
	Pause        string
	Resume       string
	Reset        string
	Settings     string
	Quit         string
	Morning      string
	Indefinitely string
	PauseFor     func(time.Duration) string
}

// Snapshot is what the tray shows at one moment.
type Snapshot struct {
	State status.Lines
	// Style is settings.StyleMark, StyleTime or StyleProgress.
	Style string
	// Title is the menu bar's text for the time style, as "9m".
	Title string
	// Icon is the rendered PNG, and IconKey identifies it so that unchanged
	// icons are not set again.
	Icon    []byte
	IconKey string
	// Status is the planner's state, for the menu matrix.
	Phase     planner.Phase
	Strict    bool
	CanPause  bool
	CanResume bool
	CanReset  bool
	CanSkipTo bool
	// MiniEnabled and LongEnabled decide the skip submenu.
	MiniEnabled, LongEnabled bool
	// KeepInStrict keeps settings in a strict break's menu.
	KeepInStrict   bool
	PauseDurations []time.Duration
}

// Controller is the tray icon and its menu.
type Controller struct {
	tray    *mygo.Tray
	menu    *mygo.Menu
	items   map[string]*mygo.MenuItem
	pause   []*mygo.MenuItem
	actions Actions
	labels  Labels

	durations []time.Duration

	lastSignature string
	lastIcon      string
	lastTitle     string
	lastTooltip   string
}

// New creates the tray icon with its menu. It must run on the main thread.
func New(icon []byte, actions Actions, labels Labels) (*Controller, error) {
	c := &Controller{actions: actions, labels: labels}
	c.menu = c.buildMenu()
	t, err := mygo.NewTray(mygo.TrayOptions{
		ID:              "fermata",
		Icon:            icon,
		IconIsTemplate:  true,
		MonospacedTitle: true,
		ToolTip:         "Fermata",
	})
	if err != nil {
		return nil, err
	}
	c.tray = t
	if runtime.GOOS == "linux" {
		// AppIndicator only shows menus, and tells nothing of clicks.
		t.SetMenu(c.menu)
	} else {
		if actions.TogglePopover != nil {
			t.OnClick(actions.TogglePopover)
		}
		t.OnRightClick(func() {
			if actions.HidePopover != nil {
				actions.HidePopover()
			}
			// The menu for this popup only, so that a left click keeps
			// opening the popover.
			t.SetMenu(c.menu)
			defer t.SetMenu(nil)
			t.PopUpMenu()
		})
	}
	return c, nil
}

// ShowMenu opens the tray menu programmatically, as the popover's menu
// button does where a right click may not reach the icon.
func (c *Controller) ShowMenu() {
	c.tray.SetMenu(c.menu)
	defer c.tray.SetMenu(nil)
	c.tray.PopUpMenu()
}

// Native returns the mygo tray, which a flyout anchors to.
func (c *Controller) Native() *mygo.Tray { return c.tray }

// Menu returns the menu, for tests.
func (c *Controller) Menu() *mygo.Menu { return c.menu }

// Relabel replaces the menu's words, as when the language changes.
func (c *Controller) Relabel(l Labels) {
	c.labels = l
	c.setLabel(idSkipTo, l.SkipTo)
	c.setLabel(idSkipMini, l.Mini)
	c.setLabel(idSkipLong, l.Long)
	c.setLabel(idPause, l.Pause)
	c.setLabel(idResume, l.Resume)
	c.setLabel(idReset, l.Reset)
	c.setLabel(idSettings, l.Settings)
	c.setLabel(idQuit, l.Quit)
	c.setLabel(idMorning, l.Morning)
	c.setLabel(idForever, l.Indefinitely)
	c.setPauses(c.durations)
	c.tray.SetMenu(c.menu)
	c.lastSignature = ""
}

// Update applies a snapshot. Labels change in place, so an open menu is not
// rebuilt away.
func (c *Controller) Update(s Snapshot) {
	sig := signature(s)
	if sig != c.lastSignature {
		c.applyItems(s)
		c.lastSignature = sig
	}
	if s.IconKey != c.lastIcon && len(s.Icon) > 0 {
		if err := c.tray.SetIcon(s.Icon, true); err == nil {
			c.lastIcon = s.IconKey
		}
	}
	title := ""
	if s.Style == "time" {
		title = s.Title
	}
	if title != c.lastTitle {
		c.tray.SetTitle(title)
		c.lastTitle = title
	}
	if s.State.Tooltip != c.lastTooltip {
		c.tray.SetToolTip(s.State.Tooltip)
		c.lastTooltip = s.State.Tooltip
	}
}

// applyItems sets what the menu shows for a state.
func (c *Controller) applyItems(s Snapshot) {
	c.setLabel("status", s.State.Status)
	c.setSubtitle("status", s.State.Detail)
	c.item("status").SetEnabled(false)

	strict := s.Strict && (s.Phase == planner.Breaking || s.Phase == planner.ManualAwait)
	showCommands := !strict
	showResume := s.CanResume
	if strict {
		showCommands = false
		showResume = false
	}
	// While the app is paused for a reason of its own, only settings, about
	// and quit are useful.
	autoPaused := s.Phase == planner.Paused && !s.CanResume

	c.show("skipTo", showCommands && s.CanSkipTo)
	c.show("pause", showCommands && s.CanPause && !autoPaused)
	c.show("resume", showResume && !strict)
	c.show("reset", showCommands && s.CanReset && !autoPaused)
	c.show("settings", showCommands || strict && s.KeepInStrict)
	c.show("quit", !strict)

	c.visible("skipMini", s.MiniEnabled)
	c.visible("skipLong", s.LongEnabled)

	c.setPauses(s.PauseDurations)
	c.visible("pauseMorning", true)
	c.visible("pauseForever", true)
}

func (c *Controller) item(id string) *mygo.MenuItem {
	return c.items[id]
}

func (c *Controller) show(id string, visible bool) {
	if it := c.item(id); it != nil {
		it.SetVisible(visible)
	}
}

func (c *Controller) visible(id string, visible bool) {
	c.show(id, visible)
}

func (c *Controller) setLabel(id, label string) {
	if it := c.item(id); it != nil && it.Label != label {
		it.SetLabel(label)
	}
}

func (c *Controller) setSubtitle(id, subtitle string) {
	if it := c.item(id); it != nil && it.Subtitle != subtitle {
		it.SetSubtitle(subtitle)
	}
}

// signature names everything the menu shows, so that nothing is set twice.
func signature(s Snapshot) string {
	var b strings.Builder
	b.WriteString(s.State.Status)
	b.WriteByte('|')
	b.WriteString(s.State.Detail)
	b.WriteString("|strict=")
	b.WriteString(boolString(s.Strict && (s.Phase == planner.Breaking || s.Phase == planner.ManualAwait)))
	b.WriteString("|phase=")
	b.WriteString(s.Phase.String())
	b.WriteString("|pause=")
	b.WriteString(boolString(s.CanPause))
	b.WriteString("|resume=")
	b.WriteString(boolString(s.CanResume))
	b.WriteString("|reset=")
	b.WriteString(boolString(s.CanReset))
	b.WriteString("|skip=")
	b.WriteString(boolString(s.CanSkipTo))
	b.WriteString("|mini=")
	b.WriteString(boolString(s.MiniEnabled))
	b.WriteString("|long=")
	b.WriteString(boolString(s.LongEnabled))
	b.WriteString("|keep=")
	b.WriteString(boolString(s.KeepInStrict))
	for _, d := range s.PauseDurations {
		b.WriteString("|")
		b.WriteString(d.String())
	}
	return b.String()
}

func boolString(v bool) string {
	if v {
		return "1"
	}
	return "0"
}
