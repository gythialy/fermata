// Package presence watches the system for what pauses breaks: the user being
// away, Focus or Do Not Disturb, a locked screen, an excluded application. It
// reports events; the app decides what they mean to the planner.
package presence

import (
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/egoist/mygo"
)

// Kind is what happened.
type Kind uint8

const (
	// IdleEnter means the user has been away long enough to pause breaks.
	IdleEnter Kind = iota + 1
	// IdleLeave means the user is back.
	IdleLeave
	// DNDEnter means Focus or Do Not Disturb went on.
	DNDEnter
	// DNDLeave means it went off.
	DNDLeave
	// Lock means the screen locked or the system will sleep.
	Lock
	// Unlock means the screen came back or the system woke.
	Unlock
	// Suspend, then Resume, bracket a system sleep.
	Suspend
	Resume
	// ExclusionEnter means an app exclusion rule now pauses breaks.
	ExclusionEnter
	// ExclusionLeave means no rule matches any more.
	ExclusionLeave
	// Unavailable means a probe cannot read its system state.
	Unavailable
)

// String returns the event's name, for logs.
func (k Kind) String() string {
	switch k {
	case IdleEnter:
		return "idle-enter"
	case IdleLeave:
		return "idle-leave"
	case DNDEnter:
		return "dnd-enter"
	case DNDLeave:
		return "dnd-leave"
	case Lock:
		return "lock"
	case Unlock:
		return "unlock"
	case Suspend:
		return "suspend"
	case Resume:
		return "resume"
	case ExclusionEnter:
		return "exclusion-enter"
	case ExclusionLeave:
		return "exclusion-leave"
	case Unavailable:
		return "unavailable"
	default:
		return "unknown"
	}
}

// Event is one change in the system's presence.
type Event struct {
	Kind   Kind
	At     time.Time
	Detail string
}

// Rule is an app exclusion rule: while its processes run, breaks pause, or
// only then do breaks run.
type Rule struct {
	Name      string
	Processes []string
	// Pause tells that breaks pause while the processes run. When false
	// they run only while the processes run.
	Pause bool
}

// Source is what presence reads. The zero value is unusable; use
// SystemSource, or set the fields in a test.
type Source struct {
	IdleTime func() time.Duration
	ProbeDND func() (bool, error)
	// Running reports whether any of the processes is running.
	Running  func(processes []string) (bool, error)
	Power    bool
	Dispatch func(func())
	Now      func() time.Time
	Log      *slog.Logger
}

// Config is what the manager watches.
type Config struct {
	IdleEnabled  bool
	IdleReset    time.Duration
	DNDEnabled   bool
	Rule         *Rule
	PollInterval time.Duration
}

// idleAway is how long without input counts as being away.
const idleAway = 20 * time.Second

// defaultInterval is how often the monitors look, unless configured otherwise.
const defaultInterval = 2 * time.Second

// Manager watches the system and reports events on the main thread.
type Manager struct {
	src  Source
	emit func(Event)

	cfgMu sync.RWMutex
	cfg   Config

	// state is only read and written on the main thread, through Dispatch.
	away       bool
	idlePaused bool
	dndOn      bool
	excluded   bool
	warnedDND  bool
	warnedExcl bool
	// dndBroken stops the Focus probe for good once it fails: a permission
	// the user has not granted does not change by asking every two seconds,
	// and the system call costs a process each time.
	dndBroken atomic.Bool

	mu      sync.Mutex
	started bool
	stop    chan struct{}
	wg      sync.WaitGroup
	unsubs  []func()
}

// New returns a manager that reports to emit on the main thread.
func New(src Source, emit func(Event)) *Manager {
	if src.Dispatch == nil {
		src.Dispatch = mygo.RunOnMain
	}
	if src.Now == nil {
		src.Now = time.Now
	}
	if src.Log == nil {
		src.Log = slog.Default()
	}
	return &Manager{src: src, emit: emit, stop: make(chan struct{})}
}

// Configure changes what is watched. Changing a watched state resets it, and
// reports the pause ending so the app can resume.
func (m *Manager) Configure(cfg Config) {
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = defaultInterval
	}
	m.cfgMu.Lock()
	old := m.cfg
	m.cfg = cfg
	m.cfgMu.Unlock()
	if !cfg.IdleEnabled && m.idlePaused {
		m.idlePaused = false
		m.away = false
		m.emit(Event{Kind: IdleLeave, At: m.src.Now(), Detail: "disabled"})
	}
	if !cfg.DNDEnabled && m.dndOn {
		m.dndOn = false
		m.emit(Event{Kind: DNDLeave, At: m.src.Now(), Detail: "disabled"})
	}
	// A settings change is the moment to try the Focus probe once more, as
	// the user may have granted access since.
	m.dndBroken.Store(false)
	if !sameRule(old.Rule, cfg.Rule) && m.excluded {
		m.excluded = false
		m.emit(Event{Kind: ExclusionLeave, At: m.src.Now(), Detail: "rule changed"})
	}
}

