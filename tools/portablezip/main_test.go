package main

import (
	"archive/zip"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// packed lists the archive's entries in order.
func packed(t *testing.T, out string) []string {
	t.Helper()
	r, err := zip.OpenReader(out)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	var names []string
	for _, f := range r.File {
		names = append(names, f.Name)
	}
	return names
}

// TestRunPacksTheAppOnly keeps the installer and the build's bookkeeping out
// of an archive people hand to each other.
func TestRunPacksTheAppOnly(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{
		"Fermata.exe", "icon.png", "tray.png",
		".mygo-files", "fermata-setup-1.0.0-amd64.exe", "Fermata-old-portable.zip",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(name), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	out := filepath.Join(t.TempDir(), "Fermata-1.0.0-windows-amd64-portable.zip")
	if err := run(dir, out); err != nil {
		t.Fatal(err)
	}

	want := []string{"Fermata.exe", "icon.png", "tray.png"}
	if got := packed(t, out); !slices.Equal(got, want) {
		t.Errorf("archive holds %v, want %v", got, want)
	}
}

// TestRunFailsOnNothingToShip: a directory of only installers is a build
// mistake, and an empty archive would hide it.
func TestRunFailsOnNothingToShip(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "fermata-setup-1.0.0-amd64.exe"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run(dir, filepath.Join(t.TempDir(), "out.zip")); err == nil {
		t.Error("a directory with nothing to ship should fail the build")
	}
}
