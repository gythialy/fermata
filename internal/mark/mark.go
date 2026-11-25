// Package mark draws the product's one graphic asset: the fermata, an arc
// with a dot under it. The app icon, the menu bar template, the break window,
// the welcome window and the settings sidebar all use this geometry, so none
// of them can drift from the others.
package mark

import (
	"math"

	"github.com/egoist/mygo/ui"
)

// The design's canvas, in which Arc, Stroke, Dot and DotRadius are written.
const Canvas = 32

// Vec is a point of the canvas.
type Vec struct{ X, Y float64 }

// Arc is the cubic that draws the top of the fermata.
var Arc = [4]Vec{{6, 18}, {8, 8.5}, {24, 8.5}, {26, 18}}

// Stroke is the arc's width on the canvas.
const Stroke = 2.2

// Dot is the center of the dot, and DotRadius its radius.
var Dot = Vec{16, 25}

const DotRadius = 2.6

// Points returns the arc as n+1 points along the cubic.
func Points(n int) []Vec {
	out := make([]Vec, 0, n+1)
	for i := range n + 1 {
		t := float64(i) / float64(n)
		mt := 1 - t
		out = append(out, Vec{
			X: mt*mt*mt*Arc[0].X + 3*mt*mt*t*Arc[1].X + 3*mt*t*t*Arc[2].X + t*t*t*Arc[3].X,
			Y: mt*mt*mt*Arc[0].Y + 3*mt*mt*t*Arc[1].Y + 3*mt*t*t*Arc[2].Y + t*t*t*Arc[3].Y,
		})
	}
	return out
}

// Paint draws the mark in r: the arc in arcColor and the dot in dotColor.
func Paint(p *ui.Painter, r ui.Rect, arcColor, dotColor ui.Color) {
	PaintWith(p, r, Stroke, DotRadius, arcColor, dotColor)
}

// PaintWith draws the mark with an optically chosen arc width and dot radius,
// in canvas units, as a small or a large copy needs.
func PaintWith(p *ui.Painter, r ui.Rect, stroke, dotRadius float32, arcColor, dotColor ui.Color) {
	side := min(r.W, r.H)
	if side <= 0 {
		return
	}
	scale := side / Canvas
	ox := r.X + (r.W-side)/2
	oy := r.Y + (r.H-side)/2
	path := (&ui.Path{}).
		MoveTo(ox+float32(Arc[0].X*float64(scale)), oy+float32(Arc[0].Y*float64(scale))).
		CubeTo(
			ox+float32(Arc[1].X*float64(scale)), oy+float32(Arc[1].Y*float64(scale)),
			ox+float32(Arc[2].X*float64(scale)), oy+float32(Arc[2].Y*float64(scale)),
			ox+float32(Arc[3].X*float64(scale)), oy+float32(Arc[3].Y*float64(scale)),
		)
	path.Cap(ui.RoundCap)
	p.StrokePath(path, stroke*scale, arcColor)

	radius := dotRadius * scale
	cx := ox + float32(Dot.X*float64(scale))
	cy := oy + float32(Dot.Y*float64(scale))
	p.FillPath((&ui.Path{}).Circle(cx, cy, radius), dotColor)
}

// Raster reports whether a point of a size×size image is on the mark, for
// the menu bar's template image: the arc's distance to the canvas' curve,
// and the dot's circle.
func Raster(x, y, size, stroke, dotRadius float64) bool {
	scale := size / Canvas
	cx, cy := x/scale, y/scale
	if math.Hypot(cx-Dot.X, cy-Dot.Y) <= dotRadius {
		return true
	}
	points := Points(64)
	half := stroke / 2
	for i := 1; i < len(points); i++ {
		if distanceToSegment(cx, cy, points[i-1], points[i]) <= half {
			return true
		}
	}
	return false
}

func distanceToSegment(x, y float64, a, b Vec) float64 {
	dx, dy := b.X-a.X, b.Y-a.Y
	if dx == 0 && dy == 0 {
		return math.Hypot(x-a.X, y-a.Y)
	}
	t := ((x-a.X)*dx + (y-a.Y)*dy) / (dx*dx + dy*dy)
	t = min(max(t, 0), 1)
	return math.Hypot(a.X+t*dx-x, a.Y+t*dy-y)
}
