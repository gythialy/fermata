package app

import (
	"runtime"
	"time"

	"github.com/gythialy/fermata/internal/breakwin"
	"github.com/gythialy/fermata/internal/planner"
	"github.com/gythialy/fermata/internal/status"
	"github.com/gythialy/fermata/internal/tray"
	"github.com/gythialy/fermata/locales"
)

// messages is the status text in the language the app shows.
type messages struct{}

func (messages) Preparing() string               { return locales.MenuPreparing() }
func (messages) NextMini(n int) string           { return locales.MenuNextMini(n) }
func (messages) NextLong(n int) string           { return locales.MenuNextLong(n) }
func (messages) AfterMini(n int) string          { return locales.MenuAfterMini(n) }
func (messages) SoonMini() string                { return locales.MenuSoonMini() }
func (messages) SoonLong() string                { return locales.MenuSoonLong() }
func (messages) NotifiedMini(s int) string       { return locales.MenuNotifiedMini(s) }
func (messages) NotifiedLong(s int) string       { return locales.MenuNotifiedLong(s) }
func (messages) BreakingMini(left string) string { return locales.MenuBreakingMini(left) }
func (messages) BreakingLong(left string) string { return locales.MenuBreakingLong(left) }
func (messages) StrictMini(left string) string   { return locales.MenuStrictMini(left) }
func (messages) StrictLong(left string) string   { return locales.MenuStrictLong(left) }
func (messages) BreakOver() string               { return locales.MenuBreakOver() }
func (messages) PausedTimed(m int) string        { return locales.MenuPausedTimed(m) }
func (messages) PausedForever() string           { return locales.MenuPausedForever() }
func (messages) PausedDnd() string               { return locales.MenuPausedDnd() }
func (messages) PausedIdle() string              { return locales.MenuPausedIdle() }
func (messages) PausedLocked() string            { return locales.MenuPausedLocked() }
func (messages) PausedExclusion() string         { return locales.MenuPausedExclusion() }
func (messages) TooltipTitle() string            { return locales.TrayTooltip() }

// trayLabels are the menu's words.
func trayLabels() tray.Labels {
	return tray.Labels{
		SkipTo:       locales.MenuSkipTo(),
		Mini:         locales.MenuMiniBreak(),
		Long:         locales.MenuLongBreak(),
		Pause:        locales.MenuPause(),
		Resume:       locales.MenuResume(),
		Reset:        locales.MenuReset(),
		Settings:     settingsLabel(),
		Quit:         quitLabel(),
		Morning:      locales.MenuPauseMorning(),
		Indefinitely: locales.MenuPauseIndefinitely(),
		PauseFor:     pauseLabel,
	}
}

// settingsLabel moves to "Settings…" on macOS and "Settings" elsewhere.
func settingsLabel() string {
	if runtime.GOOS == "darwin" {
		return locales.MenuSettingsEllipsis()
	}
	return locales.MenuSettings()
}

// quitLabel names quitting as each platform's menus do.
func quitLabel() string {
	if runtime.GOOS == "windows" {
		return locales.MenuExit()
	}
	return locales.MenuQuit()
}

// pauseLabel names a timed pause the way the menu does.
func pauseLabel(d time.Duration) string {
	switch d {
	case 30 * time.Minute:
		return locales.MenuPause30()
	case time.Hour:
		return locales.MenuPause1h()
	case 2 * time.Hour:
		return locales.MenuPause2h()
	case 5 * time.Hour:
		return locales.MenuPause5h()
	}
	return locales.DurationMinutes(int(d.Minutes()))
}

// breakLabel names a break window's label.
func breakLabel(kind planner.Kind, strict bool) string {
	switch {
	case strict && kind == planner.Long:
		return locales.BreakLabelStrictLong()
	case strict:
		return locales.BreakLabelStrictMini()
	case kind == planner.Long:
		return locales.BreakLabelLong()
	default:
		return locales.BreakLabelMini()
	}
}

// actionLabel names the break window's one exit.
func actionLabel(action breakwin.Action) string {
	switch action {
	case breakwin.Postpone:
		return locales.BreakPostpone()
	case breakwin.Finish:
		return locales.BreakFinish()
	default:
		return locales.BreakSkip()
	}
}

// statusDuration is used where a countdown is needed outside the view.
func statusDuration(d time.Duration) string { return status.Duration(d) }
