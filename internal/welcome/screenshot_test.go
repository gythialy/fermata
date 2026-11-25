//go:build screenshots

package welcome

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/gythialy/fermata/internal/fonts"
)

// TestScreenshots writes the first-run window used by the README and the
// documentation. It is only built with the screenshots tag:
//
//	go test -tags screenshots -run TestScreenshots ./internal/welcome
func TestScreenshots(t *testing.T) {
	if err := fonts.Load(); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join("..", "..", "docs", "images")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	var started []bool
	w := newWindow(t, &started)
	tt := ui.NewTester(w.view, 800, 540)
	tt.SetScale(2)
	tt.Frame()
	writePNG(t, tt.Image(), filepath.Join(out, "welcome.png"))
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
