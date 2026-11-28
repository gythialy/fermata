// Package mark draws the product's one graphic asset: the cool moon, a
// crescent and a spark. The app icon, menu bar template, break window,
// welcome window and settings sidebar use this geometry, so none can drift.
package mark

import (
	"math"

	"github.com/egoist/mygo/ui"
)

// Canvas is the design canvas the geometry is written in.
const Canvas = 32

// Vec is a point of the canvas.
type Vec struct{ X, Y float64 }

// The crescent is the moon disc minus the cut disc; the spark stays outside
// both discs. Small copies tune the cut and spark radii, never the centers.
const (
	MoonX, MoonY   = 15.375, 15.625
	MoonRadius     = 8.5
	CutX, CutY     = 19.75, 12.8125
	CutRadius      = 8.0625
	SparkX, SparkY = 22.8125, 22.1875
	SparkRadius    = 1.6875
)

// arcSamples keeps the polygon's edges below a pixel at in-app sizes.
const arcSamples = 96

var basePolygon = crescent(CutRadius, arcSamples)

// Points returns the sampled crescent boundary at canvas scale.
func Points(n int) []Vec {
	if n < 1 {
		n = 1
	}
	return crescent(CutRadius, n)
}

// Paint draws the moon in r, with the default cut and spark radii.
func Paint(p *ui.Painter, r ui.Rect, moon, spark ui.Color) {
	PaintWith(p, r, CutRadius, SparkRadius, moon, spark)
}

// PaintWith draws the moon with optically chosen cut and spark radii, in
// canvas units, for a small or a large copy.
func PaintWith(p *ui.Painter, r ui.Rect, cutRadius, sparkRadius float32, moon, spark ui.Color) {
	side := min(r.W, r.H)
	if side <= 0 {
		return
	}
	scale := float64(side) / Canvas
	ox := float64(r.X) + (float64(r.W)-float64(side))/2
	oy := float64(r.Y) + (float64(r.H)-float64(side))/2

	path := &ui.Path{}
	for i, v := range polygonFor(float64(cutRadius)) {
		x, y := ox+v.X*scale, oy+v.Y*scale
		if i == 0 {
			path.MoveTo(float32(x), float32(y))
		} else {
			path.LineTo(float32(x), float32(y))
		}
	}
	path.Close()
	p.FillPath(path, moon)

	cx, cy := ox+SparkX*scale, oy+SparkY*scale
	p.FillPath((&ui.Path{}).Circle(float32(cx), float32(cy), float32(float64(sparkRadius)*scale)), spark)
}

// Raster reports whether a point of a size×size image is on the moon, for
// the menu bar's template image. It uses the same sampled crescent as Paint.
func Raster(x, y, size, cutRadius, sparkRadius float64) bool {
	scale := size / Canvas
	cx, cy := x/scale, y/scale
	if math.Hypot(cx-SparkX, cy-SparkY) <= sparkRadius {
		return true
	}
	return insidePolygon(cx, cy, polygonFor(cutRadius))
}

func polygonFor(cutRadius float64) []Vec {
	if cutRadius == CutRadius {
		return basePolygon
	}
	return crescent(cutRadius, arcSamples)
}

type disc struct{ x, y, r float64 }

// crescent walks the outer arc forward, then the cut arc backward, so the
// boundary is a single non-self-intersecting polygon without a field-colored
// patch over the cut disc.
func crescent(cutRadius float64, n int) []Vec {
	outer := disc{MoonX, MoonY, MoonRadius}
	cut := disc{CutX, CutY, cutRadius}
	a, b := intersections(outer, cut)

	outline := arc(outer, b, a, true, n)
	inner := arc(cut, a, b, false, n)
	return append(outline, inner[1:len(inner)-1]...)
}

func intersections(a, b disc) (Vec, Vec) {
	dx, dy := b.x-a.x, b.y-a.y
	d := math.Hypot(dx, dy)
	ux, uy := dx/d, dy/d
	along := (a.r*a.r - b.r*b.r + d*d) / (2 * d)
	across := math.Sqrt(max(0, a.r*a.r-along*along))
	px, py := a.x+along*ux, a.y+along*uy
	return Vec{px + across*uy, py - across*ux}, Vec{px - across*uy, py + across*ux}
}

// arc sweeps from one intersection to the other in the requested direction.
func arc(c disc, from, to Vec, increasing bool, n int) []Vec {
	start, end := math.Atan2(from.Y-c.y, from.X-c.x), math.Atan2(to.Y-c.y, to.X-c.x)
	if increasing {
		for end <= start {
			end += 2 * math.Pi
		}
	} else {
		for end >= start {
			end -= 2 * math.Pi
		}
	}
	out := make([]Vec, 0, n+1)
	for i := range n + 1 {
		angle := start + (end-start)*float64(i)/float64(n)
		out = append(out, Vec{c.x + c.r*math.Cos(angle), c.y + c.r*math.Sin(angle)})
	}
	return out
}

func insidePolygon(x, y float64, pts []Vec) bool {
	inside := false
	for i, j := 0, len(pts)-1; i < len(pts); j, i = i, i+1 {
		a, b := pts[j], pts[i]
		if (a.Y > y) != (b.Y > y) && x < a.X+(y-a.Y)*(b.X-a.X)/(b.Y-a.Y) {
			inside = !inside
		}
	}
	return inside
}
