package app

import (
	"os"
	"runtime"
	"time"

	"github.com/egoist/mygo"

	"github.com/gythialy/fermata/internal/fullscreen"
	"github.com/gythialy/fermata/internal/planner"
	"github.com/gythialy/fermata/internal/settings"
)

const (
	// holdTyping and holdFullscreen name why a due break waits.
	holdTyping     = "typing"
	holdFullscreen = "fullscreen"

	// typingQuiet is how long without input counts as a pause in typing.
	typingQuiet = 500 * time.Millisecond
	// typingHoldLast is the longest a break waits for typing to stop.
	typingHoldLast = 30 * time.Second
	// fullscreenHoldLast is the longest a break waits for a fullscreen app.
	fullscreenHoldLast = time.Hour
	// holdPoll is how often a held break checks whether it may start.
	holdPoll = 2 * time.Second
)

// holdNeeded reports why a due break should wait, if it should.
func (a *App) holdNeeded(now time.Time) string {
	if a.pl == nil {
		return ""
	}
	st, _ := a.pl.Snapshot()
	return holdDecision(st.Phase, st.BreakAt, st.NotifyAt, now, a.store.Get().Presence, a.fullscreenActive, a.typing())
}

// holdDecision is holdNeeded's rule, without the app's moving parts.
func holdDecision(phase planner.Phase, breakAt, notifyAt, now time.Time,
	presence settings.Presence, fullscreen, typing bool) string {
	if phase != planner.Waiting && phase != planner.Notified {
		return ""
	}
	breakDue := !breakAt.IsZero() && !now.Before(breakAt)
	noticeDue := !notifyAt.IsZero() && !now.Before(notifyAt)
	if !breakDue && !noticeDue {
		return ""
	}
	if presence.Fullscreen && fullscreen {
		return holdFullscreen
	}
	if presence.WaitWhileTyping && breakDue && typing {
		return holdTyping
	}
	return ""
}

// typing reports whether the user typed or moved the mouse just now.
func (a *App) typing() bool {
	return mygo.Power.IdleTime() < typingQuiet
}

// beginHold stops the break from starting and wakes up to look again.
func (a *App) beginHold(reason string, now time.Time) {
	a.holdReason = reason
	a.holdStart = now
	if a.notifier != nil {
		a.notifier.Close()
	}
	a.log.Info("break held", "reason", reason)
	a.drv.Arm(now.Add(holdPoll))
	a.refreshTray()
}

// holdTick decides whether a held break may start now.
func (a *App) holdTick(now time.Time) {
	limit := typingHoldLast
	if a.holdReason == holdFullscreen {
		limit = fullscreenHoldLast
	}
	if now.Sub(a.holdStart) >= limit {
		a.log.Info("break released", "reason", a.holdReason, "held", limit.String())
		a.releaseHold(now)
		return
	}
	if a.holdStillNeeded() {
		a.drv.Arm(now.Add(holdPoll))
		a.refreshTray()
		return
	}
	a.log.Info("break released", "reason", a.holdReason)
	a.releaseHold(now)
}

// holdStillNeeded reports whether the reason for holding is still there.
func (a *App) holdStillNeeded() bool {
	return holdStill(a.holdReason, a.store.Get().Presence, a.fullscreenActive, a.typing())
}

// holdStill is holdStillNeeded's rule, without the app's moving parts.
func holdStill(reason string, presence settings.Presence, fullscreen, typing bool) bool {
	switch reason {
	case holdFullscreen:
		return presence.Fullscreen && fullscreen
	case holdTyping:
		return presence.WaitWhileTyping && typing
	}
	return false
}

// releaseHold starts the held break, which keeps its whole duration, or
// steps the planner when the break is not due yet.
func (a *App) releaseHold(now time.Time) {
	a.holdReason = ""
	st, _ := a.pl.Snapshot()
	if (st.Phase == planner.Waiting || st.Phase == planner.Notified) && !now.Before(st.BreakAt) {
		a.step(planner.Show{Now: now})
		return
	}
	a.step(planner.Tick{Now: now})
}

// startFullscreenWatch looks for a fullscreen app every few seconds.
func (a *App) startFullscreenWatch() {
	if runtime.GOOS != "darwin" {
		return
	}
	a.scheduleFullscreenCheck()
}

// scheduleFullscreenCheck arms the next look, slower when the setting is off.
func (a *App) scheduleFullscreenCheck() {
	if a.quitting {
		return
	}
	delay := holdPoll
	if !a.store.Get().Presence.Fullscreen {
		delay = 15 * time.Second
	}
	if a.fullscreenTimer != nil {
		a.fullscreenTimer.Stop()
	}
	a.fullscreenTimer = time.AfterFunc(delay, func() {
		mygo.RunOnMain(a.checkFullscreen)
	})
}

// checkFullscreen reads the window list and tells the tray and a held break.
func (a *App) checkFullscreen() {
	if a.quitting {
		return
	}
	defer a.scheduleFullscreenCheck()
	if !a.store.Get().Presence.Fullscreen {
		a.setFullscreen(false)
		return
	}
	var displays []fullscreen.Rect
	for _, d := range mygo.Screen.Displays() {
		displays = append(displays, fullscreen.Rect{
			X: float64(d.Bounds.X),
			Y: float64(d.Bounds.Y),
			W: float64(d.Bounds.Width),
			H: float64(d.Bounds.Height),
		})
	}
	active, err := fullscreen.Active(displays, os.Getpid())
	if err != nil {
		if !a.fullscreenBroken {
			a.fullscreenBroken = true
			a.log.Info("fullscreen watch is unavailable", "error", err)
		}
		active = false
	}
	a.setFullscreen(active)
}

// setFullscreen remembers whether a fullscreen app is in front, and lets a
// break held for it go when the app leaves fullscreen.
func (a *App) setFullscreen(on bool) {
	if on == a.fullscreenActive {
		return
	}
	a.fullscreenActive = on
	if !on && a.holdReason == holdFullscreen {
		a.holdTick(time.Now())
		return
	}
	a.refreshTray()
}
