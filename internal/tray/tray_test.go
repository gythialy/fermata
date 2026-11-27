package tray

import (
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo"
)

// icon records what a menu popup does to a tray icon.
type icon struct{ calls []string }

func (i *icon) SetMenu(m *mygo.Menu) {
	if m == nil {
		i.calls = append(i.calls, "clear")
		return
	}
	i.calls = append(i.calls, "set")
}

func (i *icon) PopUpMenu() { i.calls = append(i.calls, "popup") }

func TestPopUpLeavesTheIconWithoutAMenu(t *testing.T) {
	var got icon
	popUp(&got, &mygo.Menu{})
	want := "set popup clear"
	if strings.Join(got.calls, " ") != want {
		t.Errorf("the menu popup did %q, want %q", strings.Join(got.calls, " "), want)
	}
}

// TestRelabelNeedsNoIcon keeps Relabel off the icon: it relabels the menu it
// holds, and leaves the icon without one (popUp), because a menu on the icon
// turns every click into the menu, a left click included. The controller here
// has no native tray, so a call to it panics.
func TestRelabelNeedsNoIcon(t *testing.T) {
	c := newController()
	c.Relabel(Labels{
		Quit:     "Quit Fermata",
		Settings: "Settings…",
		PauseFor: func(d time.Duration) string { return d.String() },
	})
	if got := c.menu.ItemByID(idQuit).Label; got != "Quit Fermata" {
		t.Errorf("Quit label = %q, want it relabelled", got)
	}
}