// Start begins the monitors. Power events come from the system directly.
func (m *Manager) Start() {
	m.mu.Lock()
	if m.started {
		m.mu.Unlock()
		return
	}
	m.started = true
	stop := m.stop
	m.mu.Unlock()

	m.wg.Go(func() { m.loop(stop, m.watchIdle) })
	m.wg.Go(func() { m.loop(stop, m.watchDND) })
	m.wg.Go(func() { m.loop(stop, m.watchExclusion) })

	if m.src.Power {
		m.unsubs = append(m.unsubs,
			mygo.Power.OnSuspend(func() { m.send(Suspend) }),
			mygo.Power.OnResume(func() { m.send(Resume) }),
			mygo.Power.OnLockScreen(func() { m.send(Lock) }),
			mygo.Power.OnUnlockScreen(func() { m.send(Unlock) }),
		)
	}
}

// Stop ends the monitors.
func (m *Manager) Stop() {
	m.mu.Lock()
	if !m.started {
		m.mu.Unlock()
		return
	}
	m.started = false
	close(m.stop)
	m.wg.Wait()
	unsubs := m.unsubs
	m.unsubs = nil
	m.stop = make(chan struct{})
	m.mu.Unlock()
	for _, off := range unsubs {
		off()
	}
}

// send reports a power event on the main thread.
func (m *Manager) send(k Kind) {
	m.src.Dispatch(func() {
		m.emit(Event{Kind: k, At: m.src.Now()})
	})
}

// loop calls watch every interval until stop closes.
func (m *Manager) loop(stop chan struct{}, watch func()) {
	ticker := time.NewTicker(m.interval())
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			watch()
		}
	}
}

// interval is how often the monitors look.
func (m *Manager) interval() time.Duration {
	m.cfgMu.RLock()
	defer m.cfgMu.RUnlock()
	if m.cfg.PollInterval <= 0 {
		return defaultInterval
	}
	return m.cfg.PollInterval
}

// config returns a copy of what is watched.
func (m *Manager) config() Config {
	m.cfgMu.RLock()
	defer m.cfgMu.RUnlock()
	return m.cfg
}

func (m *Manager) watchIdle() {
	if !m.config().IdleEnabled || m.src.IdleTime == nil {
		return
	}
	idle := m.src.IdleTime()
	m.src.Dispatch(func() { m.PollIdle(m.src.Now(), idle) })
}

func (m *Manager) watchDND() {
	if !m.config().DNDEnabled || m.src.ProbeDND == nil || m.dndBroken.Load() {
		return
	}
	on, err := m.src.ProbeDND()
	m.src.Dispatch(func() { m.PollDND(on, err) })
}

func (m *Manager) watchExclusion() {
	rule := m.config().Rule
	if rule == nil || m.src.Running == nil {
		// The state belongs to the main thread, even when there is nothing to
		// poll.
		m.src.Dispatch(func() { m.PollExclusion(nil, false, nil) })
		return
	}
	running, err := m.src.Running(rule.Processes)
	m.src.Dispatch(func() { m.PollExclusion(rule, running, err) })
}

// PollIdle applies the idle state at now; the monitors and tests call it on
// the main thread.
func (m *Manager) PollIdle(now time.Time, idle time.Duration) {
	cfg := m.config()
	if !cfg.IdleEnabled {
		return
	}
	if idle >= idleAway {
		m.away = true
		if !m.idlePaused && idle >= cfg.IdleReset {
			m.idlePaused = true
			m.emit(Event{Kind: IdleEnter, At: now, Detail: idle.String()})
		}
		return
	}
	if !m.away {
		return
	}
	m.away = false
	paused := m.idlePaused
	m.idlePaused = false
	detail := "back"
	if paused {
		detail = "back from away"
	}
	m.emit(Event{Kind: IdleLeave, At: now, Detail: detail})
}

// PollDND applies a Focus reading.
func (m *Manager) PollDND(on bool, err error) {
	if err != nil {
		// The system will not tell the app Focus until the user grants it
		// access; stop asking for this run.
		m.dndBroken.Store(true)
		if !m.warnedDND {
			m.warnedDND = true
			m.emit(Event{Kind: Unavailable, At: m.src.Now(), Detail: "focus: " + err.Error()})
		}
		on = false
	}
	if !m.config().DNDEnabled {
		return
	}
	if on == m.dndOn {
		return
	}
	m.dndOn = on
	if on {
		m.emit(Event{Kind: DNDEnter, At: m.src.Now()})
	} else {
		m.emit(Event{Kind: DNDLeave, At: m.src.Now()})
	}
}

// PollExclusion applies whether a rule's processes run.
func (m *Manager) PollExclusion(rule *Rule, running bool, err error) {
	if err != nil {
		if !m.warnedExcl {
			m.warnedExcl = true
			m.emit(Event{Kind: Unavailable, At: m.src.Now(), Detail: "processes: " + err.Error()})
		}
		running = false
	}
	hit := false
	if rule != nil {
		hit = running == rule.Pause
	}
	if hit == m.excluded {
		return
	}
	m.excluded = hit
	if hit {
		m.emit(Event{Kind: ExclusionEnter, At: m.src.Now(), Detail: rule.Name})
	} else {
		m.emit(Event{Kind: ExclusionLeave, At: m.src.Now(), Detail: rule.Name})
	}
}

func sameRule(a, b *Rule) bool {
	switch {
	case a == nil && b == nil:
		return true
	case a == nil || b == nil:
		return false
	}
	if a.Name != b.Name || a.Pause != b.Pause || len(a.Processes) != len(b.Processes) {
		return false
	}
	for i := range a.Processes {
		if a.Processes[i] != b.Processes[i] {
			return false
		}
	}
	return true
}
