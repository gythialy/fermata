package app

import (
	"fmt"
	"math"
	"runtime"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/gythialy/fermata/internal/planner"
	"github.com/gythialy/fermata/internal/popover"
	"github.com/gythialy/fermata/internal/status"
	"github.com/gythialy/fermata/locales"
)

// popoverWidth is the flyout's width in DIPs. It is the one size the design
// fixes: the height follows the content through FitContent.
const popoverWidth = 320

// showPopover opens the status popover, or refreshes it if it is open.
func (a *App) showPopover() {
	if a.tray == nil {
		return
	}
	if a.popover != nil {
		a.popover.Invalidate()
		return
	}
	a.togglePopover()
}

// togglePopover opens or closes the status popover.
func (a *App) togglePopover() {
	if a.popover != nil {
		a.hidePopover()
		return
	}
	a.popover = mygo.NewFlyout(mygo.FlyoutOptions{
		Tray:      a.tray.Native(),
		Placement: mygo.PlacementBottom,
		Gap:       4,
		Width:     popoverWidth,
		// The panel is as tall as what it shows: the action rows come and
		// go with the state, and MyGo lays the content out again as the
		// window takes the measured size.
		FitContent:   true,
		MinWidth:     popoverWidth,
		MaxWidth:     popoverWidth,
		MaxHeight:    560,
		Focusable:    false,
		Popover:      runtime.GOOS == "darwin",
		Shadow:       true,
		CornerRadius: 14,
		Content:      ui.View(a.popoverView),
	})
	a.popover.OnClosed(func() { a.popover = nil })
	a.popover.Show()
	a.popover.Invalidate()
	a.tickPopover()
	a.log.Debug("popover opened")
}

// hidePopover closes the status popover before the menu opens.
func (a *App) hidePopover() {
	if a.popover == nil {
		return
	}
	if a.popoverTimer != nil {
		a.popoverTimer.Stop()
		a.popoverTimer = nil
	}
	p := a.popover
	a.popover = nil
	p.Close()
}

// showTrayMenu opens the menu where a right click may not reach the icon.
func (a *App) showTrayMenu() {
	a.hidePopover()
	if a.tray != nil {
		a.tray.ShowMenu()
	}
}

// tickPopover keeps the popover's countdown moving while it shows.
func (a *App) tickPopover() {
	if a.popover == nil {
		return
	}
	a.popover.Invalidate()
	if a.popoverTimer != nil {
		a.popoverTimer.Stop()
	}
	a.popoverTimer = time.AfterFunc(time.Second, func() { mygo.RunOnMain(a.tickPopover) })
}

// popoverView draws the popover under the menu bar icon.
func (a *App) popoverView(c *ui.Context) {
	now := time.Now()
	st, spec := a.pl.Snapshot()
	lines := a.trayLines(now)
	opts := popover.Options{
		State:     popoverState(st),
		Remaining: remainingLabel(st, now),
		Detail:    a.popoverDetail(st, lines),
		Share:     progressShare(st, spec, a.pl.Config(), now),
		Actions:   a.popoverActions(st),
		Menu:      a.showTrayMenu,
		Close:     a.hidePopover,
	}
	// A strict break has no way out, so it offers none here either.
	if !st.Strict {
		opts.Settings = func() {
			a.hidePopover()
			a.openSettings()
		}
	}
	popover.Show(c, opts)
}

// popoverState names the moment the flyout shows.
func popoverState(st planner.State) string {
	kind := locales.PopoverMini()
	if st.Kind == planner.Long {
		kind = locales.PopoverLong()
	}
	switch st.Phase {
	case planner.Waiting:
		if st.Kind == planner.Long {
			return locales.PopoverNextLong()
		}
		return locales.PopoverNextMini()
	case planner.Notified:
		return kind + " · " + locales.PopoverStarting()
	case planner.Breaking, planner.ManualAwait:
		if st.Strict {
			return locales.PopoverStrict() + " · " + kind
		}
		return kind + " · " + locales.PopoverInProgress()
	case planner.Paused:
		return locales.PopoverPaused()
	case planner.Preparing:
		return kind
	}
	return kind
}

// popoverDetail is the flyout's second line: the rhythm while waiting, the
// suggestion during a break, and nothing where the state says it already.
func (a *App) popoverDetail(st planner.State, lines status.Lines) string {
	switch st.Phase {
	case planner.Waiting, planner.Notified:
		return lines.Detail
	case planner.Breaking:
		return a.lastIdea.Text
	}
	return ""
}

// popoverActions returns the buttons this state offers, in order.
func (a *App) popoverActions(st planner.State) []popover.Action {
	if st.Strict {
		return nil
	}
	switch st.Phase {
	case planner.Waiting:
		return []popover.Action{
			{Label: locales.PopoverTakeBreak(), Primary: true, Run: func() {
				a.hidePopover()
				a.step(planner.Advance{Now: time.Now()})
			}},
			{Label: locales.PopoverPause(), Run: func() {
				a.hidePopover()
				a.pauseFor(30*time.Minute, false)
			}},
		}
	case planner.Notified:
		return []popover.Action{
			{Label: locales.PopoverStartNow(), Primary: true, Run: func() {
				a.hidePopover()
				a.step(planner.Advance{Now: time.Now()})
			}},
			{Label: locales.PopoverPush5(), Run: func() {
				a.hidePopover()
				a.step(planner.Defer{Now: time.Now(), By: 5 * time.Minute})
			}},
			{Label: locales.PopoverPause(), Run: func() {
				a.hidePopover()
				a.pauseFor(30*time.Minute, false)
			}},
		}
	case planner.Paused:
		return []popover.Action{
			{Label: locales.MenuResume(), Primary: true, Run: func() {
				a.hidePopover()
				a.resume()
			}},
		}
	}
	return nil
}

// remainingLabel is the time until the next break, or the rest of this one.
// The notice counts in seconds, so the last ten seconds read as they pass.
func remainingLabel(st planner.State, now time.Time) string {
	switch st.Phase {
	case planner.Waiting:
		minutes := max(int(math.Ceil(st.BreakAt.Sub(now).Minutes())), 0)
		return fmt.Sprintf("%dm", minutes)
	case planner.Notified:
		seconds := max(int(math.Ceil(st.BreakAt.Sub(now).Seconds())), 0)
		return fmt.Sprintf("%ds", seconds)
	case planner.Breaking, planner.ManualAwait:
		left := max(st.EndAt.Sub(now), 0)
		return fmt.Sprintf("%d:%02d", int(left.Minutes()), int(left.Seconds())%60)
	case planner.Paused:
		if !st.PauseUntil.IsZero() {
			left := max(st.PauseUntil.Sub(now), 0)
			return fmt.Sprintf("%d:%02d", int(left.Minutes()), int(left.Seconds())%60)
		}
		return "—"
	}
	return ""
}
