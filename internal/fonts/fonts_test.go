package fonts

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"unicode"

	"golang.org/x/image/font/sfnt"
)

// stringsIn returns every string value of a JSON document, so that a test can
// cover exactly the text the serif draws.
func stringsIn(t *testing.T, path string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatal(err)
	}
	var out []string
	var walk func(any)
	walk = func(v any) {
		switch v := v.(type) {
		case string:
			if !strings.HasPrefix(v, "@") {
				out = append(out, v)
			}
		case []any:
			for _, item := range v {
				walk(item)
			}
		case map[string]any:
			for key, item := range v {
				if strings.HasPrefix(key, "@") {
					continue
				}
				walk(item)
			}
		}
	}
	walk(v)
	return out
}

func covers(t *testing.T, file string, texts []string, latin bool) {
	t.Helper()
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	font, err := sfnt.Parse(data)
	if err != nil {
		t.Fatalf("%s: %v", file, err)
	}
	var buf sfnt.Buffer
	seen := map[rune]bool{}
	for _, s := range texts {
		for _, r := range s {
			if seen[r] || unicode.IsSpace(r) {
				continue
			}
			// The English catalog names the Chinese language, which the
			// system sans draws; the Latin serif need not cover Han.
			if latin && unicode.Is(unicode.Han, r) {
				continue
			}
			seen[r] = true
			if i, err := font.GlyphIndex(&buf, r); err != nil || i == 0 {
				t.Errorf("%s does not cover %q (U+%04X)", file, r, r)
			}
		}
	}
}

func TestNotoSerifSCCoversTheChineseCopy(t *testing.T) {
	var texts []string
	for _, path := range []string{
		"../../internal/ideas/assets/zh-Hans.json",
		"../../locales/zh-Hans.json",
	} {
		texts = append(texts, stringsIn(t, path)...)
	}
	covers(t, "assets/noto-serif-sc-light.otf", texts, false)
	covers(t, "assets/noto-serif-sc-regular.otf", texts, false)
}

func TestSpectralCoversTheEnglishCopy(t *testing.T) {
	var texts []string
	for _, path := range []string{
		"../../internal/ideas/assets/en.json",
		"../../locales/en.json",
	} {
		texts = append(texts, stringsIn(t, path)...)
	}
	covers(t, "assets/spectral-light.ttf", texts, true)
	covers(t, "assets/spectral-regular.ttf", texts, true)
}

func TestLoad(t *testing.T) {
	if err := Load(); err != nil {
		t.Fatal(err)
	}
}
