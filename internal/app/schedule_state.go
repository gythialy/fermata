package app

import (
	"time"

	"github.com/gythialy/fermata/internal/paths"
	"github.com/gythialy/fermata/internal/planner"
	"github.com/gythialy/fermata/internal/savedstate"
)

// initScheduleState finds the file that remembers the next break.
func (a *App) initScheduleState() {
	path, err := paths.Schedule()
	if err != nil {
		a.log.Info("no saved schedule", "error", err)
		return
	}
	a.statePath = path
}

// restoreSchedule resumes a schedule saved less than ten minutes ago, so a
// quick restart continues the countdown instead of starting it over. It
// reports whether it restored one.
func (a *App) restoreSchedule() bool {
	if a.statePath == "" || a.pl == nil {
		return false
	}
	st, ok := savedstate.Load(a.statePath, time.Now())
	if !ok {
		return false
	}
	a.apply(a.pl.Step(planner.Restore{
		Now:      time.Now(),
		N:        st.N,
		BreakAt:  st.BreakAt,
		NotifyAt: st.NotifyAt,
	}))
	state := a.pl.State()
	if state.Phase != planner.Waiting || !state.BreakAt.Equal(st.BreakAt) {
		return false
	}
	a.log.Info("schedule resumed", "n", st.N, "breakAt", st.BreakAt.Format(time.RFC3339))
	return true
}

// persistSchedule saves a waiting schedule, and forgets it otherwise: a
// break in progress or a pause does not come back after a restart.
func (a *App) persistSchedule() {
	if a.statePath == "" || a.pl == nil {
		return
	}
	st, _ := a.pl.Snapshot()
	if st.Phase != planner.Waiting && st.Phase != planner.Notified {
		if err := savedstate.Clear(a.statePath); err != nil && !a.stateWarned {
			a.stateWarned = true
			a.log.Info("schedule state is not writable", "error", err)
		}
		return
	}
	err := savedstate.Save(a.statePath, savedstate.State{
		SavedAt:  time.Now(),
		N:        st.N,
		BreakAt:  st.BreakAt,
		NotifyAt: st.NotifyAt,
	})
	if err != nil && !a.stateWarned {
		a.stateWarned = true
		a.log.Info("schedule state is not writable", "error", err)
	}
}
