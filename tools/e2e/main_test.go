package main

import "testing"

// TestSlug names the files of a capture after their window: ASCII, lower
// case, and never empty.
func TestSlug(t *testing.T) {
	for _, tc := range []struct{ title, want string }{
		{"Settings", "settings"},
		{"设置", "window"},
		{"Fermata — Break", "fermata--break"},
		{"  ", "window"},
	} {
		if got := slug(tc.title); got != tc.want {
			t.Errorf("slug(%q) = %q, want %q", tc.title, got, tc.want)
		}
	}
}

func TestEnvOr(t *testing.T) {
	t.Setenv("FERMATA_E2E_TEST_VALUE", "set")
	if got := envOr("FERMATA_E2E_TEST_VALUE", "fallback"); got != "set" {
		t.Errorf("envOr set = %q", got)
	}
	if got := envOr("FERMATA_E2E_TEST_VALUE_UNSET", "fallback"); got != "fallback" {
		t.Errorf("envOr unset = %q", got)
	}
}
