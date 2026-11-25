// Package icons draws the interface's small glyphs — plus, minus, close,
// more, check and chevron — as paths on a 16-unit grid. The design draws its
// own icons instead of borrowing Unicode characters, so the stroke weight
// and the optical size stay the same on every platform.
package icons

import (
	"math"

	"github.com/egoist/mygo/ui"
)

// Box returns the square an icon of side s should occupy inside r, centered:
// the glyph keeps a 1:1 aspect whatever the control's shape.
func Box(r ui.Rect, s float32) ui.Rect {
	s = min(s, min(r.W, r.H))
	return ui.Rect{X: r.X + (r.W-s)/2, Y: r.Y + (r.H-s)/2, W: s, H: s}
}

// Minus draws the horizontal bar.
func Minus(p *ui.Painter, r ui.Rect, c ui.Color) {
	line(p, r, 0.25, 0.5, 0.75, 0.5, c)
}

// Plus draws the cross.
func Plus(p *ui.Painter, r ui.Rect, c ui.Color) {
	line(p, r, 0.25, 0.5, 0.75, 0.5, c)
	line(p, r, 0.5, 0.25, 0.5, 0.75, c)
}

// Close draws the cross of a close button.
func Close(p *ui.Painter, r ui.Rect, c ui.Color) {
	line(p, r, 0.28, 0.28, 0.72, 0.72, c)
	line(p, r, 0.72, 0.28, 0.28, 0.72, c)
}

// More draws the three dots of a more button.
func More(p *ui.Painter, r ui.Rect, c ui.Color) {
	side := min(r.W, r.H)
	radius := side * 0.07
	for _, x := range []float32{0.32, 0.5, 0.68} {
		cx := r.X + r.W*x
		cy := r.Y + r.H*0.5
		p.Fill(ui.Rect{X: cx - radius, Y: cy - radius, W: radius * 2, H: radius * 2}, c, radius)
	}
}

// Check draws a checkmark.
func Check(p *ui.Painter, r ui.Rect, c ui.Color) {
	line(p, r, 0.24, 0.52, 0.43, 0.71, c)
	line(p, r, 0.43, 0.71, 0.78, 0.31, c)
}

// Chevron draws a downward or upward chevron.
func Chevron(p *ui.Painter, r ui.Rect, c ui.Color, up bool) {
	if up {
		line(p, r, 0.3, 0.62, 0.5, 0.4, c)
		line(p, r, 0.5, 0.4, 0.7, 0.62, c)
		return
	}
	line(p, r, 0.3, 0.38, 0.5, 0.6, c)
	line(p, r, 0.5, 0.6, 0.7, 0.38, c)
}

// line draws the segment between two points of the 0–1 grid with round caps.
func line(p *ui.Painter, r ui.Rect, x0, y0, x1, y1 float32, c ui.Color) {
	width := max(1.4, min(r.W, r.H)*0.1)
	half := width / 2
	ax, ay := r.X+r.W*x0, r.Y+r.H*y0
	bx, by := r.X+r.W*x1, r.Y+r.H*y1
	dx, dy := bx-ax, by-ay
	length := float32(math.Hypot(float64(dx), float64(dy)))
	if length == 0 {
		return
	}
	nx, ny := -dy/length*half, dx/length*half
	path := &ui.Path{}
	path.MoveTo(ax+nx, ay+ny)
	path.LineTo(bx+nx, by+ny)
	path.LineTo(bx-nx, by-ny)
	path.LineTo(ax-nx, ay-ny)
	path.Close()
	p.FillPath(path, c)
	p.Fill(ui.Rect{X: ax - half, Y: ay - half, W: half * 2, H: half * 2}, c, half)
	p.Fill(ui.Rect{X: bx - half, Y: by - half, W: half * 2, H: half * 2}, c, half)
}
