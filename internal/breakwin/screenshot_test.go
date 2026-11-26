//go:build screenshots

package breakwin

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/i18n"
	"github.com/egoist/mygo/ui"
	"github.com/gythialy/fermata/internal/fonts"
	"github.com/gythialy/fermata/internal/ideas"
	"github.com/gythialy/fermata/internal/planner"
	"github.com/gythialy/fermata/locales"
)

// shotLocales are the languages the README and the documentation show: English
// writes docs/images and zh-Hans writes docs/images/zh-Hans.
var shotLocales = []struct {
	locale string
	dir    []string
}{
	{"en", []string{"..", "..", "docs", "images"}},
	{"zh-Hans", []string{"..", "..", "docs", "images", "zh-Hans"}},
}

// shotIdeas are fixed suggestions per language, so re-running the target does
// not churn the images with a different draw.
var shotIdeas = map[string]struct{ mini, long ideas.Idea }{
	"en": {
		mini: ideas.Idea{Text: "Look out a window until your eyes relax."},
		long: ideas.Idea{Title: "Step away", Text: "Walk to a window and look at something far away."},
	},
	"zh-Hans": {
		mini: ideas.Idea{Text: "望向窗外，直到眼睛放松。"},
		long: ideas.Idea{Title: "离开一下", Text: "走到窗边，看看很远的地方。"},
	},
}

// TestScreenshots writes the break windows used by the README and the
// documentation, once per language. It is only built with the screenshots tag:
//
//	go test -tags screenshots -run TestScreenshots ./internal/breakwin
func TestScreenshots(t *testing.T) {
	if err := fonts.Load(); err != nil {
		t.Fatal(err)
	}
	for _, shot := range shotLocales {
		t.Run(shot.locale, func(t *testing.T) {
			useLocale(t, shot.locale)
			out := filepath.Join(shot.dir...)
			if err := os.MkdirAll(out, 0o755); err != nil {
				t.Fatal(err)
			}

			t.Run("mini", func(t *testing.T) {
				f, tt := newFixture(t)
				now := tt.Now()
				f.m.spec.StartAt = now.Add(-4 * time.Second)
				f.m.spec.EndAt = now.Add(16 * time.Second)
				f.m.spec.Duration = 20 * time.Second
				f.m.phase = planner.Breaking
				f.m.opts.Label = breakLabel
				f.m.opts.ActionLabel = actionLabel
				f.m.idea = shotIdeas[shot.locale].mini
				writePNG(t, tt, filepath.Join(out, "break-mini.png"))
			})

			t.Run("long", func(t *testing.T) {
				f, tt := newFixture(t)
				now := tt.Now()
				f.m.opts.Label = breakLabel
				f.m.opts.ActionLabel = actionLabel
				f.m.spec = planner.BreakSpec{
					Kind:     planner.Long,
					StartAt:  now.Add(-time.Minute),
					EndAt:    now.Add(4 * time.Minute),
					Duration: 5 * time.Minute,
					Postpone: planner.PostponeConfig{Enabled: true, Limit: 1, Percent: 30, Time: 5 * time.Minute},
				}
				f.m.idea = shotIdeas[shot.locale].long
				f.m.phase = planner.Breaking
				writePNG(t, tt, filepath.Join(out, "break-long.png"))
			})
		})
	}
}

// useLocale draws the views in locale until the test ends.
func useLocale(t *testing.T, locale string) {
	t.Helper()
	i18n.SetLocaleFunc(func() string { return locale })
	t.Cleanup(func() { i18n.SetLocaleFunc(nil) })
}

// breakLabel names the break the way internal/app does.
func breakLabel(kind planner.Kind, strict bool) string {
	switch {
	case strict && kind == planner.Long:
		return locales.BreakLabelStrictLong()
	case strict:
		return locales.BreakLabelStrictMini()
	case kind == planner.Long:
		return locales.BreakLabelLong()
	default:
		return locales.BreakLabelMini()
	}
}

// actionLabel names the break window's one exit the way internal/app does.
func actionLabel(a Action) string {
	switch a {
	case Postpone:
		return locales.BreakPostpone()
	case Finish:
		return locales.BreakFinish()
	default:
		return locales.BreakSkip()
	}
}

// writePNG renders the tester at 2x, cuts the field's corners out of it and
// writes it to path.
func writePNG(t *testing.T, tt *ui.Tester, path string) {
	t.Helper()
	tt.SetScale(2)
	tt.Frame()
	writeImage(t, cutCorners(tt.Image(), 2), path)
}

// writeImage encodes img into a PNG at path.
func writeImage(t *testing.T, img image.Image, path string) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(file, img); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

// cutCorners turns the field's rounded corners transparent, so the stills and
// the loop keep the window's shape and sit on GitHub's light and dark themes
// alike instead of showing the white canvas the headless renderer starts from.
//
// The renderer blends its anti-aliased corner with that white canvas. Rather
// than guess the curve, every pixel in a corner box is un-blended: p = field·a
// + white·(1−a), so a falls out of the channel that differs most from white,
// and the pixel is written back as the field's colour at that alpha. Pixels
// outside the curve come out fully transparent, and the window edge's own
// anti-aliasing is kept exactly.
//
// Content never reaches a corner box, which is the radius plus room for the
// renderer's soft corner.
func cutCorners(img *image.RGBA, scale float32) *image.NRGBA {
	out := image.NewNRGBA(img.Bounds())
	draw.Draw(out, out.Bounds(), img, img.Bounds().Min, draw.Src)
	radius := int(fieldRadius * scale)
	if radius < 1 {
		return out
	}
	bounds := out.Bounds()
	size := radius + 24
	if size*2 > bounds.Dx() || size*2 > bounds.Dy() {
		return out
	}
	// A point inside the field, clear of both the corner and the content.
	fill := out.NRGBAAt(bounds.Min.X+size-4, bounds.Min.Y+size-4)
	channel, span := 0, 255-float64(fill.R)
	for i, value := range []uint8{fill.R, fill.G, fill.B} {
		if d := 255 - float64(value); d > span {
			channel, span = i, d
		}
	}
	if span == 0 {
		return out
	}
	origins := [4][2]int{
		{0, 0},
		{bounds.Dx() - size, 0},
		{0, bounds.Dy() - size},
		{bounds.Dx() - size, bounds.Dy() - size},
	}
	for _, origin := range origins {
		for y := origin[1]; y < origin[1]+size; y++ {
			for x := origin[0]; x < origin[0]+size; x++ {
				at := out.PixOffset(bounds.Min.X+x, bounds.Min.Y+y)
				a := (255 - float64(out.Pix[at+channel])) / span
				a = min(max(a, 0), 1)
				out.SetNRGBA(bounds.Min.X+x, bounds.Min.Y+y, color.NRGBA{
					R: fill.R, G: fill.G, B: fill.B,
					A: uint8(a*255 + 0.5),
				})
			}
		}
	}
	return out
}
