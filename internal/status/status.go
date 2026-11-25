// Package status turns the planner's state into the tray's words. It is a
// pure function of a snapshot and a clock, so every state is testable without
// a window.
package status

import (
	"fmt"
	"math"
	"time"

	"github.com/gythialy/fermata/internal/planner"
)

// Messages is the text the status needs, in the language the app shows. The
// app implements it from the catalog; tests implement it with markers.
type Messages interface {
	Preparing() string
	NextMini(n int) string
	NextLong(n int) string
	AfterMini(n int) string
	SoonMini() string
	SoonLong() string
	NotifiedMini(seconds int) string
	NotifiedLong(seconds int) string
	BreakingMini(left string) string
	BreakingLong(left string) string
	StrictMini(left string) string
	StrictLong(left string) string
	BreakOver() string
	PausedTimed(minutes int) string
	PausedForever() string
	PausedDnd() string
	PausedIdle() string
	PausedLocked() string
	PausedExclusion() string
	TooltipTitle() string
}

// Extra is what the planner's state alone does not tell.
type Extra struct {
	// LongAfterMinis is how many mini breaks remain before the next long
	// break, shown on the second line; 0 shows none.
	LongAfterMinis int
}

// Lines is what the tray shows.
type Lines struct {
	// Status is the status line of the menu, and the tooltip's second part.
	Status string
	// Detail is the menu item's subtitle, when the status has one.
	Detail string
	// Tooltip is the tray icon's tooltip: the title, then the status.
	Tooltip string
}

// Format returns the words for a state at now.
func Format(st planner.State, now time.Time, extra Extra, m Messages) Lines {
	var lines Lines
	switch st.Phase {
	case planner.Preparing:
		lines.Status = m.Preparing()
	case planner.Paused:
		lines.Status = pausedText(st, now, m)
	case planner.Breaking:
		left := Duration(st.EndAt.Sub(now))
		if st.Strict {
			lines.Status = strictText(st.Kind, left, m)
		} else {
			lines.Status = breakingText(st.Kind, left, m)
		}
	case planner.ManualAwait:
		lines.Status = m.BreakOver()
	case planner.Notified:
		seconds := ceilSeconds(st.BreakAt.Sub(now))
		lines.Status = notifiedText(st.Kind, seconds, m)
	case planner.Waiting:
		remaining := st.BreakAt.Sub(now)
		switch {
		case remaining <= 0:
			lines.Status = soonText(st.Kind, m)
		case remaining < time.Minute:
			lines.Status = notifiedText(st.Kind, ceilSeconds(remaining), m)
		default:
			lines.Status = nextText(st.Kind, roundMinutes(remaining), m)
		}
		if st.Kind == planner.Mini && extra.LongAfterMinis > 0 {
			lines.Detail = m.AfterMini(extra.LongAfterMinis)
		}
	}
	lines.Tooltip = m.TooltipTitle()
	if lines.Status != "" {
		lines.Tooltip += "\n\n" + lines.Status
	}
	if lines.Detail != "" {
		lines.Tooltip += "\n" + lines.Detail
	}
	return lines
}

func pausedText(st planner.State, now time.Time, m Messages) string {
	switch st.Reasons.Primary() {
	case planner.Manual:
		if st.PauseUntil.IsZero() {
			return m.PausedForever()
		}
		return m.PausedTimed(max(roundMinutes(st.PauseUntil.Sub(now)), 1))
	case planner.DND:
		return m.PausedDnd()
	case planner.Idle:
		return m.PausedIdle()
	case planner.Lock:
		return m.PausedLocked()
	case planner.Exclusion:
		return m.PausedExclusion()
	default:
		return m.PausedForever()
	}
}

func nextText(kind planner.Kind, minutes int, m Messages) string {
	if kind == planner.Long {
		return m.NextLong(minutes)
	}
	return m.NextMini(minutes)
}

func notifiedText(kind planner.Kind, seconds int, m Messages) string {
	if kind == planner.Long {
		return m.NotifiedLong(seconds)
	}
	return m.NotifiedMini(seconds)
}

func soonText(kind planner.Kind, m Messages) string {
	if kind == planner.Long {
		return m.SoonLong()
	}
	return m.SoonMini()
}

func breakingText(kind planner.Kind, left string, m Messages) string {
	if kind == planner.Long {
		return m.BreakingLong(left)
	}
	return m.BreakingMini(left)
}

func strictText(kind planner.Kind, left string, m Messages) string {
	if kind == planner.Long {
		return m.StrictLong(left)
	}
	return m.StrictMini(left)
}

func roundMinutes(d time.Duration) int {
	if d <= 0 {
		return 0
	}
	return int(math.Round(d.Minutes()))
}

func ceilSeconds(d time.Duration) int {
	if d <= 0 {
		return 0
	}
	return int(math.Ceil(d.Seconds()))
}

// Duration writes a countdown as m:ss, or h:mm:ss from an hour on.
func Duration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	d = d.Round(time.Second)
	h := int(d / time.Hour)
	m := int(d/time.Minute) % 60
	s := int(d/time.Second) % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

// Overshoot writes how long a break has waited for the user, as +m:ss.
func Overshoot(d time.Duration) string { return "+" + Duration(d) }

// Clock writes a time of day, in 24 or 12 hour form with AM/PM.
func Clock(t time.Time, hour12 bool) string {
	if hour12 {
		return t.Format("3:04 PM")
	}
	return t.Format("15:04")
}
