// Package theme builds the Held Note theme: paper and ink neutrals, one
// vermilion signal, and the break field's six presets.
package theme

import (
	"math"

	"github.com/egoist/mygo/ui"
	markpkg "github.com/gythialy/fermata/internal/mark"
	"github.com/gythialy/fermata/internal/settings"
)

// The tokens of the design system, as the app uses them.
var (
	accentLight = ui.Hex("#B8432F")
	accentDark  = ui.Hex("#F0836C")
	paper       = ui.Hex("#F3F2EA")
	ink         = ui.Hex("#171A17")
	moonLight   = ui.Hex("#1B233A")
	moonDark    = ui.Hex("#F3EEE2")
)

// Apply themes the window that c builds and returns the theme.
func Apply(c *ui.Context) *ui.Theme {
	var t ui.Theme
	if c.Theme().Dark {
		t = *dark()
	} else {
		t = *light()
	}
	t = *t.WithPreferences(c.Preferences())
	t.Radius = 8
	t.Spacing = 4
	t.FontSize = 13 * Scale(c)
	c.SetTheme(&t)
	return &t
}

func light() *ui.Theme {
	t := *ui.LightTheme()
	t.Background = ui.Hex("#F6F4ED")
	t.Surface = ui.Hex("#EAE8DF")
	t.SurfaceHover = ui.Hex("#E2E0D7")
	t.SurfacePressed = ui.Hex("#DAD8CE")
	t.Border = ui.Hex("#D6D4CA")
	t.Text = ui.Hex("#171A17")
	t.TextMuted = ui.Hex("#5D635B")
	t.Accent = accentLight
	t.AccentHover = ui.Hex("#A63A28")
	t.AccentPressed = ui.Hex("#8F3223")
	t.AccentText = ui.Hex("#FFFFFF")
	t.Danger = ui.Hex("#B3261E")
	t.Warning = ui.Hex("#8A5A00")
	t.Success = ui.Hex("#3E5F4E")
	t.Selection = ui.Hex("#F6D9D2")
	t.Focus = ui.Hex("#B8432F").Alpha(0.55)
	t.FocusRingWidth = 2
	t.FocusRingOffset = 2
	t.Inverse = ui.Hex("#171A17")
	t.InverseText = ui.Hex("#F6F4ED")
	return &t
}

func dark() *ui.Theme {
	t := *ui.DarkTheme()
	t.Background = ui.Hex("#121513")
	t.Surface = ui.Hex("#1B211D")
	t.SurfaceHover = ui.Hex("#232A25")
	t.SurfacePressed = ui.Hex("#2B322C")
	t.Border = ui.Hex("#2E342F")
	t.Text = ui.Hex("#ECEDE5")
	t.TextMuted = ui.Hex("#A8AEA4")
	t.Accent = accentDark
	t.AccentHover = ui.Hex("#F49A84")
	t.AccentPressed = ui.Hex("#F7B1A0")
	t.AccentText = ui.Hex("#451009")
	t.Danger = ui.Hex("#F2B8B5")
	t.Warning = ui.Hex("#F0C36A")
	t.Success = ui.Hex("#A9C8B4")
	t.Selection = ui.Hex("#5C2418")
	t.Focus = ui.Hex("#E2674E").Alpha(0.60)
	t.FocusRingWidth = 2
	t.FocusRingOffset = 2
	t.Inverse = ui.Hex("#ECEDE5")
	t.InverseText = ui.Hex("#121513")
	return &t
}

// Field returns the break field's background for the settings, and the
// foreground that has contrast on it: ink on the light presets, paper on the
// dark ones.
func Field(bw settings.BreakWindow, darkAppearance bool) (bg, fg ui.Color) {
	bg = fieldColor(bw, darkAppearance)
	if Luminance(bg) > 0.5 {
		return bg, ink
	}
	return bg, paper
}

// longFields are the long break's deepen values, one step under each preset
// in the design system. A custom field has no named value, so it is mixed.
var longFields = map[string]ui.Color{
	settings.FieldPine:    ui.Hex("#1B322C"),
	settings.FieldInk:     ui.Hex("#0E1311"),
	settings.FieldOxblood: ui.Hex("#351D20"),
	settings.FieldSlate:   ui.Hex("#20282D"),
	settings.FieldSand:    ui.Hex("#C2B7A3"),
	settings.FieldPaper:   ui.Hex("#E6E4DA"),
}

// Deeper returns the long break's version of a field color. The presets use
// the design system's deepen values; a custom field has none, so it is mixed.
func Deeper(field string, bg ui.Color) ui.Color {
	if c, ok := longFields[field]; ok {
		return c
	}
	return bg.Mix(ui.Hex("#000000"), 0.12)
}

