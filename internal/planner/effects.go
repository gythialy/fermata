package planner

import "time"

// Effect is what the app does after a step: arm a timer, open or close a
// window, notify. The planner changes no external state itself.
type Effect interface{ isEffect() }

// Arm sets the single deadline the scheduler wakes the planner at.
type Arm struct{ At time.Time }

// Disarm cancels the deadline.
type Disarm struct{}

// Notify sends the notice before a break.
type Notify struct {
	Kind Kind
	Lead time.Duration
}

// CloseNotice closes the notice, if one is still shown.
type CloseNotice struct{}

// StartBreak opens the break window group.
type StartBreak struct{ Spec BreakSpec }

// CloseReason says why a break ended.
type CloseReason uint8

const (
	// Completed is a break that ran its time.
	Completed CloseReason = iota + 1
	// Postponed is a break moved later.
	Postponed
	// Skipped is a break the user ended early.
	Skipped
	// Interrupted is a break closed by a pause, a reset or the system.
	Interrupted
)

// String returns the reason's name, as logs write it.
func (r CloseReason) String() string {
	switch r {
	case Completed:
		return "completed"
	case Postponed:
		return "postponed"
	case Skipped:
		return "skipped"
	case Interrupted:
		return "interrupted"
	default:
		return "unknown"
	}
}

// CloseBreak closes the break window group.
type CloseBreak struct{ Reason CloseReason }

// Sound names the planner's sound cues; the app maps them to the settings'
// sounds and volume.
const (
	SoundStart = "start"
	SoundEnd   = "end"
)

// PlaySound plays a sound cue.
type PlaySound struct{ Name string }

// NotifyResume sends the notification that a manual pause ended.
type NotifyResume struct{}

// TrayDirty asks the app to refresh the tray's title, icon and menu.
type TrayDirty struct{}

func (Arm) isEffect()          {}
func (Disarm) isEffect()       {}
func (Notify) isEffect()       {}
func (CloseNotice) isEffect()  {}
func (StartBreak) isEffect()   {}
func (CloseBreak) isEffect()   {}
func (PlaySound) isEffect()    {}
func (NotifyResume) isEffect() {}
func (TrayDirty) isEffect()    {}
