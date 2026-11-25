package planner

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Phase is what the planner is doing.
type Phase uint8

const (
	// Preparing is the state before the first schedule: the app is starting
	// or the welcome window is open.
	Preparing Phase = iota
	// Waiting is the time before the next break, or its notice.
	Waiting
	// Notified is the notice before the next break.
	Notified
	// Breaking is a break counting down.
	Breaking
	// ManualAwait is a break whose countdown ran out and which waits for
	// the user to finish it.
	ManualAwait
	// Paused is the state with no schedule, because of at least one reason.
	Paused
)

// String returns the phase's name, as logs write it.
func (p Phase) String() string {
	switch p {
	case Preparing:
		return "preparing"
	case Waiting:
		return "waiting"
	case Notified:
		return "notified"
	case Breaking:
		return "breaking"
	case ManualAwait:
		return "manual-await"
	case Paused:
		return "paused"
	default:
		return "unknown"
	}
}

// Reason is why breaks are paused. Pausing is a set: breaks resume once the
// last reason is gone, and the reasons name the pause in the tray.
type Reason uint8

const (
	// Manual is a pause the user asked for, timed or indefinite.
	Manual Reason = 1 << iota
	// DND is the system's Focus or Do Not Disturb.
	DND
	// Idle is the user being away from the keyboard.
	Idle
	// Lock is a locked screen or a sleeping system.
	Lock
	// Exclusion is an app exclusion rule.
	Exclusion
)

// Has reports whether the set includes r.
func (r Reason) Has(o Reason) bool { return r&o != 0 }

// With returns the set with o added.
func (r Reason) With(o Reason) Reason { return r | o }

// Without returns the set with o removed.
func (r Reason) Without(o Reason) Reason { return r &^ o }

// Empty reports whether no reason is set.
func (r Reason) Empty() bool { return r == 0 }

// Primary returns the reason the status names, in display order:
// manual, dnd, idle, exclusion, lock.
func (r Reason) Primary() Reason {
	for _, o := range [...]Reason{Manual, DND, Idle, Exclusion, Lock} {
		if r.Has(o) {
			return o
		}
	}
	return 0
}

// String returns the reasons' names, comma separated, for logs.
func (r Reason) String() string {
	if r == 0 {
		return "none"
	}
	var names []string
	for _, o := range [...]Reason{Manual, DND, Idle, Lock, Exclusion} {
		if r.Has(o) {
			names = append(names, o.name())
		}
	}
	return strings.Join(names, ",")
}

func (r Reason) name() string {
	switch r {
	case Manual:
		return "manual"
	case DND:
		return "dnd"
	case Idle:
		return "idle"
	case Lock:
		return "lock"
	case Exclusion:
		return "exclusion"
	default:
		return "unknown"
	}
}

// State is the planner's state after a step.
type State struct {
	Phase Phase
	// Kind is the next break's kind while waiting, and the current break's
	// kind while breaking.
	Kind Kind
	// N is the number of the next break in the sequence, from 1; 0 while
	// paused or idle.
	N int
	// NotifyAt is when the notice goes out; zero when there is none.
	NotifyAt time.Time
	// BreakAt is when the next break starts.
	BreakAt time.Time
	// StartAt, EndAt and Duration describe the break in progress.
	StartAt  time.Time
	EndAt    time.Time
	Duration time.Duration
	// Postpones is how often the current break was postponed.
	Postpones int
	// Reasons is why breaks are paused; empty unless Phase is Paused, or a
	// break runs while a pause reason arrived.
	Reasons Reason
	// PauseUntil ends a timed manual pause; zero means indefinite.
	PauseUntil time.Time
	// Strict and ManualFinish are snapshotted when the break starts.
	Strict       bool
	ManualFinish bool
	// Missed tells that the last break was missed, as when the system slept
	// through it. It is diagnostic: it changes no behavior.
	Missed bool

	// anchor is when the wait for the next break began: the end of the last
	// break, or a start, resume or reset.
	anchor time.Time
	// force starts the next break as this kind, whatever N says.
	force *Kind
	// restart carries a postponed break's snapshot to its new start.
	restart *breakOverride
	// title and text override the next break's idea, from the CLI.
	title, text string
	// curTitle, curText and postpone belong to the running break, so that a
	// postponement keeps them whatever the settings say by then.
	curTitle, curText string
	postpone          PostponeConfig
}

