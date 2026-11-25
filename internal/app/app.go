// Package app is Fermata's coordinator: it owns the planner, the windows,
// the tray, the timers and the system events, and it is the only package that
// steps the planner.
package app

import (
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/gythialy/fermata/internal/breakwin"
	"github.com/gythialy/fermata/internal/cli"
	"github.com/gythialy/fermata/internal/fonts"
	"github.com/gythialy/fermata/internal/ideas"
	"github.com/gythialy/fermata/internal/logging"
	"github.com/gythialy/fermata/internal/notify"
	"github.com/gythialy/fermata/internal/paths"
	"github.com/gythialy/fermata/internal/planner"
	"github.com/gythialy/fermata/internal/prefs"
	"github.com/gythialy/fermata/internal/presence"
	"github.com/gythialy/fermata/internal/scheduler"
	"github.com/gythialy/fermata/internal/settings"
	"github.com/gythialy/fermata/internal/sounds"
	"github.com/gythialy/fermata/internal/tray"
	"github.com/gythialy/fermata/internal/trayicon"
	"github.com/gythialy/fermata/internal/welcome"
	"github.com/gythialy/fermata/locales"
)

// Options are the app's identity and the command it was launched with.
type Options struct {
	Version string
	Build   string
	Commit  string
	// Initial is the command line the first instance was started with,
	// besides starting the app.
	Initial cli.Command
}

// App is the running application.
type App struct {
	opts Options
	log  *slog.Logger
	logs io.Closer

	store    *settings.Store
	pl       *planner.Planner
	drv      *scheduler.Driver
	tray     *tray.Controller
	breaks   *breakwin.Manager
	prefs    *prefs.Window
	welcome  *welcome.Window
	pres     *presence.Manager
	notifier *notify.Notifier
	sounds   *sounds.Player
	ideas    *ideas.Deck

	danger      int
	previewIdea ideas.Idea
	lastSpec    planner.BreakSpec
	lastIdea    ideas.Idea

	started     bool
	welcomeOpen bool
	pending     []cli.Command
	quitting    bool
	corrupt     bool

	popover      *mygo.Window
	popoverTimer *time.Timer

	shortcutAcc string
	shortcutOn  bool
	trayTimer   *time.Timer
	updateTimer *time.Timer

	// The app holds a due break while a fullscreen app is in front or the
	// user is typing, then shows it with its whole duration.
	holdReason       string
	holdStart        time.Time
	fullscreenActive bool
	fullscreenBroken bool
	fullscreenTimer  *time.Timer
	bgPool           []string
	lastBg           string
	trayRefresh      time.Duration

	loginOn       bool
	langTag       string
	focusReadable bool
	statePath     string
	stateWarned   bool
	updateVersion string
	updateURL     string
}

// New returns the app, to be started from App.WhenReady.
func New(opts Options) *App {
	return &App{opts: opts}
}

// Start loads everything and opens the app's first surface.
func (a *App) Start() {
	if a.started {
		return
	}
	a.started = true
	a.startLogging()
	if err := fonts.Load(); err != nil {
		// The system's serifs still draw the break, less consistently.
		a.log.Warn("the bundled serif fonts did not load", "err", err)
	}
	a.loadSettings()
	a.applyLocale()
	a.applyTheme()
	a.ideas = ideas.NewDeck(a.appLocale())
	a.previewIdea = a.ideas.Next(planner.Mini)
	a.pl = planner.New(configFromSettings(a.store.Get()))
	a.drv = scheduler.New(a.onTick)
	a.initScheduleState()
	a.notifier = notify.New(a.log)
	a.sounds = sounds.New(a.log)
	a.buildTray()
	a.buildBreakWindows()
	a.buildSettingsWindow()
	a.buildWelcomeWindow()
	a.startPresence()
	a.startFullscreenWatch()
	a.watchAppEvents()
	a.applyLogin()
	if a.corrupt && a.prefs != nil {
		a.prefs.SetBanner(locales.BannerConfigCorrupt(), "", nil)
	}
	a.scheduleUpdateCheck()
	a.scheduleTrayRefresh()
	a.log.Info("Fermata started",
		"version", a.opts.Version,
		"locale", a.appLocale(),
		"config", a.store.Path(),
		"firstRun", a.store.Get().FirstRun,
	)

	switch {
	case a.store.Get().FirstRun:
		if name := a.opts.Initial.Name; name != "" && name != cli.CmdStart {
			a.pending = append(a.pending, a.opts.Initial)
		}
		a.showWelcome()
	case a.opts.Initial.Name != "" && a.opts.Initial.Name != cli.CmdStart:
		if !a.restoreSchedule() {
			a.beginSchedule()
		}
		a.HandleCommand(a.opts.Initial)
	default:
		if !a.restoreSchedule() {
			a.beginSchedule()
		}
	}
}

