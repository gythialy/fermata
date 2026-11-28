package theme

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/gythialy/fermata/internal/settings"
)

func TestDeeperUsesTheDesignValues(t *testing.T) {
	cases := []struct {
		field string
		want  string
	}{
		{settings.FieldPine, "#1B322C"},
		{settings.FieldInk, "#0E1311"},
		{settings.FieldOxblood, "#351D20"},
		{settings.FieldSlate, "#20282D"},
		{settings.FieldSand, "#C2B7A3"},
		{settings.FieldPaper, "#E6E4DA"},
	}
	for _, c := range cases {
		if got := Deeper(c.field, ui.Hex("#000000")); got != ui.Hex(c.want) {
			t.Errorf("Deeper(%s) = %v, want %s", c.field, got, c.want)
		}
	}
}

func TestDeeperMixesACustomField(t *testing.T) {
	base := ui.Hex("#808080")
	if got, want := Deeper(settings.FieldCustom, base), base.Mix(ui.Hex("#000000"), 0.12); got != want {
		t.Errorf("Deeper(custom) = %v, want %v", got, want)
	}
}

func TestAppIconAndMarkPalettes(t *testing.T) {
	for _, tc := range []struct {
		dark            bool
		bg, moon, spark ui.Color
	}{
		{false, ui.Hex("#E7EAF0"), ui.Hex("#1B233A"), ui.Hex("#B8432F")},
		{true, ui.Hex("#161B2A"), ui.Hex("#F3EEE2"), ui.Hex("#F0836C")},
	} {
		bg, moon, spark := AppIcon(tc.dark)
		if bg != tc.bg || moon != tc.moon || spark != tc.spark {
			t.Errorf("AppIcon(%v) = %v, %v, %v; want %v, %v, %v", tc.dark, bg, moon, spark, tc.bg, tc.moon, tc.spark)
		}
		wantMoon, wantSpark := ink, accentLight
		if tc.dark {
			wantMoon, wantSpark = paper, accentDark
		}
		gotMoon, gotSpark := Mark(bg)
		if gotMoon != wantMoon || gotSpark != wantSpark {
			t.Errorf("Mark(%v) = %v, %v; want %v, %v", bg, gotMoon, gotSpark, wantMoon, wantSpark)
		}
	}
}
