package breakwin

import (
	"testing"

	"github.com/egoist/mygo"
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
