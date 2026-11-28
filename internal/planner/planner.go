package planner

import (
	"time"

	"github.com/gythialy/fermata/internal/skiptime"
)

// lateGrace is how late a break may start and still count: a timer that fires
// within this is a timer, later than it is a sleep or a stalled process, and
// the break is missed instead of shown.
const lateGrace = 2 * time.Second

// defaultSkipToDelay is the wait before a break the user skipped to, so that
// the previous window closes first.
const defaultSkipToDelay = 100 * time.Millisecond

// Planner holds the schedule. Its zero value is idle; use New.
type Planner struct {
	cfg Config
	st  State
}

// New returns a planner that is idle until its first Reset, as the app is
// before the welcome window closes.
func New(cfg Config) *Planner { return &Planner{cfg: cfg} }

// Config returns the configuration in force.
func (p *Planner) Config() Config { return p.cfg }

// State returns a copy of the state.
func (p *Planner) State() State { return p.st }

// Step applies one input, then every deadline that is due at its time, and
// returns the effects to perform, in order.
func (p *Planner) Step(in Input) []Effect {
	now := nowOf(in)
	before := p.st.signature()
	var eff []Effect
	switch in := in.(type) {
	case Tick, Wake:
		// Time passing has no change of its own.
	case Pause:
		eff = p.pause(in)
	case Resume:
		eff = p.resume(in)
	case Postpone:
		eff = p.postpone(in.Now)
	case Skip:
		eff = p.skip(in.Now)
	case Finish:
		eff = p.finish(in.Now)
	case Show:
		eff = p.show(in.Now)
	case Restore:
		eff = p.restore(in)
	case Advance:
		eff = p.advanceBreak(in.Now)
	case Defer:
		eff = p.deferBreak(in)

	case Reset:
		eff = p.reset(in.Now)
	case SkipTo:
		eff = p.skipTo(in)
	case NextText:
		p.st.title, p.st.text = in.Title, in.Text
	case SetConfig:
		p.setConfig(in)
	}
	eff = append(eff, p.syncSkipPeriods(now)...)
	grace := lateGrace
	if _, ok := in.(Wake); ok {
		// A wake never shows a break the sleep hid.
		grace = 0
	}
	eff = append(eff, p.advance(now, grace)...)
	eff = p.withDeadline(eff, now)
	if before != p.st.signature() {
		eff = append(eff, TrayDirty{})
	}
	return eff
}

// advance applies every deadline that is due at now, in order. A break due
// within grace starts; one due earlier is missed, never shown late.
func (p *Planner) advance(now time.Time, grace time.Duration) []Effect {
	var eff []Effect
	for range 8 {
		switch p.st.Phase {
		case Paused:
			if !p.st.Reasons.Has(Manual) || p.st.PauseUntil.IsZero() || now.Before(p.st.PauseUntil) {
				return eff
			}
			eff = append(eff, p.expirePause(now)...)
		case Waiting:
			if !p.st.BreakAt.IsZero() && !now.Before(p.st.BreakAt) {
				if now.Sub(p.st.BreakAt) > grace {
					eff = append(eff, p.miss(now)...)
					continue
				}
				eff = append(eff, p.start(now)...)
				continue
			}
			if !p.st.NotifyAt.IsZero() && !now.Before(p.st.NotifyAt) {
				p.st.Phase = Notified
				eff = append(eff, Notify{Kind: p.st.Kind, Lead: p.cfg.For(p.st.Kind).LeadTime()})
				continue
			}
			return eff
		case Notified:
			if now.Before(p.st.BreakAt) {
				return eff
			}
			if now.Sub(p.st.BreakAt) > grace {
				eff = append(eff, p.miss(now)...)
				continue
			}
			eff = append(eff, p.start(now)...)
		case Breaking:
			if now.Before(p.st.EndAt) {
				return eff
			}
			eff = append(eff, p.endBreak(now)...)
		default:
			return eff
		}
	}
	return eff
}

