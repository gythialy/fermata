// Command genicons draws the app icon in its light and dark appearances, the
// menu bar template icon, and compiles the app icon's asset catalog.
//
// The app icon follows the name's other reading: 停琴伫凉月 from 停琴, a
// crescent held above one point of light. The same moon is the in-app mark,
// drawn by internal/mark; the desktop icon is its largest copy.
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

	// Geometry comes from internal/mark so the desktop icon cannot drift.
	moonX = markpkg.MoonX * iconSize / markpkg.Canvas
	moonY = markpkg.MoonY * iconSize / markpkg.Canvas
	moonR = markpkg.MoonRadius * iconSize / markpkg.Canvas
	cutX  = markpkg.CutX * iconSize / markpkg.Canvas
	cutY  = markpkg.CutY * iconSize / markpkg.Canvas
	cutR  = markpkg.CutRadius * iconSize / markpkg.Canvas
	// The point of light sits outside the crescent, inside the safe area.
	sparkX = markpkg.SparkX * iconSize / markpkg.Canvas
	sparkY = markpkg.SparkY * iconSize / markpkg.Canvas
	sparkR = markpkg.SparkRadius * iconSize / markpkg.Canvas
)

type vec struct{ x, y float64 }

// The two appearances invert the ink so one crescent keeps contrast on both
// menu-bar shades; dark mode keeps the night scene itself.
type palette struct {
	field, moon, spark color.NRGBA
}

var (
	lightIcon = palette{
		field: color.NRGBA{R: 0xE7, G: 0xEA, B: 0xF0, A: 0xff},
		moon:  color.NRGBA{R: 0x1B, G: 0x23, B: 0x3A, A: 0xff},
		spark: color.NRGBA{R: 0xB8, G: 0x43, B: 0x2F, A: 0xff},
	}
	darkIcon = palette{
		field: color.NRGBA{R: 0x16, G: 0x1B, B: 0x2A, A: 0xff},
		moon:  color.NRGBA{R: 0xF3, G: 0xEE, B: 0xE2, A: 0xff},
		spark: color.NRGBA{R: 0xF0, G: 0x83, B: 0x6C, A: 0xff},
	}
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
	if err := save("resources/icon.png", previewIcon(lightIcon)); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(iconDir, "Assets"), 0o755); err != nil {
		return err
	}
	files := map[string]image.Image{
		"bg-light.png":   fieldLayer(lightIcon),
		"bg-dark.png":    fieldLayer(darkIcon),
		"mark-light.png": moonLayer(lightIcon),
		"mark-dark.png":  moonLayer(darkIcon),
	}
	for name, img := range files {
		if err := save(filepath.Join(iconDir, "Assets", name), img); err != nil {
			return err
		}
	}
	doc := map[string]any{
		"groups": []any{
			map[string]any{
				// Layers render front to back: the mark sits above the field.
				// The other order draws the field over the mark, and the app
				// icon shows as an empty square.
				"layers": []any{
					map[string]any{
						"name":       "mark",
						"image-name": "mark-light.png",
						"image-name-specializations": []any{
							map[string]any{"value": "mark-light.png"},
							map[string]any{"appearance": "dark", "value": "mark-dark.png"},
						},
					},
					map[string]any{
						"name":       "field",
						"image-name": "bg-light.png",
						"image-name-specializations": []any{
							map[string]any{"value": "bg-light.png"},
							map[string]any{"appearance": "dark", "value": "bg-dark.png"},
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

// previewIcon draws the icon as macOS shows it: the field and the moon inside
// the 824-point squircle, on transparent corners.
func previewIcon(pal palette) image.Image {
	const margin, side, radius = 100.0, 824.0, 185.0
	return render(iconSize, func(p vec) color.NRGBA {
		if !inRound(p.x, p.y, margin, margin, side, radius) {
			return color.NRGBA{}
		}
		if c, ok := moonAt(p, pal); ok {
			return c
		}
		return pal.field
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

// fieldLayer draws the field alone, as the icon's background layer.
func fieldLayer(pal palette) image.Image {
	return render(iconSize, func(p vec) color.NRGBA { return pal.field })
}

// moonLayer draws the crescent and its point of light on transparency, as the
// icon's foreground layer.
func moonLayer(pal palette) image.Image {
	return render(iconSize, func(p vec) color.NRGBA {
		if c, ok := moonAt(p, pal); ok {
			return c
		}
		return color.NRGBA{}
	})
}

// moonAt colors a point of the icon's graphic and reports whether it is on it.
// The cut disc is subtracted from the paper disc, leaving a crescent.
func moonAt(p vec, pal palette) (color.NRGBA, bool) {
	if math.Hypot(p.x-sparkX, p.y-sparkY) <= sparkR {
		return pal.spark, true
	}
	if math.Hypot(p.x-moonX, p.y-moonY) > moonR {
		return color.NRGBA{}, false
	}
	if math.Hypot(p.x-cutX, p.y-cutY) <= cutR {
		return color.NRGBA{}, false
	}
	return pal.moon, true
}

// writeTray writes the 18 pt menu bar template at 2×, matching the runtime.
func writeTray() error {
	return os.WriteFile("resources/tray.png", trayicon.Mark(36), 0o644)
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
