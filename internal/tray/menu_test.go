package tray

import (
	"testing"
	"time"

	"github.com/gythialy/fermata/internal/planner"
	"github.com/gythialy/fermata/internal/status"
)

func newController() *Controller {
	c := &Controller{
		actions: Actions{},
		labels: Labels{
			SkipTo: "Skip to", Mini: "Mini break", Long: "Long break",
			Pause: "Pause breaks", Resume: "Resume breaks", Reset: "Reset breaks",
			Settings: "Settings…", Quit: "Quit Fermata",
			Morning: "Until morning", Indefinitely: "Indefinitely",
			PauseFor: func(d time.Duration) string { return d.String() },
		},
	}
	c.menu = c.buildMenu()
	return c
}

func visible(c *Controller, id string) bool {
	item := c.menu.ItemByID(id)
	if item == nil {
		return false
	}
	return item.IsVisible()
}

func TestMenuWhileWaiting(t *testing.T) {
	c := newController()
	c.applyItems(Snapshot{
		Phase: planner.Waiting, CanPause: true, CanReset: true, CanSkipTo: true,
		MiniEnabled: true, LongEnabled: true,
		PauseDurations: []time.Duration{30 * time.Minute, time.Hour},
		State:          status.Lines{Status: "Next mini break in about 9 minutes"},
	})
	for _, id := range []string{idStatus, idSkipTo, idPause, idReset, idSettings, idQuit} {
		if !visible(c, id) {
			t.Errorf("%s is hidden while waiting", id)
		}
	}
	if visible(c, idResume) {
		t.Error("resume is shown while waiting")
	}
	if !visible(c, idSkipMini) || !visible(c, idSkipLong) {
		t.Error("both break kinds are enabled but the skip submenu hides one")
	}
	if got := c.menu.ItemByID(idStatus).Label; got != "Next mini break in about 9 minutes" {
		t.Errorf("status %q", got)
	}
	// The skip submenu must name both kinds, or macOS shows an empty popup.
	skip := c.menu.ItemByID(idSkipTo)
	if len(skip.Submenu) != 2 {
		t.Fatalf("skip submenu has %d items", len(skip.Submenu))
	}
	for i, want := range []string{"Mini break", "Long break"} {
		if skip.Submenu[i].Label != want {
			t.Errorf("skip submenu %d label %q, want %q", i, skip.Submenu[i].Label, want)
		}
	}
}

func TestMenuWhileManuallyPaused(t *testing.T) {
	c := newController()
	c.applyItems(Snapshot{
		Phase: planner.Paused, CanResume: true,
		State: status.Lines{Status: "Paused — indefinitely"},
	})
	if !visible(c, idResume) {
		t.Error("resume is hidden while paused")
	}
	for _, id := range []string{idSkipTo, idPause, idReset} {
		if visible(c, id) {
			t.Errorf("%s is shown while paused", id)
		}
	}
	if !visible(c, idSettings) || !visible(c, idQuit) {
		t.Error("settings or quit hidden while paused")
	}
}

func TestMenuWhileStrict(t *testing.T) {
	c := newController()
	c.applyItems(Snapshot{
		Phase:  planner.Breaking,
		Strict: true,
		State:  status.Lines{Status: "Strict mini break — 0:14 left"},
	})
	for _, id := range []string{idSkipTo, idPause, idResume, idReset, idSettings, idQuit} {
		if visible(c, id) {
			t.Errorf("%s is offered during a strict break", id)
		}
	}
	if !visible(c, idStatus) {
		t.Error("the status line is hidden during a strict break")
	}
}

func TestStrictMenuCanKeepSettings(t *testing.T) {
	c := newController()
	c.applyItems(Snapshot{
		Phase: planner.Breaking, Strict: true, KeepInStrict: true,
		State: status.Lines{Status: "Strict mini break — 0:14 left"},
	})
	if !visible(c, idSettings) {
		t.Error("settings and about are hidden with KeepInStrict")
	}
	if visible(c, idQuit) {
		t.Error("quit is offered during a strict break")
	}
}

func TestPauseSubmenuLabels(t *testing.T) {
	c := newController()
	c.applyItems(Snapshot{
		Phase: planner.Waiting, CanPause: true, CanReset: true, CanSkipTo: true,
		MiniEnabled: true, LongEnabled: false,
		PauseDurations: []time.Duration{30 * time.Minute},
		State:          status.Lines{Status: "x"},
	})
	if c.pause[0].Label != (30 * time.Minute).String() {
		t.Errorf("pause label %q", c.pause[0].Label)
	}
	if c.pause[1].IsVisible() {
		t.Error("an unused pause entry is visible")
	}
	if visible(c, idSkipLong) {
		t.Error("a disabled break kind is in the skip submenu")
	}
}

func TestMenuHasNoAbout(t *testing.T) {
	c := newController()
	if item := c.menu.ItemByID("about"); item != nil {
		t.Fatalf("the tray menu has an About item: %q", item.Label)
	}
}
