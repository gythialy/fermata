//go:build screenshots

package breakwin

import (
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

// writePNG renders the tester at 2x and writes it to path.
func writePNG(t *testing.T, tt *ui.Tester, path string) {
	t.Helper()
	tt.SetScale(2)
	tt.Frame()
	img := tt.Image()
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
