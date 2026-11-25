// Package units writes a duration the way the interface names it in a
// stepper or a summary: minutes when it is whole minutes, seconds otherwise.
package units

import (
	"fmt"

	"github.com/gythialy/fermata/locales"
)

// Short writes ms as "5 min" or "20 s".
func Short(ms int) string {
	if ms >= 60_000 && ms%60_000 == 0 {
		return locales.UnitMinutes(ms / 60_000)
	}
	return locales.UnitSeconds(ms / 1000)
}

// Compact writes ms for the rhythm chips: "20s", "5m", "1h".
func Compact(ms int) string {
	switch {
	case ms >= 3_600_000 && ms%3_600_000 == 0:
		return fmt.Sprintf("%dh", ms/3_600_000)
	case ms >= 60_000 && ms%60_000 == 0:
		return locales.UnitMinutesShort(ms / 60_000)
	default:
		return locales.UnitSecondsShort(ms / 1000)
	}
}
