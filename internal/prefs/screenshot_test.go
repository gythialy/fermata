//go:build screenshots

package prefs

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/gythialy/fermata/internal/fonts"
)

// TestScreenshots writes the four settings pages used by the README and the
// documentation. It is only built with the screenshots tag:
//
//	go test -tags screenshots -run TestScreenshots ./internal/prefs
func TestScreenshots(t *testing.T) {
	if err := fonts.Load(); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join("..", "..", "docs", "images")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	w, _ := newWindow(t)
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
