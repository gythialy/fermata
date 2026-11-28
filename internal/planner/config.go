package planner

import (
	"time"

	"github.com/gythialy/fermata/internal/skiptime"
)

// Kind is the kind of a break.
type Kind uint8

const (
	// Mini is a short break.
	Mini Kind = iota + 1
	// Long is a long break.
	Long
)

// String returns the kind's name, as settings and logs write it.
func (k Kind) String() string {
	switch k {
	case Mini:
		return "mini"
	case Long:
		return "long"
	default:
		return "unknown"
	}
}

// PostponeConfig is how long a break may be postponed, and when.
type PostponeConfig struct {
	Enabled bool
	// Time is how long the break waits when postponed.
	Time time.Duration
	// Limit is how often one break may be postponed; 0 disables postponing.
	Limit int
	// Percent is the share of the break, from its start, in which
	// postponing is allowed; 0 disables it and 100 allows it throughout.
	Percent int
}

// Allows reports whether a break of the given elapsed time may still be
// postponed, ignoring how often it was postponed already.
func (c PostponeConfig) Allows(elapsed, duration time.Duration) bool {
	if !c.Enabled || c.Limit <= 0 || c.Percent <= 0 || c.Time <= 0 {
		return false
	}
	if duration <= 0 {
		return false
	}
	return elapsed <= duration*time.Duration(c.Percent)/100
}

// BreakConfig is one kind of break's settings.
type BreakConfig struct {
	Enabled      bool
	Duration     time.Duration
	Interval     time.Duration
	Notify       bool
	Lead         time.Duration
	Postpone     PostponeConfig
	Strict       bool
	ManualFinish bool
}

// LeadTime returns the notice before a break, zero when notifications are
// off or the lead is not positive.
func (c BreakConfig) LeadTime() time.Duration {
	if !c.Notify {
		return 0
	}
	return max(c.Lead, 0)
}

// Config is the planner's whole configuration. The planner never reads the
// settings itself: the app hands it a Config built from the settings store.
type Config struct {
	Mini, Long         BreakConfig
	EveryNBreaks       int
	PauseDurations     []time.Duration
	MorningHour        int
	SkipPeriodsEnabled bool
	SkipPeriods        []skiptime.Period
}

// For returns the settings of one kind of break.
func (c Config) For(k Kind) BreakConfig {
	if k == Long {
		return c.Long
	}
	return c.Mini
}

// EveryN is the number of short breaks between long ones, at least 1.
func (c Config) EveryN() int { return max(c.EveryNBreaks, 1) }

// KindFor returns what the n-th break of a sequence is. Only one kind on
// makes every break that kind; otherwise every EveryN+1-th break is long.
func (c Config) KindFor(n int) Kind {
	switch {
	case !c.Mini.Enabled && c.Long.Enabled:
		return Long
	case !c.Long.Enabled:
		return Mini
	}
	if n > 0 && n%(c.EveryN()+1) == 0 {
		return Long
	}
	return Mini
}

// Gap is the wait between the end of one break and the start of the next.
func (c Config) Gap() time.Duration {
	if c.Mini.Enabled || !c.Long.Enabled {
		return c.Mini.Interval
	}
	return c.Long.Interval
}
