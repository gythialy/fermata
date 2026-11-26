//go:build screenshots

package popover

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/i18n"
	"github.com/egoist/mygo/ui"
	"github.com/gythialy/fermata/internal/fonts"
	"github.com/gythialy/fermata/internal/status"
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

// TestScreenshots writes the status popover used by the README and the
// documentation, once per language. It is only built with the screenshots tag:
//
//	go test -tags screenshots -run TestScreenshots ./internal/popover
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
			opts := waitingLocalized()
			tt := ui.NewTester(func(c *ui.Context) { Show(c, opts) }, 320, 360)
			tt.SetScale(2)
			tt.Frame()
			writePNG(t, tt.Image(), filepath.Join(out, "popover.png"))
		})
	}
}

// waitingLocalized is the waiting flyout in the current language, built the way
// internal/app builds it.
func waitingLocalized() Options {
	return Options{
		State:     locales.PopoverNextMini(),
		Remaining: status.Duration(21 * time.Minute),
		Detail:    locales.MenuAfterMini(4),
		Share:     0.34,
		Actions: []Action{
			{Label: locales.PopoverTakeBreak(), Primary: true},
			{Label: locales.PopoverPause()},
		},
		Settings: func() {},
		Menu:     func() {},
		Close:    func() {},
	}
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
