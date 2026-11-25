// Command genicons draws the app icon in its light and dark appearances, the
// menu bar template icon, and compiles the app icon's asset catalog.
//
// The icon is the product's one mark: Fermata is the English name, and 停琴
// ("the lute stops") is its Chinese reading. The dot is the finger on the
// string; no extra line is drawn, because below the dot it reads as a face at
// 16–32 px. The menu bar keeps the same, bolder mark.
//
//	go run ./tools/genicons
package main

import (
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"

	markpkg "github.com/gythialy/fermata/internal/mark"
	"github.com/gythialy/fermata/internal/trayicon"
)

const (
	// iconSize is the app icon's side: macOS rounds it itself.
	iconSize = 1024
	// The Icon Composer document and the catalog actool compiles from it.
	iconDir = "icons/AppIcon.icon"
	carPath = "icons/AppIcon.car"
	// The mark sits in a 700-point box centered on the icon, above the
	// macOS 824-point safe area's edge; its ink is centered too, not just
	// its box, because the fermata is bottom-heavy.
	markSide = 700.0
	markLift = 2.6 // canvas units up, to center the ink

)

type vec struct{ x, y float64 }

// The sand appearance: a warm field, an ink or paper mark, one vermilion dot.
var (
	lightTop = color.NRGBA{R: 0xF2, G: 0xEA, B: 0xD9, A: 0xff}
	lightLow = color.NRGBA{R: 0xE6, G: 0xDA, B: 0xC2, A: 0xff}
	lightInk = color.NRGBA{R: 0x1B, G: 0x1A, B: 0x14, A: 0xff}
	lightDot = color.NRGBA{R: 0xC9, G: 0x4B, B: 0x33, A: 0xff}

	darkTop  = color.NRGBA{R: 0x2A, G: 0x24, B: 0x1C, A: 0xff}
	darkLow  = color.NRGBA{R: 0x16, G: 0x12, B: 0x10, A: 0xff}
	darkMark = color.NRGBA{R: 0xF3, G: 0xEE, B: 0xE2, A: 0xff}
	darkDot  = color.NRGBA{R: 0xF0, G: 0x83, B: 0x6C, A: 0xff}
)

func main() {
	if err := writeAppIcons(); err != nil {
		log.Fatal(err)
	}
	if err := writeTray(); err != nil {
		log.Fatal(err)
	}
	if err := compileCatalog(); err != nil {
		log.Printf("AppIcon.car: %v", err)
	}
}

// writeAppIcons writes the app icon, its two appearances as an Icon Composer
// document, and, where Xcode is installed, the asset catalog compiled from it.
func writeAppIcons() error {
	if err := save("resources/icon.png", previewIcon()); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(iconDir, "Assets"), 0o755); err != nil {
		return err
	}
	files := map[string]image.Image{
		"bg-light.png":   fieldLayer(true),
		"bg-dark.png":    fieldLayer(false),
		"mark-light.png": markLayer(true),
		"mark-dark.png":  markLayer(false),
	}
	for name, img := range files {
		if err := save(filepath.Join(iconDir, "Assets", name), img); err != nil {
			return err
		}
	}
	doc := map[string]any{
		"groups": []any{
			map[string]any{
				"layers": []any{
					map[string]any{
						"name":       "field",
						"image-name": "bg-light.png",
						"image-name-specializations": []any{
							map[string]any{"value": "bg-light.png"},
							map[string]any{"appearance": "dark", "value": "bg-dark.png"},
						},
					},
					map[string]any{
						"name":       "mark",
						"image-name": "mark-light.png",
						"image-name-specializations": []any{
							map[string]any{"value": "mark-light.png"},
							map[string]any{"appearance": "dark", "value": "mark-dark.png"},
						},
					},
				},
				"lighting":     "individual",
				"shadow":       map[string]any{"kind": "none", "opacity": 0.5},
				"translucency": map[string]any{"enabled": false, "value": 0.5},
			},
		},
		"supported-platforms": map[string]any{"squares": "shared"},
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(iconDir, "icon.json"), append(data, '\n'), 0o644); err != nil {
		return err
	}
	return compileCatalog()
}

// compileCatalog compiles the Icon Composer document with actool, which needs
// Xcode. The committed AppIcon.car lets a build without Xcode carry both
// appearances too.
func compileCatalog() error {
	if _, err := exec.LookPath("xcrun"); err != nil {
		return errNotXcode
	}
	src, err := filepath.Abs(iconDir)
	if err != nil {
		return err
	}
	out, err := os.MkdirTemp("", "mygo-icon")
	if err != nil {
		return err
	}
	defer os.RemoveAll(out)
	cmd := exec.Command("xcrun", "actool", src,
		"--compile", out,
		"--platform", "macosx",
		"--minimum-deployment-target", "13.0",
		"--app-icon", "AppIcon",
		"--target-device", "mac",
		"--output-partial-info-plist", filepath.Join(out, "partial.plist"),
		"--output-format", "human-readable-text",
	)
	if msg, err := cmd.CombinedOutput(); err != nil {
		return &actoolError{err: err, output: string(msg)}
	}
	data, err := os.ReadFile(filepath.Join(out, "Assets.car"))
	if err != nil {
		return err
	}
	dst, err := filepath.Abs(carPath)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}

