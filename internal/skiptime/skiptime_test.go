package skiptime

import (
	"testing"
	"time"
)

var (
	wed = time.Date(2026, 11, 25, 9, 0, 0, 0, time.UTC)
	thu = wed.AddDate(0, 0, 1)
	fri = wed.AddDate(0, 0, 2)
	mon = wed.AddDate(0, 0, -2)
	sun = wed.AddDate(0, 0, 4)
)

const (
	monBit = 1 << 0
	wedBit = 1 << 2
	thuBit = 1 << 3
	sunBit = 1 << 6
)

func clock(day time.Time, hour, minute int) time.Time {
	return time.Date(day.Year(), day.Month(), day.Day(), hour, minute, 0, 0, day.Location())
}

var plusTwo = time.FixedZone("plus two", 2*60*60)

func TestActive(t *testing.T) {
	tests := []struct {
		name    string
		periods []Period
		at      time.Time
		want    bool
	}{
		{
			name:    "inside a same-day window",
			periods: []Period{{StartMin: 9 * 60, EndMin: 10 * 60}},
			at:      clock(wed, 9, 30),
			want:    true,
		},
		{
			name:    "before the start",
			periods: []Period{{StartMin: 9 * 60, EndMin: 10 * 60}},
			at:      clock(wed, 8, 59),
		},
		{
			name:    "at the start",
			periods: []Period{{StartMin: 9 * 60, EndMin: 10 * 60}},
			at:      clock(wed, 9, 0),
			want:    true,
		},
		{
			name:    "at the end",
			periods: []Period{{StartMin: 9 * 60, EndMin: 10 * 60}},
			at:      clock(wed, 10, 0),
		},
		{
			name:    "an unselected weekday",
			periods: []Period{{StartMin: 9 * 60, EndMin: 10 * 60, Days: wedBit}},
			at:      clock(thu, 9, 30),
		},
		{
			name:    "a selected weekday",
			periods: []Period{{StartMin: 9 * 60, EndMin: 10 * 60, Days: wedBit}},
			at:      clock(wed, 9, 30),
			want:    true,
		},
		{
			name:    "sunday is day seven",
			periods: []Period{{StartMin: 9 * 60, EndMin: 10 * 60, Days: sunBit}},
			at:      clock(sun, 9, 30),
			want:    true,
		},
		{
			name:    "all days",
			periods: []Period{{StartMin: 9 * 60, EndMin: 10 * 60, Days: AllDays}},
			at:      clock(thu, 9, 30),
			want:    true,
		},
		{
			name:    "overnight before midnight",
			periods: []Period{{StartMin: 22 * 60, EndMin: 6 * 60, Days: wedBit}},
			at:      clock(wed, 23, 30),
			want:    true,
		},
		{
			name:    "overnight after midnight",
			periods: []Period{{StartMin: 22 * 60, EndMin: 6 * 60, Days: wedBit}},
			at:      clock(thu, 2, 0),
			want:    true,
		},
		{
			name:    "at the overnight end",
			periods: []Period{{StartMin: 22 * 60, EndMin: 6 * 60, Days: wedBit}},
			at:      clock(thu, 6, 0),
		},
		{
			name:    "overnight does not reach a second day",
			periods: []Period{{StartMin: 22 * 60, EndMin: 6 * 60, Days: wedBit}},
			at:      clock(fri, 2, 0),
		},
		{
			name:    "overnight into a selected day",
			periods: []Period{{StartMin: 22 * 60, EndMin: 6 * 60, Days: wedBit | thuBit}},
			at:      clock(thu, 2, 0),
			want:    true,
		},
		{
			name:    "one of several periods matches",
			periods: []Period{{StartMin: 8 * 60, EndMin: 9 * 60}, {StartMin: 9 * 60, EndMin: 10 * 60}},
			at:      clock(wed, 9, 30),
			want:    true,
		},
		{
			name:    "invalid minutes are ignored",
			periods: []Period{{StartMin: 1500, EndMin: 1600}},
			at:      clock(wed, 9, 30),
		},
		{
			name:    "minutes use the time's location",
			periods: []Period{{StartMin: 9 * 60, EndMin: 10 * 60}},
			at:      time.Date(2026, 11, 25, 7, 30, 0, 0, time.UTC).In(plusTwo),
			want:    true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Active(tc.periods, tc.at); got != tc.want {
				t.Errorf("active %v, want %v", got, tc.want)
			}
		})
	}
}

