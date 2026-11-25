// Package scheduler wakes the app at the planner's single deadline.
package scheduler

import (
	"sync"
	"time"

	"github.com/egoist/mygo"
)

// Driver owns the one timer a schedule needs. Each Arm replaces the deadline;
// a callback that arrives for an older one is ignored.
type Driver struct {
	mu       sync.Mutex
	gen      uint64
	timer    *time.Timer
	stopped  bool
	fire     func(time.Time)
	dispatch func(func())
	now      func() time.Time
}

// New returns a driver that calls fire on the main thread.
func New(fire func(time.Time)) *Driver {
	return &Driver{fire: fire, dispatch: mygo.RunOnMain, now: time.Now}
}

// Arm sets the deadline, replacing the one before it.
func (d *Driver) Arm(at time.Time) {
	d.mu.Lock()
	d.gen++
	gen := d.gen
	if d.timer != nil {
		d.timer.Stop()
		d.timer = nil
	}
	if !d.stopped {
		delay := time.Until(at)
		if delay < 0 {
			delay = 0
		}
		d.timer = time.AfterFunc(delay, func() {
			d.dispatch(func() {
				d.mu.Lock()
				current := gen == d.gen && !d.stopped
				d.mu.Unlock()
				if current {
					d.fire(d.now())
				}
			})
		})
	}
	d.mu.Unlock()
}

// Stop cancels the deadline.
func (d *Driver) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.gen++
	if d.timer != nil {
		d.timer.Stop()
		d.timer = nil
	}
}

// Close stops the driver for good; a timer already due does nothing.
func (d *Driver) Close() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.stopped = true
	d.gen++
	if d.timer != nil {
		d.timer.Stop()
		d.timer = nil
	}
}
