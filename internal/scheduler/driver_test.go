package scheduler

import (
	"sync/atomic"
	"testing"
	"time"
)

func newTestDriver(fire func(time.Time)) *Driver {
	d := New(fire)
	d.dispatch = func(f func()) { f() }
	d.now = time.Now
	return d
}

func TestArmFires(t *testing.T) {
	fired := make(chan time.Time, 1)
	d := newTestDriver(func(at time.Time) { fired <- at })
	defer d.Close()
	d.Arm(time.Now().Add(10 * time.Millisecond))
	select {
	case at := <-fired:
		if at.IsZero() {
			t.Error("fired without a time")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the deadline did not fire")
	}
}

func TestArmReplacesTheDeadline(t *testing.T) {
	var count atomic.Int32
	ch := make(chan struct{}, 4)
	d := newTestDriver(func(time.Time) {
		count.Add(1)
		ch <- struct{}{}
	})
	defer d.Close()
	d.Arm(time.Now().Add(10 * time.Millisecond))
	d.Arm(time.Now().Add(80 * time.Millisecond))
	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatal("the new deadline did not fire")
	}
	time.Sleep(100 * time.Millisecond)
	if got := count.Load(); got != 1 {
		t.Errorf("fired %d times, want 1", got)
	}
}

func TestStopCancels(t *testing.T) {
	var count atomic.Int32
	d := newTestDriver(func(time.Time) { count.Add(1) })
	defer d.Close()
	d.Arm(time.Now().Add(20 * time.Millisecond))
	d.Stop()
	time.Sleep(100 * time.Millisecond)
	if got := count.Load(); got != 0 {
		t.Errorf("fired %d times after stop", got)
	}
}