// AppIcon returns the cool moon's colors for an appearance: field, moon and
// spark. Both appearances come from the icon's palette.
func AppIcon(dark bool) (bg, moon, spark ui.Color) {
	if dark {
		return ui.Hex("#161B2A"), moonDark, accentDark
	}
	return ui.Hex("#E7EAF0"), moonLight, accentLight
}

// PaintMark draws the moon in r. Small copies shrink the cut and enlarge the
// spark so the crescent keeps its weight in the sidebar and the previews.
func PaintMark(p *ui.Painter, r ui.Rect, moon, spark ui.Color) {
	side := min(r.W, r.H)
	cut, radius := float32(markpkg.CutRadius), float32(markpkg.SparkRadius)
	switch {
	case side < 24:
		cut, radius = 7.55, 2.25
	case side < 32:
		cut, radius = 7.75, 2.05
	case side < 64:
		cut, radius = 7.95, 1.9
	}
	markpkg.PaintWith(p, r, cut, radius, moon, spark)
}

// PaintAppIconMark draws the app icon's moon, at any size.
func PaintAppIconMark(p *ui.Painter, r ui.Rect, moon, spark ui.Color) {
	PaintMark(p, r, moon, spark)
}

// Mark returns ink and vermilion on a light field, paper and coral on a dark
// one: the moon keeps contrast whatever the break field is.
func Mark(bg ui.Color) (moon, spark ui.Color) {
	if Luminance(bg) > 0.5 {
		return ink, accentLight
	}
	return paper, accentDark
}

func fieldColor(bw settings.BreakWindow, darkAppearance bool) ui.Color {
	switch bw.Field {
	case settings.FieldInk:
		return ui.Hex("#141A18")
	case settings.FieldOxblood:
		if darkAppearance {
			return ui.Hex("#43262A")
		}
		return ui.Hex("#4A2A2E")
	case settings.FieldSlate:
		if darkAppearance {
			return ui.Hex("#2A3339")
		}
		return ui.Hex("#2E383F")
	case settings.FieldSand:
		return ui.Hex("#CFC5B2")
	case settings.FieldPaper:
		return ui.Hex("#EFEEE6")
	case settings.FieldCustom:
		if c, ok := parseHex(bw.CustomField); ok {
			return c
		}
		fallthrough
	default:
		if darkAppearance {
			return ui.Hex("#24413A")
		}
		return ui.Hex("#2B4A41")
	}
}

// Luminance returns how light a color is, from 0 for black to 1 for white, as
// WCAG's relative luminance.
func Luminance(c ui.Color) float64 {
	channel := func(v uint8) float64 {
		x := float64(v) / 255
		if x <= 0.04045 {
			return x / 12.92
		}
		return math.Pow((x+0.055)/1.055, 2.4)
	}
	return 0.2126*channel(c.R) + 0.7152*channel(c.G) + 0.0722*channel(c.B)
}

// parseHex reads #rrggbb, the only form the settings keep.
func parseHex(s string) (ui.Color, bool) {
	if len(s) != 7 || s[0] != '#' {
		return ui.Color{}, false
	}
	for _, c := range s[1:] {
		if !('0' <= c && c <= '9') && !('a' <= c && c <= 'f') && !('A' <= c && c <= 'F') {
			return ui.Color{}, false
		}
	}
	return ui.Hex(s), true
}

// ApplyFlyout themes the status flyout. A menu bar utility has no inactive
// document to fade for, and ui.PrimaryButton would otherwise draw its one
// action as a plain button in a flyout that never takes the keyboard.
func ApplyFlyout(c *ui.Context) *ui.Theme {
	t := Apply(c)
	t.InactiveAccent = t.Accent
	c.SetTheme(t)
	return t
}

// Font families. The UI keeps the system's sans so that each platform's
// controls feel native; the break voice is the two bundled serifs, with the
// system's serifs only as a last fallback.
const (
	Serif = `Spectral, "Noto Serif SC", "Iowan Old Style", "Songti SC", "SimSun", serif`
	Mono  = `"SF Mono", "Cascadia Mono", "JetBrains Mono", monospace`
)

// Scale is the system's text scale, from 1 to 1.5. MyGo's controls apply it
// to their default size; the app's own text passes through Px so that both
// grow together, and 200% settings no longer leave the app at 100%.
func Scale(c *ui.Context) float32 {
	s := c.Preferences().TextScale
	if s < 1 || s != s {
		return 1
	}
	return min(s, 1.5)
}

// Px scales a design size or line height by the system's text scale and
// rounds it to half a point.
func Px(c *ui.Context, v float32) float32 {
	return float32(math.Round(float64(v*Scale(c))*2)) / 2
}