// startLogging writes the log beside the app's other files.
func (a *App) startLogging() {
	dir, err := paths.Logs()
	if err != nil {
		a.log = slog.Default()
		return
	}
	debug := false
	if v := os.Getenv("FERMATA_DEBUG"); v != "" && v != "0" {
		debug = true
	}
	a.log, a.logs = logging.Setup(dir, debug)
	slog.SetDefault(a.log)
}

// loadSettings reads config.json, or writes the defaults.
func (a *App) loadSettings() {
	path, err := paths.Config()
	if err != nil {
		a.log.Error("config path", "error", err)
		a.store = settings.New("config.json", settings.Default())
		return
	}
	store, warnings, err := settings.Load(path)
	if err != nil {
		a.log.Error("read settings; using the defaults", "error", err)
		a.store = settings.New(path, settings.Default())
		return
	}
	a.store = store
	for _, warning := range warnings {
		a.log.Warn("settings", "warning", warning)
		if contains(warning, "unreadable") || contains(warning, "empty") || contains(warning, "no config") {
			a.corrupt = a.corrupt || contains(warning, "unreadable")
		}
	}
	if err := store.LastWriteError(); err != nil {
		a.log.Warn("settings are not saved", "error", err)
	}
	store.OnChanged(a.onConfigChanged)
	store.OnWriteError(func(err error) {
		a.log.Warn("save settings", "error", err)
		mygo.RunOnMain(func() {
			if a.prefs != nil {
				a.prefs.SetBanner(locales.BannerWriteFailed(), "", nil)
			}
		})
	})
}

// buildTray creates the menu bar icon and its menu.
func (a *App) buildTray() {
	c, err := tray.New(trayicon.Mark(36), tray.Actions{
		SkipTo:        a.skipTo,
		Pause:         a.pauseFor,
		PauseMorning:  a.pauseUntilMorning,
		Resume:        a.resume,
		Reset:         a.reset,
		Settings:      a.openSettings,
		Quit:          a.quit,
		TogglePopover: a.togglePopover,
		HidePopover:   a.hidePopover,
	}, trayLabels())
	if err != nil {
		a.log.Error("create tray", "error", err)
		return
	}
	a.tray = c
	a.refreshTray()
}

// buildBreakWindows prepares the break windows' manager.
func (a *App) buildBreakWindows() {
	a.breaks = breakwin.New(breakwin.Options{
		Settings:    func() settings.BreakWindow { return a.store.Get().BreakWindow },
		Idea:        func(kind planner.Kind) ideas.Idea { return a.ideas.Next(kind) },
		Clock:       a.clock,
		Action:      a.onBreakAction,
		Label:       breakLabel,
		ActionLabel: actionLabel,
		Dark:        func() bool { return mygo.Theme.IsDark() },
		Background:  a.backgroundImage,
		Log:         a.log,
	})
}

// buildSettingsWindow prepares the settings window.
func (a *App) buildSettingsWindow() {
	a.prefs = prefs.New(prefs.Options{
		Settings:       func() *settings.Settings { return a.store.Get() },
		Touched:        func() { a.store.Touched() },
		Reset:          a.resetSettings,
		PreviewSound:   func(name string) { a.playNamed(name) },
		Dark:           func() bool { return mygo.Theme.IsDark() },
		Clock:          a.clock,
		Displays:       func() []mygo.Display { return mygo.Screen.Displays() },
		FocusAvailable: func() bool { return a.focusReadable },
		Version:        func() (string, string) { return a.opts.Version, a.opts.Build },
		Update:         func() (string, string) { return a.updateVersion, a.updateURL },
		Website:        websiteURL,
		Source:         sourceURL,
		Issues:         issuesURL,
		PreviewText:    func() string { return a.previewIdea.Text },
		Log:            a.log,
	})
}

// buildWelcomeWindow prepares the first-run window.
func (a *App) buildWelcomeWindow() {
	a.welcome = welcome.New(welcome.Options{
		Settings: func() *settings.Settings { return a.store.Get() },
		Touched:  func() { a.store.Touched() },
		Start:    a.finishWelcome,
		Dark:     func() bool { return mygo.Theme.IsDark() },
		Clock:    a.clock,
		Idea:     func() ideas.Idea { return a.previewIdea },
		Log:      a.log,
	})
}

// startPresence watches the system for what pauses breaks.
func (a *App) startPresence() {
	a.focusReadable = presence.FocusReadable()
	a.pres = presence.New(presence.SystemSource(), a.onPresence)
	a.configurePresence()
	a.pres.Start()
}

