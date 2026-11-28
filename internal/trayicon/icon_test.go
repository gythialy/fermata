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
	// The spark of the moon sits at (22.8125, 22.1875) of the 32-unit
	// canvas, mapped into the 24.5 pt span: pixel (28, 27) at 36 px.
	if pixelAlpha(img, 28, 27) < 200 {
		t.Error("no spark")
	}
	// Transparent above the moon.
	if pixelAlpha(img, 18, 2) != 0 {
		t.Error("ink above the mark")
	}
}

func TestTemplatePixels(t *testing.T) {
	states := map[string][]byte{
		"mark":   Mark(36),
		"paused": Paused(36),
		"ring":   Ring(36, 1),
	}
	for name, data := range states {
		img := decode(t, data, 36)
		for y := range 36 {
			for x := range 36 {
				c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
				if c.A > 0 && (c.R != 0 || c.G != 0 || c.B != 0) {
					t.Fatalf("%s at %d,%d is not black with alpha: %+v", name, x, y, c)
				}
			}
		}
	}
}

func TestMarkPadding(t *testing.T) {
	img := decode(t, Mark(36), 36)
	minX, minY, maxX, maxY := 36, 36, -1, -1
	for y := range 36 {
		for x := range 36 {
			if pixelAlpha(img, x, y) > 0 {
				minX, minY = min(minX, x), min(minY, y)
				maxX, maxY = max(maxX, x), max(maxY, y)
			}
		}
	}
	if minX < 3 || minY < 3 || maxX > 32 || maxY > 32 {
		t.Errorf("ink bbox %d,%d–%d,%d needs padding in a 36 px template", minX, minY, maxX, maxY)
	}
	if w := maxX - minX + 1; w < 25 || w > 28 {
		t.Errorf("ink width %d, want 25–28 px", w)
	}
	if h := maxY - minY + 1; h < 24 || h > 27 {
		t.Errorf("ink height %d, want 24–27 px", h)
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
