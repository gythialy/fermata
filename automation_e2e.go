//go:build e2e

// The e2e build is driven from outside, by the mygo automation plugin whose
// protocol is how a test reads and acts on a running app. Release builds
// leave this file, and the plugin, out.
package main

import (
	"os"
	"path/filepath"

	"github.com/egoist/mygo"

	_ "github.com/egoist/mygo/plugins/automation"

	"github.com/gythialy/fermata/internal/cli"
)

// FERMATA_E2E_DATA names a directory for the run's settings, logs and cache,
// and gives the instance a name of its own, so that a test never disturbs an
// installed Fermata — nor is forwarded to it and quits before the test finds
// it.
func init() {
	// The commands a test drives the app with, which a release build reports
	// as unknown.
	cli.Debug = true
	dir := os.Getenv("FERMATA_E2E_DATA")
	if dir == "" {
		return
	}
	appName = "Fermata E2E"
	for _, sub := range []string{"", "logs", "cache"} {
		_ = os.MkdirAll(filepath.Join(dir, sub), 0o755)
	}
	mygo.App.SetPath(mygo.PathUserData, dir)
	mygo.App.SetPath(mygo.PathLogs, filepath.Join(dir, "logs"))
	mygo.App.SetPath(mygo.PathCache, filepath.Join(dir, "cache"))
}
