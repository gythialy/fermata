package locales

import (
	"encoding/json"
	"os"
	"sort"
	"strings"
	"testing"
)

// message is the shape of one catalog entry, without its metadata.
type message map[string]any

func read(t *testing.T, path string) map[string]message {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	out := map[string]message{}
	for key, value := range raw {
		if strings.HasPrefix(key, "@") {
			continue
		}
		if s, ok := value.(string); ok {
			out[key] = message{"text": s}
		}
	}
	return out
}

func placeholders(s string) []string {
	var out []string
	for i := 0; i < len(s); i++ {
		if s[i] != '{' {
			continue
		}
		depth, end := 1, i+1
		for ; end < len(s) && depth > 0; end++ {
			switch s[end] {
			case '{':
				depth++
			case '}':
				depth--
			}
		}
		if depth != 0 {
			break
		}
		field := s[i+1 : end-1]
		for k := 0; k < len(field); k++ {
			if field[k] == ',' || field[k] == '}' {
				field = field[:k]
				break
			}
		}
		if field = strings.TrimSpace(field); field != "" {
			out = append(out, field)
		}
		i = end - 1
	}
	sort.Strings(out)
	return out
}

func TestCatalogsHaveTheSameKeysAndPlaceholders(t *testing.T) {
	en := read(t, "en.json")
	zh := read(t, "zh-Hans.json")
	if len(en) != len(zh) {
		t.Errorf("en has %d messages, zh-Hans has %d", len(en), len(zh))
	}
	for key, msg := range en {
		other, ok := zh[key]
		if !ok {
			t.Errorf("zh-Hans is missing %q", key)
			continue
		}
		a, b := placeholders(msg["text"].(string)), placeholders(other["text"].(string))
		if strings.Join(a, ",") != strings.Join(b, ",") {
			t.Errorf("%q placeholders differ: en %v, zh %v", key, a, b)
		}
	}
	for key := range zh {
		if _, ok := en[key]; !ok {
			t.Errorf("en is missing %q", key)
		}
	}
}