type actoolError struct {
	err    error
	output string
}

func (e *actoolError) Error() string { return e.err.Error() + "\n" + e.output }

var errNotXcode = &missingTool{}

type missingTool struct{}

func (*missingTool) Error() string { return "Xcode is not installed; keeping the existing AppIcon.car" }

// previewIcon draws the icon as macOS shows it: the sand field and the mark
// inside the 824-point squircle, on transparent corners.
func previewIcon() image.Image {
	const margin, side, radius = 100.0, 824.0, 185.0
	return render(iconSize, func(p vec) color.NRGBA {
		if !inRound(p.x, p.y, margin, margin, side, radius) {
			return color.NRGBA{}
		}
		if c, ok := markAt(p, lightInk, lightDot); ok {
			return c
		}
		return fieldAt(p.y, lightTop, lightLow)
	})
}

// inRound reports whether x,y is inside a rounded square.
func inRound(x, y, left, top, side, radius float64) bool {
	if x < left || x > left+side || y < top || y > top+side {
		return false
	}
	cx := math.Min(math.Max(x, left+radius), left+side-radius)
	cy := math.Min(math.Max(y, top+radius), top+side-radius)
	return math.Hypot(x-cx, y-cy) <= radius
}

// fieldLayer draws the sand field alone, as the icon's background layer.
func fieldLayer(light bool) image.Image {
	top, low := lightTop, lightLow
	if !light {
		top, low = darkTop, darkLow
	}
	return render(iconSize, func(p vec) color.NRGBA { return fieldAt(p.y, top, low) })
}

// markLayer draws the fermata alone on transparency, as the icon's foreground
// layer.
func markLayer(light bool) image.Image {
	mark, dot := lightInk, lightDot
	if !light {
		mark, dot = darkMark, darkDot
	}
	return render(iconSize, func(p vec) color.NRGBA {
		if c, ok := markAt(p, mark, dot); ok {
			return c
		}
		return color.NRGBA{}
	})
}

// fieldAt is the vertical gradient of the field.
func fieldAt(y float64, top, low color.NRGBA) color.NRGBA {
	t := math.Max(0, math.Min(1, y/iconSize))
	return color.NRGBA{
		R: uint8(float64(top.R)*(1-t) + float64(low.R)*t),
		G: uint8(float64(top.G)*(1-t) + float64(low.G)*t),
		B: uint8(float64(top.B)*(1-t) + float64(low.B)*t),
		A: 0xff,
	}
}

// markAt colors a point of the fermata, and reports whether it is on it. The
// same 32-unit geometry the app draws is scaled into the icon's 700-point
// box, with the ink centered.
func markAt(p vec, mark, dot color.NRGBA) (color.NRGBA, bool) {
	scale := markSide / float64(markpkg.Canvas)
	ox := (iconSize - markSide) / 2
	oy := (iconSize - markSide) / 2
	lx := (p.x - ox) / scale
	ly := (p.y-oy)/scale + markLift
	if math.Hypot(lx-markpkg.Dot.X, ly-markpkg.Dot.Y) <= markpkg.DotRadius {
		return dot, true
	}
	points := markpkg.Points(64)
	for i := 1; i < len(points); i++ {
		if distanceToSegment(lx, ly, points[i-1], points[i]) <= markpkg.Stroke/2 {
			return mark, true
		}
	}
	return color.NRGBA{}, false
}

// writeTray writes the menu bar template icon, which the system tints.
func writeTray() error {
	return os.WriteFile("resources/tray.png", trayicon.Mark(44), 0o644)
}

// distanceToSegment is the distance from p to the segment a-b, in canvas
// units.
func distanceToSegment(x, y float64, a, b markpkg.Vec) float64 {
	dx, dy := b.X-a.X, b.Y-a.Y
	if dx == 0 && dy == 0 {
		return math.Hypot(x-a.X, y-a.Y)
	}
	t := ((x-a.X)*dx + (y-a.Y)*dy) / (dx*dx + dy*dy)
	t = math.Max(0, math.Min(1, t))
	return math.Hypot(a.X+t*dx-x, a.Y+t*dy-y)
}

// render samples draw in a grid per pixel and averages the premultiplied
// colors, which keeps the edges smooth without a color fringe.
func render(size int, sample func(p vec) color.NRGBA) *image.NRGBA {
	const samples = 3
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	for y := range size {
		for x := range size {
			var r, g, b, a float64
			for sy := range samples {
				for sx := range samples {
					p := vec{
						x: float64(x) + (float64(sx)+0.5)/samples,
						y: float64(y) + (float64(sy)+0.5)/samples,
					}
					c := sample(p)
					w := float64(c.A) / 255
					r += float64(c.R) * w
					g += float64(c.G) * w
					b += float64(c.B) * w
					a += float64(c.A)
				}
			}
			n := float64(samples * samples)
			alpha := a / n
			var out color.NRGBA
			if alpha > 0 {
				out = color.NRGBA{
					R: uint8(math.Min(255, r/n*255/alpha+0.5)),
					G: uint8(math.Min(255, g/n*255/alpha+0.5)),
					B: uint8(math.Min(255, b/n*255/alpha+0.5)),
					A: uint8(alpha + 0.5),
				}
			}
			img.SetNRGBA(x, y, out)
		}
	}
	return img
}

func save(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
