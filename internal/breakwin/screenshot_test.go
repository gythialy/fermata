//go:build screenshots

package breakwin

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/gythialy/fermata/internal/fonts"
	"github.com/gythialy/fermata/internal/ideas"
	"github.com/gythialy/fermata/internal/planner"
)

// TestScreenshots writes the break windows used by the README and the
// documentation. It is only built with the screenshots tag:
//
//	go test -tags screenshots -run TestScreenshots ./internal/breakwin
func TestScreenshots(t *testing.T) {
	if err := fonts.Load(); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join("..", "..", "docs", "images")
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
		writePNG(t, tt, filepath.Join(out, "break-mini.png"))
	})

	t.Run("long", func(t *testing.T) {
		f, tt := newFixture(t)
		now := tt.Now()
		f.m.opts.Label = func(planner.Kind, bool) string { return "LONG BREAK" }
		f.m.spec = planner.BreakSpec{
			Kind:     planner.Long,
			StartAt:  now.Add(-time.Minute),
			EndAt:    now.Add(4 * time.Minute),
			Duration: 5 * time.Minute,
			Postpone: planner.PostponeConfig{Enabled: true, Limit: 1, Percent: 30, Time: 5 * time.Minute},
		}
		f.m.idea = ideas.Idea{
			Title: "Step away",
			Text:  "Walk to a window and look at something far away.",
		}
		f.m.phase = planner.Breaking
		writePNG(t, tt, filepath.Join(out, "break-long.png"))
	})
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