// start opens the break that is due at now.
func (p *Planner) start(now time.Time) []Effect {
	kind := p.st.Kind
	if p.st.force != nil {
		kind = *p.st.force
	}
	c := p.cfg.For(kind)
	duration, strict, manual, postpone := c.Duration, c.Strict, c.ManualFinish, c.Postpone
	title, text := p.st.title, p.st.text
	if o := p.st.restart; o != nil {
		duration, strict, manual, postpone = o.Duration, o.Strict, o.ManualFinish, o.Postpone
		title, text = o.Title, o.Text
	}
	spec := BreakSpec{
		Kind:         kind,
		Number:       p.st.N,
		StartAt:      now,
		EndAt:        now.Add(duration),
		Duration:     duration,
		Strict:       strict,
		ManualFinish: manual,
		Postpone:     postpone,
		Title:        title,
		Text:         text,
	}
	p.st.Phase = Breaking
	p.st.Kind = kind
	p.st.StartAt, p.st.EndAt, p.st.Duration = now, spec.EndAt, duration
	p.st.Strict, p.st.ManualFinish = strict, manual
	p.st.postpone = postpone
	p.st.curTitle, p.st.curText = title, text
	p.st.NotifyAt = time.Time{}
	p.st.Missed = false
	p.st.restart = nil
	p.st.force = nil
	p.st.title, p.st.text = "", ""
	return []Effect{CloseNotice{}, StartBreak{Spec: spec}, PlaySound{Name: SoundStart}}
}

// endBreak closes a break whose countdown ran out at now: a manual finish
// waits for the user, any other break schedules the next one.
func (p *Planner) endBreak(now time.Time) []Effect {
	if p.st.ManualFinish {
		p.st.Phase = ManualAwait
		return []Effect{PlaySound{Name: SoundEnd}}
	}
	eff := []Effect{CloseBreak{Reason: Completed}, PlaySound{Name: SoundEnd}}
	p.st.Postpones = 0
	p.st.N++
	p.st.postpone = PostponeConfig{}
	p.st.curTitle, p.st.curText = "", ""
	if !p.st.Reasons.Empty() {
		p.st.Phase = Paused
		p.st.N = 0
		p.st.PauseUntil = time.Time{}
		p.st.StartAt, p.st.EndAt, p.st.Duration = time.Time{}, time.Time{}, 0
		return eff
	}
	anchor := p.st.EndAt
	if now.Sub(anchor) > lateGrace {
		// The app was not running when the break ended: the next wait begins
		// now, as the sleep rules say.
		anchor = now
	}
	p.schedule(p.st.N, anchor)
	return eff
}

// miss skips the break a sleep or a stalled process made the app miss.
func (p *Planner) miss(now time.Time) []Effect {
	var eff []Effect
	if p.st.Phase == Notified {
		eff = append(eff, CloseNotice{})
	}
	p.st.Missed = true
	p.st.Postpones = 0
	p.st.N++
	p.st.force = nil
	p.st.restart = nil
	p.st.postpone = PostponeConfig{}
	p.st.curTitle, p.st.curText = "", ""
	p.schedule(p.st.N, now)
	return eff
}

// pause adds a pause reason and closes what that reason closes.
func (p *Planner) pause(in Pause) []Effect {
	if in.Reason == 0 {
		return nil
	}
	if p.strict() && in.Reason != Lock && in.Reason != SkipPeriod {
		// A strict break cannot be paused by the user; a locked screen or a
		// sleeping system still ends it. A skip period holds the next break.
		return nil
	}
	p.st.Reasons = p.st.Reasons.With(in.Reason)
	if in.Reason == Manual {
		p.st.PauseUntil = in.Until
	}
	var eff []Effect
	switch in.Reason {
	case Manual, Lock:
		eff = append(eff, p.closeForPause()...)
		p.st.Phase = Paused
		p.st.N = 0
		p.st.Postpones = 0
		p.st.force = nil
		p.st.restart = nil
		p.st.postpone = PostponeConfig{}
	case DND, Idle, Exclusion, SkipPeriod:
		switch p.st.Phase {
		case Waiting, Notified:
			eff = append(eff, p.closeForPause()...)
			p.st.Phase = Paused
			p.st.N = 0
			p.st.Postpones = 0
			p.st.force = nil
			p.st.restart = nil
			p.st.postpone = PostponeConfig{}
		case Breaking, ManualAwait, Paused, Preparing:
			// A running break goes on; the reason decides about the next one.
		}
	}
	return eff
}

