package cli

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Duration parses the command line's durations: bare minutes ("30"), and
// hours, minutes and seconds with units ("1h20m", "90m", "45s"), spaces
// ignored.
func Duration(s string) (time.Duration, error) {
	t := strings.ReplaceAll(strings.TrimSpace(s), " ", "")
	if t == "" {
		return 0, usagef("empty duration")
	}
	if n, err := strconv.Atoi(t); err == nil {
		if n <= 0 {
			return 0, usagef("duration %q is not positive", s)
		}
		return time.Duration(n) * time.Minute, nil
	}
	var total time.Duration
	for i := 0; i < len(t); {
		j := i
		for j < len(t) && t[j] >= '0' && t[j] <= '9' {
			j++
		}
		if j == i || j == len(t) {
			return 0, usagef("duration %q is not a time such as 1h20m", s)
		}
		n, err := strconv.Atoi(t[i:j])
		if err != nil {
			return 0, usagef("duration %q is not a time such as 1h20m", s)
		}
		switch t[j] {
		case 'h':
			total += time.Duration(n) * time.Hour
		case 'm':
			total += time.Duration(n) * time.Minute
		case 's':
			total += time.Duration(n) * time.Second
		default:
			return 0, usagef("duration %q has no unit at %q", s, t[j:])
		}
		i = j + 1
	}
	if total <= 0 {
		return 0, usagef("duration %q is not positive", s)
	}
	return total, nil
}

// UntilMorning returns the next morning hour after now: today if it is still
// ahead, tomorrow otherwise.
func UntilMorning(now time.Time, hour int) time.Time {
	hour = min(max(hour, 0), 23)
	next := time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, now.Location())
	if !next.After(now) {
		next = next.AddDate(0, 0, 1)
	}
	return next
}

// String writes a duration the way the app's text does, m:ss or h:mm:ss.
func String(d time.Duration) string {
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