// watchAppEvents registers the app's own callbacks.
func (a *App) watchAppEvents() {
	// Fermata lives in the menu bar: closing the welcome or settings window
	// is not quitting. Without this listener MyGo quits when the last window
	// closes, which ended the app as soon as the welcome window was done.
	mygo.App.OnWindowAllClosed(func() {})
	mygo.App.OnBeforeQuit(a.onBeforeQuit)
	mygo.App.OnNotificationClick(func(id string) {
		if strings.HasPrefix(id, "break-") {
			mygo.RunOnMain(a.togglePopover)
		}
	})
	mygo.Screen.OnDisplaysChanged(a.onDisplaysChanged)
	mygo.App.OnLocaleChanged(func(string) {
		if a.tray != nil {
			a.tray.Relabel(trayLabels())
		}
		a.syncWindows()
		a.refreshTray()
	})
}

// beginSchedule starts the rhythm, unless the app is paused.
func (a *App) beginSchedule() {
	a.step(planner.Reset{Now: time.Now()})
}

// showWelcome opens the first-run window.
func (a *App) showWelcome() {
	if a.welcome == nil {
		return
	}
	a.welcomeOpen = true
	a.welcome.Show()
	if a.corrupt && a.prefs != nil {
		a.prefs.SetBanner(locales.BannerConfigCorrupt(), "", nil)
	}
}

// finishWelcome records the first decision and starts the schedule.
func (a *App) finishWelcome(openSettings bool) {
	if a.welcome != nil {
		a.welcome.Close()
	}
	a.welcomeOpen = false
	a.store.Update(func(s *settings.Settings) { s.FirstRun = false })
	if err := a.store.Flush(); err != nil {
		a.log.Warn("save first run", "error", err)
	}
	a.beginSchedule()
	pending := a.pending
	a.pending = nil
	for _, cmd := range pending {
		a.HandleCommand(cmd)
	}
	if openSettings {
		a.openSettings()
	}
}

// applyLocale follows the language setting.
func (a *App) applyLocale() {
	tag := a.store.Get().General.Language
	if tag == a.langTag {
		return
	}
	if tag == settings.LanguageSystem {
		mygo.App.SetLocale("")
	} else {
		mygo.App.SetLocale(tag)
	}
	if a.ideas != nil {
		if err := a.ideas.SetLanguage(a.appLocale()); err != nil {
			a.log.Warn("break ideas", "error", err)
		}
	}
	a.langTag = tag
	if a.prefs != nil {
		a.prefs.Refresh()
	}
	if a.welcome != nil {
		a.welcome.Refresh()
	}
	if a.tray != nil {
		a.tray.Relabel(trayLabels())
	}
	a.refreshTray()
}

// appLocale is the language the app shows, as the deck's file names it.
func (a *App) appLocale() string {
	locale := mygo.App.Locale()
	if locale == settings.LanguageChinese || strings.HasPrefix(locale, "zh") {
		return "zh-Hans"
	}
	return "en"
}

// configurePresence passes the settings' presence choices to the manager.
func (a *App) configurePresence() {
	if a.pres == nil {
		return
	}
	a.focusReadable = presence.FocusReadable()
	s := a.store.Get()
	var rule *presence.Rule
	for _, r := range s.Presence.AppExclusions {
		if !r.Active {
			continue
		}
		rule = &presence.Rule{Name: r.Name, Processes: r.Commands, Pause: r.Rule != settings.RuleResume}
		break
	}
	a.pres.Configure(presence.Config{
		IdleEnabled:  s.Presence.Idle.Enabled,
		IdleReset:    time.Duration(s.Presence.Idle.ResetAfterMs) * time.Millisecond,
		DNDEnabled:   s.Presence.DND.Enabled && a.focusReadable,
		Rule:         rule,
		PollInterval: time.Duration(s.Presence.Idle.CheckIntervalMs) * time.Millisecond,
	})
}

// applyLogin follows the login item setting, once per change.
func (a *App) applyLogin() {
	open := a.store.Get().General.OpenAtLogin
	if open == a.loginOn {
		return
	}
	if err := mygo.App.SetOpenAtLogin(open); err != nil {
		a.log.Warn("start at login", "open", open, "error", err)
	}
	a.loginOn = open
}

// openSettings brings the settings window forward.
func (a *App) openSettings() {
	if a.prefs == nil {
		return
	}
	a.prefs.Show()
}

// quit asks the app to end; a strict break refuses first.
func (a *App) quit() { mygo.App.Quit() }

// onBeforeQuit stops a quit during a strict break.
func (a *App) onBeforeQuit(e *mygo.QuitEvent) {
	if e.SessionEnding {
		a.shutdown()
		return
	}
	st, _ := a.pl.Snapshot()
	if st.Strict && (st.Phase == planner.Breaking || st.Phase == planner.ManualAwait) {
		e.PreventDefault()
		a.log.Info("quit refused: a strict break is running")
		a.notifyStrict()
		return
	}
	a.shutdown()
}

