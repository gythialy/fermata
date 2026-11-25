package ideas

import (
	"strings"
	"testing"

	"github.com/gythialy/fermata/internal/planner"
)

func TestDeckDrawsWithoutRepeating(t *testing.T) {
	d := NewDeck("en")
	n := len(d.mini)
	counts := map[string]int{}
	prev := ""
	for range n {
		idea := d.Next(planner.Mini)
		if idea.Text == "" {
			t.Fatal("empty idea")
		}
		if idea.Text == prev {
			t.Fatalf("repeated %q", idea.Text)
		}
		counts[idea.Text]++
		prev = idea.Text
	}
	if len(counts) != n {
		t.Errorf("%d distinct ideas in a %d-card deck", len(counts), n)
	}
	for text, n := range counts {
		if n != 1 {
			t.Errorf("%q drawn %d times", text, n)
		}
	}
}

func TestLongIdeasHaveATitleAndText(t *testing.T) {
	d := NewDeck("en")
	for range len(d.long) {
		idea := d.Next(planner.Long)
		if idea.Title == "" || idea.Text == "" {
			t.Fatalf("long idea without both lines: %+v", idea)
		}
	}
}

func TestDecksAreLargeAndDistinct(t *testing.T) {
	for _, lang := range []string{"en", "zh-Hans"} {
		d := NewDeck(lang)
		if len(d.mini) != 60 || len(d.long) != 36 {
			t.Errorf("%s has %d mini, %d long", lang, len(d.mini), len(d.long))
		}
		seen := map[string]bool{}
		for _, idea := range d.mini {
			if seen[idea.Text] {
				t.Errorf("%s repeats mini %q", lang, idea.Text)
			}
			seen[idea.Text] = true
		}
		seen = map[string]bool{}
		for _, idea := range d.long {
			key := idea.Title + "\x00" + idea.Text
			if seen[key] {
				t.Errorf("%s repeats long %q", lang, idea.Title)
			}
			seen[key] = true
		}
	}
}

func TestLanguages(t *testing.T) {
	d := NewDeck("zh-Hans")
	if got := d.Language(); got != "zh-Hans" {
		t.Fatalf("language %q", got)
	}
	idea := d.Next(planner.Mini)
	if !strings.ContainsAny(idea.Text, "。，") {
		t.Errorf("not Chinese: %q", idea.Text)
	}
	if err := d.SetLanguage("en"); err != nil {
		t.Fatal(err)
	}
	if got := d.Next(planner.Mini).Text; strings.ContainsAny(got, "。") {
		t.Errorf("still Chinese: %q", got)
	}
}

func TestUnknownLanguageFallsBack(t *testing.T) {
	d := NewDeck("fr")
	if got := d.Language(); got != DefaultLanguage {
		t.Errorf("language %q", got)
	}
	if idea := d.Next(planner.Mini); idea.Text == "" {
		t.Error("no idea")
	}
}
