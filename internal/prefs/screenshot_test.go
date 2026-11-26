//go:build screenshots

package prefs

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/i18n"
	"github.com/egoist/mygo/ui"
	"github.com/gythialy/fermata/internal/fonts"
	"github.com/gythialy/fermata/internal/ideas"
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

// TestScreenshots writes the four settings pages used by the README and the
// documentation, once per language. It is only built with the screenshots tag:
//
//	go test -tags screenshots -run TestScreenshots ./internal/prefs
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
			w, _ := newWindow(t)
			w.opts.PreviewText = func() string { return shotPreviewIdeas[shot.locale].Text }
			tt := ui.NewTester(w.view, 800, 720)
			for _, page := range []struct{ section, name string }{
				{SectionGeneral, "settings-general.png"},
				{SectionSchedule, "settings-schedule.png"},
				{SectionAppearance, "settings-appearance.png"},
				{SectionAbout, "settings-about.png"},
			} {
				w.section = page.section
				tt.SetScale(2)
				tt.Frame()
				writePNG(t, tt.Image(), filepath.Join(out, page.name))
			}
		})
	}
}

// shotPreviewIdeas keep the narrow first-run and appearance previews on one
// line: the Chinese suggestion is shorter than the break window's draw.
var shotPreviewIdeas = map[string]ideas.Idea{
	"en":      {Text: "Look out a window until your eyes relax."},
	"zh-Hans": {Text: "慢慢地眨几次眼。"},
}

// useLocale draws the views in locale until the test ends.
func useLocale(t *testing.T, locale string) {
	t.Helper()
	i18n.SetLocaleFunc(func() string { return locale })
	t.Cleanup(func() { i18n.SetLocaleFunc(nil) })
}

// writePNG writes img to path.
func writePNG(t *testing.T, img image.Image, path string) {
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