// closeForPause closes the notice and the break a pause ends, if any.
func (p *Planner) closeForPause() []Effect {
	var eff []Effect
	switch p.st.Phase {
	case Notified:
		eff = append(eff, CloseNotice{})
	case Breaking, ManualAwait:
		eff = append(eff, CloseBreak{Reason: Interrupted})
	}
	p.st.NotifyAt = time.Time{}
	p.st.BreakAt = time.Time{}
	p.st.StartAt, p.st.EndAt, p.st.Duration = time.Time{}, time.Time{}, 0
	p.st.Strict, p.st.ManualFinish = false, false
	p.st.curTitle, p.st.curText = "", ""
	return eff
}

// resume removes a pause reason, and schedules again once none is left.
func (p *Planner) resume(in Resume) []Effect {
	if !p.st.Reasons.Has(in.Reason) {
		return nil
	}
	p.st.Reasons = p.st.Reasons.Without(in.Reason)
	if in.Reason == Manual {
		p.st.PauseUntil = time.Time{}
	}
	if p.st.Phase != Paused || !p.st.Reasons.Empty() {
		return nil
	}
	p.schedule(1, in.Now)
	if in.Reason == Manual {
		return []Effect{NotifyResume{}}
	}
	return nil
}

// expirePause ends a timed manual pause that is due at now.
func (p *Planner) expirePause(now time.Time) []Effect {
	p.st.Reasons = p.st.Reasons.Without(Manual)
	p.st.PauseUntil = time.Time{}
	if !p.st.Reasons.Empty() {
		return nil
	}
	p.schedule(1, now)
	return []Effect{NotifyResume{}}
}

// postpone moves the running break later.
func (p *Planner) postpone(now time.Time) []Effect {
	if p.st.Phase != Breaking || p.st.Strict {
		return nil
	}
	if !p.spec().CanPostpone(now, p.st.Postpones) {
		return nil
	}
	p.st.restart = &breakOverride{
		Duration:     p.st.Duration,
		Strict:       p.st.Strict,
		ManualFinish: p.st.ManualFinish,
		Postpone:     p.st.postpone,
		Title:        p.st.curTitle,
		Text:         p.st.curText,
	}
	p.st.Postpones++
	eff := []Effect{CloseBreak{Reason: Postponed}}
	if !p.st.Reasons.Empty() {
		p.st.Phase = Paused
		p.st.N = 0
		p.st.PauseUntil = time.Time{}
		p.st.StartAt, p.st.EndAt, p.st.Duration = time.Time{}, time.Time{}, 0
		return eff
	}
	p.schedulePostponed(now)
	return eff
}

// skip ends the running break early. It is impossible while the break may be
// postponed, so that one action is offered at a time.
func (p *Planner) skip(now time.Time) []Effect {
	if p.st.Phase != Breaking || p.st.Strict {
		return nil
	}
	if p.spec().CanPostpone(now, p.st.Postpones) {
		return nil
	}
	eff := []Effect{CloseBreak{Reason: Skipped}}
	p.st.Postpones = 0
	p.st.N++
	p.st.postpone = PostponeConfig{}
	p.st.curTitle, p.st.curText = "", ""
	if !p.st.Reasons.Empty() {
		p.st.Phase = Paused
		p.st.N = 0
		p.st.PauseUntil = time.Time{}
		p.st.StartAt, p.st.EndAt, p.st.Duration = time.Time{}, time.Time{}, 0
		return eff
	}
	p.schedule(p.st.N, now)
	return eff
}

// show starts a break the app held past its deadline: a fullscreen app or
// typing. The break begins now, with its whole duration.
func (p *Planner) show(now time.Time) []Effect {
	switch p.st.Phase {
	case Waiting, Notified:
	default:
		return nil
	}
	if now.Before(p.st.BreakAt) {
		return nil
	}
	return p.start(now)
}

