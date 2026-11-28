package mark

import (
	"math"
	"testing"
)

func TestPointsOutlineTheCrescent(t *testing.T) {
	pts := Points(96)
	area := 0.0
	for i, j := 0, len(pts)-1; i < len(pts); j, i = i, i+1 {
		area += pts[j].X*pts[i].Y - pts[i].X*pts[j].Y
	}
	if got := math.Abs(area) / 2; math.Abs(got-96.4) > 0.5 {
		t.Errorf("crescent area = %.2f, want about 96.4", got)
	}
}

func TestRasterUsesTheSameCrescent(t *testing.T) {
	cases := []struct {
		name   string
		x, y   float64
		inside bool
	}{
		{"spark", SparkX, SparkY, true},
		{"lit edge", 9, 20, true},
		{"outer center", MoonX, MoonY, false},
		{"cut center", CutX, CutY, false},
		{"off canvas", 5, 5, false},
	}
	for _, tc := range cases {
		if got := Raster(tc.x, tc.y, Canvas, CutRadius, SparkRadius); got != tc.inside {
			t.Errorf("%s: Raster = %v, want %v", tc.name, got, tc.inside)
		}
	}

	// A smaller cut disc is how a small copy gains weight.
	x, y := CutX-6.64, CutY+4.27
	if Raster(x, y, Canvas, CutRadius, SparkRadius) {
		t.Error("base cut should leave this point out")
	}
	if !Raster(x, y, Canvas, 7.55, SparkRadius) {
		t.Error("a smaller cut should include this point")
	}
}
