// Package trayicon draws the menu bar icon: the fermata by default, two bars
// while breaks are paused, a progress ring for the rhythm. The images are
// template images: black with alpha, which the system tints.
package trayicon

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
	"sync"
)

// canvas is the design's 18-point menu bar canvas.
const canvas = 18.0

// shots is how many samples each pixel gets on a side, for smooth edges.
const shots = 6

// The fermata at menu bar size is bolder than the printed mark: its arc is
// 2.5 points wide and its dot 4.4 points across, so an 18-point template
// still reads in a crowded bar. The curves are the design's.
var (
	arc   = [4]point{{1.7, 10.4}, {4.1, 2.7}, {13.9, 2.7}, {16.3, 10.4}}
	dot   = point{9, 14.5}
	dotR  = 2.2
	arcW  = 2.5
	curve = bezier(arc[0], arc[1], arc[2], arc[3], 48)
)

type kind uint8

const (
	kindMark kind = iota
	kindPaused
	kindRing
)

type point struct{ x, y float64 }

type key struct {
	kind     kind
	size     int
	progress int // percent, so that a cache entry covers one step
}

var cache struct {
	sync.Mutex
	images map[key][]byte
}

// Mark returns the fermata.
func Mark(size int) []byte { return cached(kindMark, size, 0) }

// Paused returns the two bars that mean a pause.
func Paused(size int) []byte { return cached(kindPaused, size, 0) }

// Ring returns a progress ring, with progress from 0 to 1 drawn in full ink
// over a faint track.
func Ring(size int, progress float64) []byte {
	p := min(max(int(math.Round(progress*100)), 0), 100)
	return cached(kindRing, size, p)
}

func cached(k kind, size, progress int) []byte {
	ck := key{kind: k, size: size, progress: progress}
	cache.Lock()
	defer cache.Unlock()
	if cache.images == nil {
		cache.images = map[key][]byte{}
	}
	if data, ok := cache.images[ck]; ok {
		return data
	}
	data := encode(ck)
	cache.images[ck] = data
	return data
}

func encode(k key) []byte {
	var buf bytes.Buffer
	if err := png.Encode(&buf, draw(k)); err != nil {
		return nil
	}
	return buf.Bytes()
}

func draw(k key) image.Image {
	size := max(k.size, 1)
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	scale := float64(size) / canvas
	used := 2 * math.Pi * float64(k.progress) / 100
	for y := range size {
		for x := range size {
			alpha := 0.0
			for sy := range shots {
				for sx := range shots {
					px := (float64(x) + (float64(sx)+0.5)/shots) / scale
					py := (float64(y) + (float64(sy)+0.5)/shots) / scale
					alpha += alphaAt(k, px, py, used)
				}
			}
			alpha /= shots * shots
			if alpha <= 0 {
				continue
			}
			img.SetNRGBA(x, y, color.NRGBA{A: uint8(math.Round(min(alpha, 1) * 255))})
		}
	}
	return img
}

func alphaAt(k key, x, y, used float64) float64 {
	switch k.kind {
	case kindMark:
		if math.Hypot(x-dot.x, y-dot.y) <= dotR {
			return 1
		}
		for i := 1; i < len(curve); i++ {
			if distanceToSegment(x, y, curve[i-1], curve[i]) <= arcW/2 {
				return 1
			}
		}
	case kindPaused:
		for _, cx := range []float64{6.4, 11.6} {
			if roundedRect(x, y, cx, 9, 0.85, 3.7, 0.8) {
				return 1
			}
		}
	case kindRing:
		r := math.Hypot(x-9, y-9)
		switch {
		case r >= 6.5-0.8 && r <= 6.5+0.8:
			angle := math.Atan2(x-9, -(y - 9)) // clockwise from the top
			if angle < 0 {
				angle += 2 * math.Pi
			}
			if angle <= used {
				return 1
			}
			return 0.25
		case r <= 1.3:
			return 1
		}
	}
	return 0
}

// bezier returns points along a cubic Bézier curve.
func bezier(p0, p1, p2, p3 point, n int) []point {
	out := make([]point, 0, n+1)
	for i := range n + 1 {
		t := float64(i) / float64(n)
		mt := 1 - t
		out = append(out, point{
			x: mt*mt*mt*p0.x + 3*mt*mt*t*p1.x + 3*mt*t*t*p2.x + t*t*t*p3.x,
			y: mt*mt*mt*p0.y + 3*mt*mt*t*p1.y + 3*mt*t*t*p2.y + t*t*t*p3.y,
		})
	}
	return out
}

func distanceToSegment(x, y float64, a, b point) float64 {
	dx, dy := b.x-a.x, b.y-a.y
	if dx == 0 && dy == 0 {
		return math.Hypot(x-a.x, y-a.y)
	}
	t := ((x-a.x)*dx + (y-a.y)*dy) / (dx*dx + dy*dy)
	t = min(max(t, 0), 1)
	return math.Hypot(a.x+t*dx-x, a.y+t*dy-y)
}

// roundedRect reports whether (x, y) is inside a rounded rectangle centered
// at (cx, cy) with half extents hw and hh.
func roundedRect(x, y, cx, cy, hw, hh, r float64) bool {
	qx := math.Abs(x-cx) - (hw - r)
	qy := math.Abs(y-cy) - (hh - r)
	d := math.Hypot(max(qx, 0), max(qy, 0)) + min(max(qx, qy), 0) - r
	return d <= 0
}