func TestNextChange(t *testing.T) {
	tests := []struct {
		name    string
		periods []Period
		at      time.Time
		want    time.Time
	}{
		{
			name:    "inside a window, its end",
			periods: []Period{{StartMin: 9 * 60, EndMin: 10 * 60}},
			at:      clock(wed, 9, 30),
			want:    clock(wed, 10, 0),
		},
		{
			name:    "before a window, its start",
			periods: []Period{{StartMin: 9 * 60, EndMin: 10 * 60}},
			at:      clock(wed, 8, 30),
			want:    clock(wed, 9, 0),
		},
		{
			name:    "a boundary equal to now is skipped",
			periods: []Period{{StartMin: 9 * 60, EndMin: 10 * 60}},
			at:      clock(wed, 9, 0),
			want:    clock(wed, 10, 0),
		},
		{
			name:    "a second into the window",
			periods: []Period{{StartMin: 9 * 60, EndMin: 10 * 60}},
			at:      clock(wed, 9, 0).Add(time.Second),
			want:    clock(wed, 10, 0),
		},
		{
			name:    "at the end, the next day's start",
			periods: []Period{{StartMin: 9 * 60, EndMin: 10 * 60}},
			at:      clock(wed, 10, 0),
			want:    clock(thu, 9, 0),
		},
		{
			name:    "an unselected day looks ahead a week",
			periods: []Period{{StartMin: 9 * 60, EndMin: 10 * 60, Days: monBit}},
			at:      clock(wed, 9, 30),
			want:    clock(mon.AddDate(0, 0, 7), 9, 0),
		},
		{
			name:    "inside an overnight window",
			periods: []Period{{StartMin: 22 * 60, EndMin: 6 * 60}},
			at:      clock(wed, 23, 30),
			want:    clock(thu, 6, 0),
		},
		{
			name:    "before an overnight window",
			periods: []Period{{StartMin: 22 * 60, EndMin: 6 * 60}},
			at:      clock(wed, 21, 0),
			want:    clock(wed, 22, 0),
		},
		{
			name:    "after midnight in an overnight window",
			periods: []Period{{StartMin: 22 * 60, EndMin: 6 * 60, Days: wedBit}},
			at:      clock(thu, 2, 0),
			want:    clock(thu, 6, 0),
		},
		{
			name:    "a weekly overnight start a week away",
			periods: []Period{{StartMin: 22 * 60, EndMin: 6 * 60, Days: wedBit}},
			at:      clock(thu, 7, 0),
			want:    clock(wed.AddDate(0, 0, 7), 22, 0),
		},
		{
			name:    "the earliest of several boundaries wins",
			periods: []Period{{StartMin: 9 * 60, EndMin: 10 * 60}, {StartMin: 9 * 60, EndMin: 11 * 60}},
			at:      clock(wed, 9, 30),
			want:    clock(wed, 10, 0),
		},
		{
			name: "no periods",
			at:   clock(wed, 9, 30),
		},
		{
			name:    "invalid minutes are ignored",
			periods: []Period{{StartMin: -1, EndMin: 10 * 60}},
			at:      clock(wed, 9, 30),
		},
		{
			name:    "boundaries use the time's location",
			periods: []Period{{StartMin: 9 * 60, EndMin: 10 * 60}},
			at:      time.Date(2026, 11, 25, 7, 30, 0, 0, time.UTC).In(plusTwo),
			want:    time.Date(2026, 11, 25, 10, 0, 0, 0, plusTwo),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := NextChange(tc.periods, tc.at)
			if !got.Equal(tc.want) {
				t.Errorf("next change %v, want %v", got, tc.want)
			}
		})
	}
}
