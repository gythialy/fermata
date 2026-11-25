package tray

import (
	"runtime"
	"strconv"
	"time"

	"github.com/egoist/mygo"
	"github.com/gythialy/fermata/internal/planner"
)

// Menu item IDs, fixed so that Update finds them.
const (
	idStatus      = "status"
	idSkipTo      = "skipTo"
	idSkipMini    = "skipMini"
	idSkipLong    = "skipLong"
	idPause       = "pause"
	idPausePrefix = "pause"
	idMorning     = "pauseMorning"
	idForever     = "pauseForever"
	idResume      = "resume"
	idReset       = "reset"
	idSettings    = "settings"
	idQuit        = "quit"
)

// maxPauses is how many timed pauses the submenu can show.
const maxPauses = 8

// buildMenu creates the whole menu once; Update shows and hides what a state
// allows.
func (c *Controller) buildMenu() *mygo.Menu {
	c.items = map[string]*mygo.MenuItem{}

	c.items[idStatus] = &mygo.MenuItem{ID: idStatus, Disabled: true}

	c.items[idSkipMini] = &mygo.MenuItem{ID: idSkipMini, Label: c.labels.Mini,
		Click: func(*mygo.MenuItem, *mygo.Window) { c.actions.SkipTo(planner.Mini) }}
	c.items[idSkipLong] = &mygo.MenuItem{ID: idSkipLong, Label: c.labels.Long,
		Click: func(*mygo.MenuItem, *mygo.Window) { c.actions.SkipTo(planner.Long) }}
	c.items[idSkipTo] = &mygo.MenuItem{
		ID:      idSkipTo,
		Label:   c.labels.SkipTo,
		Submenu: []*mygo.MenuItem{c.items[idSkipMini], c.items[idSkipLong]},
	}

	for i := range maxPauses {
		item := &mygo.MenuItem{
			ID: idPausePrefix + strconv.Itoa(i),
			Click: func(*mygo.MenuItem, *mygo.Window) {
				c.pauseFor(i)
			},
		}
		c.pause = append(c.pause, item)
	}
	c.items[idMorning] = &mygo.MenuItem{ID: idMorning, Label: c.labels.Morning,
		Click: func(*mygo.MenuItem, *mygo.Window) { c.actions.PauseMorning() }}
	c.items[idForever] = &mygo.MenuItem{ID: idForever, Label: c.labels.Indefinitely,
		Click: func(*mygo.MenuItem, *mygo.Window) { c.actions.Pause(0, true) }}

	pauseItems := append([]*mygo.MenuItem{}, c.pause...)
	pauseItems = append(pauseItems, c.items[idMorning], c.items[idForever])
	c.items[idPause] = &mygo.MenuItem{
		ID:      idPause,
		Label:   c.labels.Pause,
		Submenu: pauseItems,
	}

	c.items[idResume] = &mygo.MenuItem{ID: idResume, Label: c.labels.Resume,
		Click: func(*mygo.MenuItem, *mygo.Window) { c.actions.Resume() }}
	c.items[idReset] = &mygo.MenuItem{ID: idReset, Label: c.labels.Reset,
		Click: func(*mygo.MenuItem, *mygo.Window) { c.actions.Reset() }}
	c.items[idSettings] = &mygo.MenuItem{ID: idSettings, Label: c.labels.Settings,
		Accelerator: settingsAccelerator(),
		Click:       func(*mygo.MenuItem, *mygo.Window) { c.actions.Settings() }}
	c.items[idQuit] = &mygo.MenuItem{ID: idQuit, Label: c.labels.Quit,
		Role:  mygo.RoleQuit,
		Click: func(*mygo.MenuItem, *mygo.Window) { c.actions.Quit() }}

	return mygo.NewMenu([]*mygo.MenuItem{
		c.items[idStatus],
		mygo.Separator(),
		c.items[idSkipTo],
		c.items[idPause],
		c.items[idResume],
		c.items[idReset],
		mygo.Separator(),
		c.items[idSettings],
		mygo.Separator(),
		c.items[idQuit],
	})
}

// pauseFor asks for the timed pause of the current menu's i-th entry.
func (c *Controller) pauseFor(i int) {
	if i < len(c.durations) {
		c.actions.Pause(c.durations[i], false)
	}
}

// setPauses gives the timed entries their labels and visibility.
func (c *Controller) setPauses(durations []time.Duration) {
	c.durations = durations
	for i, item := range c.pause {
		if i < len(durations) {
			if item.Label != c.labels.PauseFor(durations[i]) {
				item.SetLabel(c.labels.PauseFor(durations[i]))
			}
			item.SetVisible(true)
		} else {
			item.SetVisible(false)
		}
	}
}

func settingsAccelerator() string {
	if runtime.GOOS == "darwin" {
		return "CmdOrCtrl+,"
	}
	return "Ctrl+,"
}
