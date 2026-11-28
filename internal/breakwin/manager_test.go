package breakwin

import (
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/gythialy/fermata/internal/settings"
)

func TestCoversDisplay(t *testing.T) {
	d := mygo.Display{Bounds: mygo.Rectangle{Width: 1512, Height: 982}}
	if !coversDisplay(d, 1512, 982) {
		t.Error("a window the size of the display covers it")
	}
	if !coversDisplay(d, 1512, 1000) {
		t.Error("a window larger than the display covers it")
	}
	if coversDisplay(d, 1285, 835) {
		t.Error("the default 85% window leaves the desktop's bars in view")
	}
}

func TestBreakWindowSurface(t *testing.T) {
	bw := settings.BreakWindow{Transparent: true, Opacity: 0.6}
	field := ui.RGB(36, 65, 58)

	reduced := breakWindowSurface(bw, field, true, mygo.ThemePreferences{ReduceTransparency: true})
	if reduced.transparent || reduced.background != hex(field) || reduced.opacity != 1 || reduced.fade {
		t.Errorf("Reduce Transparency surface = %+v, want opaque and still", reduced)
	}
	if !bw.Transparent || bw.Opacity != 0.6 {
		t.Errorf("the user's settings changed: %+v", bw)
	}

	normal := breakWindowSurface(bw, field, true, mygo.ThemePreferences{})
	if !normal.transparent || normal.background != "" || normal.opacity != 0.6 || !normal.fade {
		t.Errorf("translucent surface = %+v, want the user's settings with a fade", normal)
	}
	still := breakWindowSurface(bw, field, true, mygo.ThemePreferences{ReduceMotion: true})
	if still.fade {
		t.Error("Reduce Motion still fades the panel in")
	}
}