// restore resumes a saved schedule in a fresh planner.
func (p *Planner) restore(in Restore) []Effect {
	switch p.st.Phase {
	case Preparing, Waiting:
	default:
		return nil
	}
	if !p.st.BreakAt.IsZero() {
		return nil
	}
	if in.BreakAt.IsZero() || !in.BreakAt.After(in.Now) {
		return nil
	}
	n := max(in.N, 1)
	notify := in.NotifyAt
	if !notify.IsZero() && (!notify.After(in.Now) || !notify.Before(in.BreakAt)) {
		notify = time.Time{}
	}
	p.st.Phase = Waiting
	p.st.Kind = p.cfg.KindFor(n)
	p.st.N = n
	p.st.BreakAt = in.BreakAt
	p.st.NotifyAt = notify
	at := in.BreakAt
	if !notify.IsZero() {
		at = notify
	}
	return []Effect{TrayDirty{}, Arm{At: at}}
}

// advanceBreak starts the next break now, as the heads-up card's Start now.
func (p *Planner) advanceBreak(now time.Time) []Effect {
	switch p.st.Phase {
	case Waiting, Notified:
	default:
		return nil
	}
	if now.Before(p.st.BreakAt) || p.st.Phase == Notified {
		return p.start(now)
	}
	return p.show(now)
}

// deferBreak pushes the next break back by in.By.
func (p *Planner) deferBreak(in Defer) []Effect {
	switch p.st.Phase {
	case Waiting, Notified:
	default:
		return nil
	}
	if in.By <= 0 {
		return nil
	}
	var eff []Effect
	if p.st.Phase == Notified {
		eff = append(eff, CloseNotice{})
	}
	base := p.st.BreakAt
	if base.Before(in.Now) {
		base = in.Now
	}
	breakAt := base.Add(in.By)
	var notifyAt time.Time
	if lead := p.cfg.For(p.st.Kind).LeadTime(); lead > 0 && breakAt.Sub(in.Now) > lead {
		notifyAt = breakAt.Add(-lead)
	}
	p.st.Phase = Waiting
	p.st.BreakAt = breakAt
	p.st.NotifyAt = notifyAt
	p.st.anchor = in.Now
	at := breakAt
	if !notifyAt.IsZero() {
		at = notifyAt
	}
	eff = append(eff, TrayDirty{}, Arm{At: at})
	return eff
}

// finish closes a break that waited for the user.
func (p *Planner) finish(now time.Time) []Effect {
	if p.st.Phase != ManualAwait {
		return nil
	}
	eff := []Effect{CloseBreak{Reason: Completed}}
	p.st.Postpones = 0
	p.st.N++
	p.st.postpone = PostponeConfig{}
	p.st.curTitle, p.st.curText = "", ""
	p.st.Strict, p.st.ManualFinish = false, false
	p.st.StartAt, p.st.EndAt, p.st.Duration = time.Time{}, time.Time{}, 0
	if !p.st.Reasons.Empty() {
		p.st.Phase = Paused
		p.st.N = 0
		p.st.PauseUntil = time.Time{}
		return eff
	}
	p.schedule(p.st.N, now)
	return eff
}

// reset forgets the sequence and schedules a first break again, unless the
// app is paused.
func (p *Planner) reset(now time.Time) []Effect {
	if p.strict() {
		return nil
	}
	var eff []Effect
	switch p.st.Phase {
	case Notified:
		eff = append(eff, CloseNotice{})
	case Breaking, ManualAwait:
		eff = append(eff, CloseBreak{Reason: Interrupted})
	}
	p.st.Postpones = 0
	p.st.force = nil
	p.st.restart = nil
	p.st.postpone = PostponeConfig{}
	p.st.curTitle, p.st.curText = "", ""
	if !p.st.Reasons.Empty() {
		p.st.Phase = Paused
		p.st.N = 0
		p.st.NotifyAt = time.Time{}
		p.st.BreakAt = time.Time{}
		p.st.StartAt, p.st.EndAt, p.st.Duration = time.Time{}, time.Time{}, 0
		p.st.Strict, p.st.ManualFinish = false, false
		return eff
	}
	p.schedule(1, now)
	return eff
}

