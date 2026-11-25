package planner

import "time"

// schedule waits one gap after anchor for the n-th break of the sequence.
// The notice comes lead before the break, or not at all when it would not fit.
func (p *Planner) schedule(n int, anchor time.Time) {
	cfg := p.cfg
	n = max(n, 1)
	kind := cfg.KindFor(n)
	if p.st.force != nil {
		// A break the user skipped to keeps its kind's notice, whatever the
		// sequence would put next.
		kind = *p.st.force
	}
	gap := cfg.Gap()
	if gap <= 0 {
		gap = time.Minute
	}
	breakAt := anchor.Add(gap)
	var notifyAt time.Time
	if lead := cfg.For(kind).LeadTime(); lead > 0 {
		lead = min(lead, gap/2)
		notifyAt = breakAt.Add(-lead)
	}
	p.st.Phase = Waiting
	p.st.Kind = kind
	p.st.N = n
	p.st.BreakAt = breakAt
	p.st.NotifyAt = notifyAt
	p.st.StartAt, p.st.EndAt, p.st.Duration = time.Time{}, time.Time{}, 0
	p.st.Strict, p.st.ManualFinish = false, false
	p.st.postpone = PostponeConfig{}
	p.st.curTitle, p.st.curText = "", ""
	p.st.anchor = anchor
}

// schedulePostponed waits the postponed break's delay after now. The break
// keeps its number and kind: it is still the same break.
func (p *Planner) schedulePostponed(now time.Time) {
	c := p.cfg.For(p.st.Kind)
	delay := p.st.postpone.Time
	if delay <= 0 {
		delay = c.Postpone.Time
	}
	if delay <= 0 {
		delay = c.Interval
	}
	if delay <= 0 {
		delay = time.Minute
	}
	breakAt := now.Add(delay)
	var notifyAt time.Time
	if lead := c.LeadTime(); lead > 0 && delay > lead {
		notifyAt = breakAt.Add(-lead)
	}
	p.st.Phase = Waiting
	p.st.BreakAt = breakAt
	p.st.NotifyAt = notifyAt
	p.st.StartAt, p.st.EndAt, p.st.Duration = time.Time{}, time.Time{}, 0
	p.st.Strict, p.st.ManualFinish = false, false
	p.st.anchor = now
}