// notifyStrict tells the user why the app stays, without a dialog.
func (a *App) notifyStrict() {
	left := "--:--"
	if a.breaks != nil && a.breaks.Visible() {
		spec := a.breaks.Spec()
		left = statusDuration(spec.Remaining(time.Now()))
	}
	st, _ := a.pl.Snapshot()
	body := locales.MenuBreakOver()
	if st.Phase == planner.Breaking {
		if st.Kind == planner.Long {
			body = locales.MenuStrictLong(left)
		} else {
			body = locales.MenuStrictMini(left)
		}
	}
	a.notifier.Show("strict", locales.AppName(), body, a.store.Get().Sound.Muted)
}

// shutdown stops everything, in an order that loses nothing.
func (a *App) shutdown() {
	if a.quitting {
		return
	}
	a.quitting = true
	a.log.Info("Fermata quitting")
	a.persistSchedule()
	if a.trayTimer != nil {
		a.trayTimer.Stop()
	}
	if a.updateTimer != nil {
		a.updateTimer.Stop()
	}
	if a.drv != nil {
		a.drv.Close()
	}
	if a.shortcutOn {
		mygo.GlobalShortcut.UnregisterAll()
		a.shortcutOn = false
	}
	if a.pres != nil {
		a.pres.Stop()
	}
	if a.notifier != nil {
		a.notifier.Close()
	}
	if a.breaks != nil {
		a.breaks.Close()
	}
	if a.welcome != nil {
		a.welcome.Close()
	}
	if a.prefs != nil {
		a.prefs.Close()
	}
	if a.store != nil {
		_ = a.store.Close()
	}
	if a.logs != nil {
		_ = a.logs.Close()
	}
}

// onDisplaysChanged rebuilds the break windows without restarting the clock.
func (a *App) onDisplaysChanged() {
	if a.breaks != nil && a.breaks.Visible() {
		a.breaks.Rebuild()
	}
	a.refreshTray()
}

// onPresence maps the system's events to pause reasons.
func (a *App) onPresence(ev presence.Event) {
	if a.quitting || a.pl == nil {
		return
	}
	a.log.Info("presence", "event", ev.Kind.String(), "detail", ev.Detail)
	switch ev.Kind {
	case presence.IdleEnter:
		a.step(planner.Pause{Now: ev.At, Reason: planner.Idle})
	case presence.IdleLeave:
		a.step(planner.Resume{Now: ev.At, Reason: planner.Idle})
		a.step(planner.Tick{Now: ev.At})
	case presence.DNDEnter:
		a.step(planner.Pause{Now: ev.At, Reason: planner.DND})
	case presence.DNDLeave:
		a.step(planner.Resume{Now: ev.At, Reason: planner.DND})
		a.step(planner.Tick{Now: ev.At})
	case presence.Lock, presence.Suspend:
		if a.store.Get().Presence.PauseOnLock {
			a.step(planner.Pause{Now: ev.At, Reason: planner.Lock})
		}
	case presence.Unlock:
		if a.store.Get().Presence.PauseOnLock {
			a.step(planner.Resume{Now: ev.At, Reason: planner.Lock})
		}
		a.step(planner.Tick{Now: ev.At})
	case presence.Resume:
		if a.store.Get().Presence.PauseOnLock {
			a.step(planner.Resume{Now: ev.At, Reason: planner.Lock})
		}
		a.step(planner.Wake{Now: ev.At})
	case presence.ExclusionEnter:
		a.step(planner.Pause{Now: ev.At, Reason: planner.Exclusion})
	case presence.ExclusionLeave:
		a.step(planner.Resume{Now: ev.At, Reason: planner.Exclusion})
		a.step(planner.Tick{Now: ev.At})
	case presence.Unavailable:
		// Nothing to ask the user for: an unreadable probe simply stops.
		a.log.Info("presence is unavailable", "detail", ev.Detail)
	}
	a.refreshTray()
}

// resetSettings restores the defaults and starts the rhythm again.
func (a *App) resetSettings() {
	a.store.Update(func(s *settings.Settings) {
		firstRun := s.FirstRun
		*s = settings.Default()
		s.FirstRun = firstRun
	})
	if err := a.store.Flush(); err != nil {
		a.log.Warn("save restored defaults", "error", err)
	}
	a.danger = 0
	a.applyLocale()
	a.applyLogin()
	a.configurePresence()
	a.step(planner.Reset{Now: time.Now()})
	a.refreshTray()
	if a.prefs != nil {
		a.prefs.Refresh()
	}
}
