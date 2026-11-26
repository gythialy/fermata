//go:build screenshots

package breakwin

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gythialy/fermata/internal/fonts"
	"github.com/gythialy/fermata/internal/planner"
)

// animationFPS is how many frames a second the clip is rendered at. The GIF
// halves it; the landing page's video keeps all of it.
const animationFPS = 24

// animationLength is how long the clip runs: the content's entrance, the
// countdown and the hairline, and the exit changing from Postpone to Skip.
const animationLength = 4400 * time.Millisecond

// animationDir is where the frames go. It sits in build/, which git ignores;
// tools/screens/animate.sh encodes it into docs/images.
var animationDir = filepath.Join("..", "..", "build", "screens", "break-loop")

// TestAnimation renders the break window's motion into frames, once per
// language, for the README's loop and the landing page's video. Time is
// simulated through ui.Tester's Advance, so every run draws the same clip and
// nothing is recorded from a screen:
//
//	make animation
func TestAnimation(t *testing.T) {
	if err := fonts.Load(); err != nil {
		t.Fatal(err)
	}
	for _, shot := range shotLocales {
		t.Run(shot.locale, func(t *testing.T) {
			useLocale(t, shot.locale)
			out := filepath.Join(animationDir, shot.locale)
			if err := os.RemoveAll(out); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(out, 0o755); err != nil {
				t.Fatal(err)
			}

			f, tt := newFixture(t)
			now := tt.Now()
			f.m.spec = planner.BreakSpec{
				Kind:     planner.Mini,
				StartAt:  now.Add(-4 * time.Second),
				EndAt:    now.Add(16 * time.Second),
				Duration: 20 * time.Second,
				Postpone: planner.PostponeConfig{Enabled: true, Limit: 1, Percent: 30, Time: 5 * time.Minute},
			}
			f.m.phase = planner.Breaking
			f.m.opts.Label = breakLabel
			f.m.opts.ActionLabel = actionLabel
			f.m.idea = shotIdeas[shot.locale].mini
			// Start with the content still rising, the way the window opens. The
			// panel's own fade and its closing belong to the window, not the
			// view, so they are not part of this render.
			f.m.animate = true
			f.m.shownAt = now
			tt.SetScale(2)

			step := time.Second / animationFPS
			frames := 0
			for elapsed := time.Duration(0); elapsed < animationLength; elapsed += step {
				tt.Frame()
				if frames == 0 && !tt.HasText(actionLabel(Postpone)) {
					t.Fatalf("the clip does not start on a Postpone exit: %v", tt.Texts())
				}
				writeImage(t, cutCorners(tt.Image(), 2), filepath.Join(out, fmt.Sprintf("f%04d.png", frames)))
				frames++
				tt.Advance(step)
			}
			// Without this the loop would show nothing that a still cannot: the
			// exit has to pass from Postpone to Skip while the hairline fills.
			if !tt.HasText(actionLabel(Skip)) {
				t.Errorf("the clip never reaches the Skip exit: %v", tt.Texts())
			}
			t.Logf("%s: %d frames, %v each", shot.locale, frames, step)
		})
	}
}
