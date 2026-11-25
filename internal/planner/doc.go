// Package planner is Fermata's break schedule, a pure state machine.
//
// It imports only the standard library and never reads the clock: each input
// carries the time, and a step returns the effects the app performs. Tests
// drive it without timers, and the app keeps the only timers there are.
package planner