// skipTo starts a break of the requested kind soon.
func (p *Planner) skipTo(in SkipTo) []Effect {
	if p.st.Phase == Paused || !p.cfg.For(in.Kind).Enabled {
		return nil
	}
	if p.strict() {
		return nil
	}
	var eff []Effect
	switch p.st.Phase {
	case Notified:
		eff = append(eff, CloseNotice{})
	case Breaking, ManualAwait:
		eff = append(eff, CloseBreak{Reason: Interrupted})
	}
	n := max(p.st.N, 1)
	if in.Kind == Long {
		n = p.cfg.EveryN() + 1
	} else if p.cfg.Mini.Enabled && p.cfg.Long.Enabled && n%(p.cfg.EveryN()+1) == 0 {
		n = 1
	}
	delay := in.Delay
	if delay <= 0 {
		delay = defaultSkipToDelay
	}
	kind := in.Kind
	p.st.Phase = Waiting
	p.st.Kind = kind
	p.st.N = n
	p.st.BreakAt = in.Now.Add(delay)
	p.st.NotifyAt = time.Time{}
	p.st.Postpones = 0
	p.st.force = &kind
	p.st.restart = nil
	p.st.postpone = PostponeConfig{}
	p.st.title, p.st.text = in.Title, in.Text
	p.st.anchor = in.Now
	return eff
}

// setConfig replaces the configuration. A running break keeps the snapshot it
// started with; a waiting one is computed again from when it began waiting.
func (p *Planner) setConfig(in SetConfig) {
	p.cfg = in.Config
	n := max(p.st.N, 1)
	anchor := p.st.anchor
	if anchor.IsZero() {
		anchor = in.Now
	}
	switch p.st.Phase {
	case Waiting:
		p.schedule(n, anchor)
		if !p.st.NotifyAt.IsZero() && !in.Now.Before(p.st.NotifyAt) {
			// The notice would be late: skip it and wait for the break.
			p.st.NotifyAt = time.Time{}
		}
	case Notified:
		// A notice already sent is not taken back, and the type it named
		// stays for this break; only the time of the break follows.
		kind, notifyAt := p.st.Kind, p.st.NotifyAt
		p.schedule(n, anchor)
		p.st.Kind, p.st.NotifyAt, p.st.Phase = kind, notifyAt, Notified
	case Preparing, Paused, Breaking, ManualAwait:
		// Nothing to compute: a paused app has no deadline and a running
		// break keeps the snapshot it started with.
	}
}

// syncSkipPeriods pauses on entry and resumes on exit, so the wait starts
// again from the moment breaks become allowed.
func (p *Planner) syncSkipPeriods(now time.Time) []Effect {
	if !p.cfg.SkipPeriodsEnabled || len(p.cfg.SkipPeriods) == 0 {
		if p.st.Reasons.Has(SkipPeriod) {
			return p.resume(Resume{Now: now, Reason: SkipPeriod})
		}
		return nil
	}
	active := skiptime.Active(p.cfg.SkipPeriods, now)
	switch {
	case active && !p.st.Reasons.Has(SkipPeriod):
		return p.pause(Pause{Now: now, Reason: SkipPeriod})
	case !active && p.st.Reasons.Has(SkipPeriod):
		return p.resume(Resume{Now: now, Reason: SkipPeriod})
	}
	return nil
}

// withDeadline adds the next wake-up time, moving an existing one earlier
// when a skip period boundary comes first.
func (p *Planner) withDeadline(eff []Effect, now time.Time) []Effect {
	boundary := p.skipBoundary(now)
	if boundary.IsZero() {
		return p.plainDeadline(eff)
	}
	for i, e := range eff {
		switch v := e.(type) {
		case Arm:
			if boundary.Before(v.At) {
				eff[i] = Arm{At: boundary}
			}
			return eff
		case Disarm:
			eff[i] = Arm{At: boundary}
			return eff
		}
	}
	if at := p.deadline(); !at.IsZero() && at.Before(boundary) {
		return append(eff, Arm{At: at})
	}
	return append(eff, Arm{At: boundary})
}

