package trayicon

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func decode(t *testing.T, data []byte, size int) image.Image {
	t.Helper()
	if len(data) == 0 {
		t.Fatal("no image")
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if b := img.Bounds(); b.Dx() != size || b.Dy() != size {
		t.Fatalf("size %dx%d, want %dx%d", b.Dx(), b.Dy(), size, size)
	}
	return img
}

func pixelAlpha(img image.Image, x, y int) uint8 {
	return color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA).A
}

func ink(t *testing.T, img image.Image) int {
	t.Helper()
	n := 0
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if pixelAlpha(img, x, y) > 0 {
				n++
			}
		}
	}
	return n
}

func TestMark(t *testing.T) {
	img := decode(t, Mark(36), 36)
	if got := ink(t, img); got < 40 {
		t.Errorf("only %d pixels of ink", got)
	}
	// The dot of the fermata sits under the arc, at (9, 14.5) of 18.
	if pixelAlpha(img, 18, 29) < 200 {
		t.Error("no dot")
	}
	// Transparent above the arc.
	if pixelAlpha(img, 18, 2) != 0 {
		t.Error("ink above the mark")
	}
}

func TestPaused(t *testing.T) {
	img := decode(t, Paused(36), 36)
	// Two bars at x 6.4 and 11.6 of 18, with a gap between them.
	if pixelAlpha(img, 13, 18) < 200 || pixelAlpha(img, 23, 18) < 200 {
		t.Error("the bars are missing")
	}
	if got := pixelAlpha(img, 18, 18); got != 0 {
		t.Errorf("the bars touch: alpha %d", got)
	}
}

func TestRing(t *testing.T) {
	empty := decode(t, Ring(36, 0), 36)
	full := decode(t, Ring(36, 1), 36)
	// The track is faint, the used part full.
	if got := pixelAlpha(empty, 18, 5); got == 0 || got >= 200 {
		t.Errorf("track alpha %d", got)
	}
	if got := pixelAlpha(full, 18, 5); got < 200 {
		t.Errorf("used alpha %d", got)
	}
	// The left of the ring is not used at 25%.
	quarter := decode(t, Ring(36, 0.25), 36)
	if pixelAlpha(quarter, 5, 18) >= 200 {
		t.Error("the arc covers more than a quarter")
	}
}

func TestCached(t *testing.T) {
	a := Mark(18)
	b := Mark(18)
	if &a[0] != &b[0] {
		t.Error("the mark was drawn twice")
	}
}