// breakOverride is the part of a break that survives a postponement: it is
// the same break, so a settings change does not alter it.
type breakOverride struct {
	Duration     time.Duration
	Strict       bool
	ManualFinish bool
	Postpone     PostponeConfig
	Title        string
	Text         string
}

// BreakSpec is a break's snapshot, taken when it starts. Settings changes do
// not alter a running break.
type BreakSpec struct {
	Kind         Kind
	Number       int
	StartAt      time.Time
	EndAt        time.Time
	Duration     time.Duration
	Strict       bool
	ManualFinish bool
	Postpone     PostponeConfig
	// Title and Text override the deck's idea for this break only.
	Title, Text string
}

// Remaining returns how much of the break is left at now, never negative.
func (s BreakSpec) Remaining(now time.Time) time.Duration {
	return max(s.EndAt.Sub(now), 0)
}

// Elapsed returns how much of the break has passed at now, from 0 to the
// duration.
func (s BreakSpec) Elapsed(now time.Time) time.Duration {
	return min(max(now.Sub(s.StartAt), 0), max(s.Duration, 0))
}

// Progress returns the share of the break that has passed, from 0 to 1.
func (s BreakSpec) Progress(now time.Time) float64 {
	if s.Duration <= 0 {
		return 0
	}
	return float64(s.Elapsed(now)) / float64(s.Duration)
}

// CanPostpone reports whether the break may be postponed at now.
func (s BreakSpec) CanPostpone(now time.Time, used int) bool {
	if s.Strict || used >= s.Postpone.Limit {
		return false
	}
	return s.Postpone.Allows(s.Elapsed(now), s.Duration)
}

// signature summarizes what the tray shows, so that a step can tell whether
// anything but the time changed. The countdowns are not part of it: the app
// refreshes them from its own timer, as often as they change.
func (s State) signature() string {
	var b strings.Builder
	b.WriteString(s.Phase.String())
	b.WriteByte('|')
	b.WriteString(s.Kind.String())
	b.WriteByte('|')
	b.WriteString(s.Reasons.String())
	b.WriteByte('|')
	b.WriteString(strconv.Itoa(s.N))
	b.WriteByte('|')
	b.WriteString(strconv.FormatBool(s.Strict))
	b.WriteByte('|')
	b.WriteString(strconv.FormatBool(s.ManualFinish))
	return b.String()
}

func (s State) validate() error {
	switch s.Phase {
	case Waiting:
		if s.BreakAt.IsZero() {
			return fmt.Errorf("waiting without a break deadline")
		}
		if s.N < 1 {
			return fmt.Errorf("waiting with n %d", s.N)
		}
	case Notified:
		if s.BreakAt.IsZero() || s.NotifyAt.IsZero() {
			return fmt.Errorf("notified without both deadlines")
		}
	case Breaking:
		if s.StartAt.IsZero() || s.EndAt.IsZero() || s.Duration <= 0 {
			return fmt.Errorf("breaking without a break snapshot")
		}
		if !s.EndAt.Equal(s.StartAt.Add(s.Duration)) {
			return fmt.Errorf("end %v is not start %v plus %v", s.EndAt, s.StartAt, s.Duration)
		}
	case ManualAwait:
		if !s.ManualFinish {
			return fmt.Errorf("manual await without manual finish")
		}
		if s.StartAt.IsZero() || s.Duration <= 0 {
			return fmt.Errorf("manual await without a break snapshot")
		}
	case Paused:
		if s.Reasons.Empty() {
			return fmt.Errorf("paused without a reason")
		}
	case Preparing:
	}
	if s.Phase != Paused && !s.Reasons.Empty() && s.Phase != Breaking && s.Phase != ManualAwait {
		return fmt.Errorf("reasons %v outside a pause", s.Reasons)
	}
	return nil
}
