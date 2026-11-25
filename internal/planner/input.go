package planner

import "time"

// Input is what moves the planner: a due time, a command, a settings
// change.
type Input interface{ isInput() }

// Tick is time passing, from a timer or a one-second refresh.
type Tick struct{ Now time.Time }

// Wake is time passing after the system slept. It never starts a break the
// sleep made the app miss.
type Wake struct{ Now time.Time }

// Pause adds a pause reason. Until ends a timed manual pause; zero is
// indefinite.
type Pause struct {
	Now    time.Time
	Reason Reason
	Until  time.Time
}

// Resume removes a pause reason.
type Resume struct {
	Now    time.Time
	Reason Reason
}

// Postpone closes the running break and starts it again later.
type Postpone struct{ Now time.Time }

// Skip closes the running break, counting it as taken.
type Skip struct{ Now time.Time }

// Finish closes a break whose countdown ran out and that waited for the
// user.
type Finish struct{ Now time.Time }

// Show starts a break that is already due, although it is later than the
// late-start limit: the app held it, for a fullscreen app or for typing.
type Show struct{ Now time.Time }

// Restore resumes the schedule a previous run saved, before its first break.
type Restore struct {
	Now      time.Time
	N        int
	BreakAt  time.Time
	NotifyAt time.Time
}

// Advance starts the next break now, before its time. It keeps its place in
// the sequence.
type Advance struct{ Now time.Time }

// Defer pushes the next break back by By, keeping its kind and number.
type Defer struct {
	Now time.Time
	By  time.Duration
}

// Reset forgets the sequence and schedules a first break again, unless the
// app is paused.
type Reset struct{ Now time.Time }

// SkipTo starts a break of Kind after Delay, whatever the sequence says.
type SkipTo struct {
	Now   time.Time
	Kind  Kind
	Delay time.Duration
	Title string
	Text  string
}

// NextText overrides the idea of the next break, without starting one.
type NextText struct {
	Now   time.Time
	Title string
	Text  string
}

// SetConfig replaces the configuration.
type SetConfig struct {
	Now    time.Time
	Config Config
}

func (Tick) isInput()      {}
func (Wake) isInput()      {}
func (Pause) isInput()     {}
func (Resume) isInput()    {}
func (Postpone) isInput()  {}
func (Skip) isInput()      {}
func (Finish) isInput()    {}
func (Show) isInput()      {}
func (Restore) isInput()   {}
func (Advance) isInput()   {}
func (Defer) isInput()     {}
func (Reset) isInput()     {}
func (SkipTo) isInput()    {}
func (NextText) isInput()  {}
func (SetConfig) isInput() {}

// nowOf returns the time an input carries.
func nowOf(in Input) time.Time {
	switch in := in.(type) {
	case Tick:
		return in.Now
	case Wake:
		return in.Now
	case Pause:
		return in.Now
	case Resume:
		return in.Now
	case Postpone:
		return in.Now
	case Skip:
		return in.Now
	case Finish:
		return in.Now
	case Show:
		return in.Now
	case Restore:
		return in.Now
	case Advance:
		return in.Now
	case Defer:
		return in.Now
	case Reset:
		return in.Now
	case SkipTo:
		return in.Now
	case NextText:
		return in.Now
	case SetConfig:
		return in.Now
	default:
		return time.Time{}
	}
}
