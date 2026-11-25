package fullscreen

import "testing"

func display(x, y, w, h float64) Rect { return Rect{X: x, Y: y, W: w, H: h} }

func TestFullscreenIn(t *testing.T) {
	displays := []Rect{display(0, 0, 1440, 900), display(1440, 0, 1920, 1080)}
	full := window{layer: 0, pid: 42, rect: display(1440, 0, 1920, 1080), hasRect: true}
	normal := window{layer: 0, pid: 43, rect: display(1500, 100, 1200, 800), hasRect: true}
	small := window{layer: 0, pid: 44, rect: display(0, 0, 60, 20), hasRect: true}
	bar := window{layer: 25, pid: 45, rect: display(0, 0, 1440, 900), hasRect: true}
	self := window{layer: 0, pid: 7, rect: display(0, 0, 1440, 900), hasRect: true}
	noBounds := window{layer: 0, pid: 46}

	tests := []struct {
		name    string
		windows []window
		want    bool
	}{
		{"frontmost fullscreen", []window{full}, true},
		{"fullscreen on the other display", []window{full}, true},
		{"frontmost normal, fullscreen behind", []window{normal, full}, false},
		{"tiny helper skipped", []window{small, full}, true},
		{"menu bar layer skipped", []window{bar, full}, true},
		{"own window skipped", []window{self, full}, true},
		{"no bounds skipped", []window{noBounds, full}, true},
		{"nothing", nil, false},
	}
	for _, tc := range tests {
		if got := fullscreenIn(tc.windows, displays, 7); got != tc.want {
			t.Errorf("%s: %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestMatchesDisplayTolerance(t *testing.T) {
	d := display(0, 0, 1440, 900)
	if !matchesDisplay(Rect{X: 1.5, Y: -1, W: 1441, H: 899}, []Rect{d}) {
		t.Error("a window within tolerance does not match")
	}
	if matchesDisplay(Rect{X: 20, Y: 0, W: 1440, H: 900}, []Rect{d}) {
		t.Error("a window outside tolerance matches")
	}
}
