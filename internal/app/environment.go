package app

import (
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/gythialy/fermata/internal/update"
)

// The about page's links.
const (
	websiteURL = "https://github.com/gythialy/fermata"
	sourceURL  = "https://github.com/gythialy/fermata"
	issuesURL  = "https://github.com/gythialy/fermata/issues/new"
)

// contains is strings.Contains for the few calls that check a warning.
func contains(s, sub string) bool { return strings.Contains(s, sub) }

// openURL opens a link in the user's browser.
func (a *App) openURL(url string) { mygo.Shell.OpenExternal(url) }

// scheduleUpdateCheck looks for a newer release once the app has settled, and
// then once a day. It never bothers the user: the about page shows what it
// found.
func (a *App) scheduleUpdateCheck() {
	if a.opts.Version == "" {
		return
	}
	go func() {
		first := time.NewTimer(8 * time.Second)
		defer first.Stop()
		<-first.C
		for {
			if a.quitting {
				return
			}
			version, url, err := update.Latest(a.opts.Version)
			if err != nil {
				a.log.Debug("update check", "error", err)
			} else if version != "" {
				mygo.RunOnMain(func() {
					a.updateVersion, a.updateURL = version, url
					if a.prefs != nil {
						a.prefs.Refresh()
					}
				})
			}
			next := time.NewTimer(24 * time.Hour)
			<-next.C
			next.Stop()
			if a.quitting {
				return
			}
		}
	}()
}