// plainDeadline arms the planner's own deadline, unless the effects carry one.
func (p *Planner) plainDeadline(eff []Effect) []Effect {
	for _, e := range eff {
		switch e.(type) {
		case Arm, Disarm:
			return eff
		}
	}
	if at := p.deadline(); !at.IsZero() {
		return append(eff, Arm{At: at})
	}
	return append(eff, Disarm{})
}

// skipBoundary is when the scheduler must wake to enter or leave a period.
func (p *Planner) skipBoundary(now time.Time) time.Time {
	if !p.cfg.SkipPeriodsEnabled || len(p.cfg.SkipPeriods) == 0 {
		return time.Time{}
	}
	if p.st.Phase == Paused && p.st.Reasons.Has(Manual) && p.st.PauseUntil.IsZero() {
		return time.Time{}
	}
	return skiptime.NextChange(p.cfg.SkipPeriods, now)
}

// deadline is when the planner next needs a Tick, zero for none.
func (p *Planner) deadline() time.Time {
	switch p.st.Phase {
	case Waiting:
		if !p.st.NotifyAt.IsZero() && (p.st.BreakAt.IsZero() || p.st.NotifyAt.Before(p.st.BreakAt)) {
			return p.st.NotifyAt
		}
		return p.st.BreakAt
	case Notified:
		return p.st.BreakAt
	case Breaking:
		return p.st.EndAt
	case Paused:
		if p.st.Reasons.Has(Manual) {
			return p.st.PauseUntil
		}
		return time.Time{}
	}
	return time.Time{}
}

// spec returns the running break's snapshot, as the view sees it.
func (p *Planner) spec() BreakSpec {
	return BreakSpec{
		Kind:         p.st.Kind,
		Number:       p.st.N,
		StartAt:      p.st.StartAt,
		EndAt:        p.st.EndAt,
		Duration:     p.st.Duration,
		Strict:       p.st.Strict,
		ManualFinish: p.st.ManualFinish,
		Postpone:     p.st.postpone,
		Title:        p.st.curTitle,
		Text:         p.st.curText,
	}
}

// Snapshot returns the state and the running break's snapshot, zero while no
// break runs.
func (p *Planner) Snapshot() (State, BreakSpec) {
	if p.st.Phase == Breaking || p.st.Phase == ManualAwait {
		return p.st, p.spec()
	}
	return p.st, BreakSpec{}
}

// strict reports whether a strict break is running, countdown or manual.
func (p *Planner) strict() bool {
	return p.st.Strict && (p.st.Phase == Breaking || p.st.Phase == ManualAwait)
}

// CanPause reports whether the menu offers pausing.
func (p *Planner) CanPause() bool {
	switch p.st.Phase {
	case Preparing, Waiting, Notified, Breaking, ManualAwait:
		return !p.strict()
	}
	return false
}

// CanResume reports whether the menu offers resuming, which only a manual
// pause does: the other reasons end on their own.
func (p *Planner) CanResume() bool {
	return p.st.Phase == Paused && p.st.Reasons.Has(Manual)
}

// CanReset reports whether the menu offers resetting the sequence.
func (p *Planner) CanReset() bool {
	switch p.st.Phase {
	case Waiting, Notified, Breaking, ManualAwait:
		return !p.strict()
	}
	return false
}

// CanSkipTo reports whether the menu offers skipping to a break.
func (p *Planner) CanSkipTo() bool {
	switch p.st.Phase {
	case Waiting, Notified, Breaking, ManualAwait:
		return !p.strict()
	}
	return false
}

// CanPostpone reports whether the running break may be postponed at now.
func (p *Planner) CanPostpone(now time.Time) bool {
	return p.st.Phase == Breaking && p.spec().CanPostpone(now, p.st.Postpones)
}

// CanSkip reports whether the running break may be skipped at now.
func (p *Planner) CanSkip(now time.Time) bool {
	return p.st.Phase == Breaking && !p.st.Strict && !p.spec().CanPostpone(now, p.st.Postpones)
}

// CanFinish reports whether a break waits for the user to finish it.
func (p *Planner) CanFinish() bool { return p.st.Phase == ManualAwait }
