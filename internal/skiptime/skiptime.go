// Package skiptime matches a local clock time against configured windows.
// It never reads the clock: callers pass the time to test.
package skiptime

import "time"

// AllDays selects every weekday, as an empty day set does.
const AllDays uint8 = 0x7f

// Period is a daily window, possibly across midnight. StartMin and EndMin are
// minutes from midnight, 0..1439. Days uses 1=Mon..7=Sun as 1<<(d-1); zero
// selects every day. An end before the start belongs to the following day.
type Period struct {
	StartMin, EndMin int
	Days             uint8
}

// Active reports whether now falls inside any of the periods.
func Active(periods []Period, now time.Time) bool {
	minute := minuteOf(now)
	today := midnight(now)
	yesterday := today.AddDate(0, 0, -1)
	for _, p := range periods {
		if !p.valid() {
			continue
		}
		if p.selects(today) && p.covers(minute) {
			return true
		}
		if p.StartMin > p.EndMin && p.selects(yesterday) && minute < p.EndMin {
			// An overnight window's end belongs to the day after its start.
			return true
		}
	}
	return false
}

// NextChange returns the earliest period boundary strictly after now, or the
// zero time when there is none.
func NextChange(periods []Period, now time.Time) time.Time {
	base := midnight(now)
	var next time.Time
	for _, p := range periods {
		if !p.valid() {
			continue
		}
		// One day back covers an overnight end on today; eight forward covers
		// a weekly window whose end is on the day after its start.
		for offset := -1; offset <= 8; offset++ {
			day := base.AddDate(0, 0, offset)
			if !p.selects(day) {
				continue
			}
			start := at(day, p.StartMin)
			if start.After(now) && (next.IsZero() || start.Before(next)) {
				next = start
			}
			endDay := day
			if p.StartMin > p.EndMin {
				endDay = day.AddDate(0, 0, 1)
			}
			end := at(endDay, p.EndMin)
			if end.After(now) && (next.IsZero() || end.Before(next)) {
				next = end
			}
		}
	}
	return next
}

func (p Period) valid() bool {
	return p.StartMin >= 0 && p.StartMin <= 1439 && p.EndMin >= 0 && p.EndMin <= 1439 && p.StartMin != p.EndMin
}

func (p Period) selects(day time.Time) bool {
	return p.Days == 0 || p.Days == AllDays || p.Days&weekdayBit(day) != 0
}

func (p Period) covers(minute int) bool {
	if p.StartMin < p.EndMin {
		return minute >= p.StartMin && minute < p.EndMin
	}
	return minute >= p.StartMin
}

func minuteOf(t time.Time) int { return t.Hour()*60 + t.Minute() }

func weekdayBit(t time.Time) uint8 {
	d := int(t.Weekday())
	if d == 0 {
		return 1 << 6 // Sunday is day 7, not Go's 0.
	}
	return 1 << (d - 1)
}

func midnight(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func at(day time.Time, minute int) time.Time {
	return time.Date(day.Year(), day.Month(), day.Day(), minute/60, minute%60, 0, 0, day.Location())
}
