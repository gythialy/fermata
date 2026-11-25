package app

import (
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/egoist/mygo"
	"github.com/gythialy/fermata/internal/breakwin"
	"github.com/gythialy/fermata/internal/paths"
	"github.com/gythialy/fermata/internal/planner"
	"github.com/gythialy/fermata/internal/settings"
	"github.com/gythialy/fermata/internal/status"
	"github.com/gythialy/fermata/internal/tray"
	"github.com/gythialy/fermata/internal/trayicon"
	"github.com/gythialy/fermata/locales"
)

// onTick wakes the planner at its deadline, after a held break has had its say.
func (a *App) onTick(now time.Time) {
	if a.pl == nil || a.quitting {
		return
	}
	if a.holdReason != "" {
		a.holdTick(now)
		return
	}
	if reason := a.holdNeeded(now); reason != "" {
		a.beginHold(reason, now)
		return
	}
	a.step(planner.Tick{Now: now})
}

// step applies one input and everything it caused.
func (a *App) step(in planner.Input) {
	if a.pl == nil || a.quitting {
		return
	}
	a.apply(a.pl.Step(in))
	if a.holdReason != "" {
		st, _ := a.pl.Snapshot()
		if st.Phase != planner.Waiting && st.Phase != planner.Notified {
			a.holdReason = ""
		} else {
			switch in.(type) {
			case planner.Tick, planner.SetConfig, planner.NextText:
				// The hold re-arms itself; these inputs do not move the break.
			default:
				a.holdReason = ""
			}
		}
	}
	st, _ := a.pl.Snapshot()
	if st.Phase != planner.Waiting && st.Phase != planner.Notified {
	}
	a.syncWindows()
	if a.popover != nil {
		a.popover.Invalidate()
	}
	a.refreshTray()
	a.ensureTrayRate()
	a.persistSchedule()
}

// apply performs the planner's effects in order.
func (a *App) apply(effects []planner.Effect) {
	for _, effect := range effects {
		switch e := effect.(type) {
		case planner.Arm:
			a.drv.Arm(e.At)
		case planner.Disarm:
			a.drv.Stop()
		case planner.CloseNotice:
			a.notifier.Close()
		case planner.Notify:
			a.notice(e)
		case planner.StartBreak:
			a.hidePopover()
			a.startBreak(e.Spec)
		case planner.CloseBreak:
			a.closeBreak(e.Reason)
		case planner.PlaySound:
			a.playCue(e.Name)
		case planner.NotifyResume:
			a.notifier.Show("resume", locales.AppName(), locales.NotifyResumed(), a.store.Get().Sound.Muted)
		case planner.TrayDirty:
			// The tray is refreshed after the effects, like the windows.
		}
	}
}

// notice shows the notice before a break.
func (a *App) notice(e planner.Notify) {
	seconds := int(math.Round(e.Lead.Seconds()))
	silent := a.store.Get().Sound.Muted
	if e.Kind == planner.Long {
		a.notifier.Show("break-long", locales.AppName(), locales.NotifyLong(seconds), silent)
		return
	}
	a.notifier.Show("break-mini", locales.AppName(), locales.NotifyMini(seconds), silent)
}

// startBreak opens the windows and registers the one shortcut.
func (a *App) startBreak(spec planner.BreakSpec) {
	a.log.Info("break started",
		"kind", spec.Kind.String(),
		"duration", spec.Duration.String(),
		"strict", spec.Strict,
		"manualFinish", spec.ManualFinish,
	)
	a.lastSpec = spec
	if a.breaks != nil {
		a.breaks.Show(spec, planner.Breaking)
		a.lastIdea = a.breaks.Idea()
	}
	a.registerShortcut()
}

// closeBreak records what the break cost and takes the windows down.
func (a *App) closeBreak(reason planner.CloseReason) {
	a.log.Info("break closed", "reason", reason.String(), "health", a.danger)
	kind := a.lastSpec.Kind
	if kind == 0 {
		kind = planner.Mini
	}
	switch reason {
	case planner.Completed:
		a.adjustHealth(-healthCost(kind))
	case planner.Postponed:
		a.adjustHealth(1)
	case planner.Skipped, planner.Interrupted:
		a.adjustHealth(healthCost(kind))
	}
	if a.breaks != nil {
		a.breaks.Close()
	}
	a.unregisterShortcut()
}

// healthCost is what a skipped or finished break counts in the health counter.
func healthCost(kind planner.Kind) int {
	if kind == planner.Long {
		return 2
	}
	return 1
}

// adjustHealth moves the soft break health counter.
func (a *App) adjustHealth(delta int) {
	if !a.store.Get().Health.Mode {
		a.danger = 0
		return
	}
	a.danger = min(max(a.danger+delta, 0), 10)
}

// playCue plays the start or end sound of the last break.
func (a *App) playCue(cue string) {
	s := a.store.Get()
	sound := s.Sound.MiniEnd
	if cue == planner.SoundStart {
		sound = s.Sound.MiniStart
		if a.lastSpec.Kind == planner.Long {
			sound = s.Sound.LongStart
		}
	} else if a.lastSpec.Kind == planner.Long {
		sound = s.Sound.LongEnd
	}
	a.sounds.Play(sound, s.Sound.Volume, s.Sound.Muted)
}

// playNamed plays a sound from the settings, for its preview button.
func (a *App) playNamed(name string) {
	a.sounds.Play(name, a.store.Get().Sound.Volume, a.store.Get().Sound.Muted)
}

// registerShortcut binds the break's shortcut while a break is on screen.
func (a *App) registerShortcut() {
	if a.shortcutOn {
		return
	}
	acc := a.store.Get().Shortcuts.EndBreak
	if acc == "" {
		return
	}
	if err := mygo.GlobalShortcut.Register(acc, a.onBreakShortcut); err != nil {
		a.log.Warn("break shortcut is taken", "accelerator", acc, "error", err)
		return
	}
	a.shortcutAcc, a.shortcutOn = acc, true
}

// unregisterShortcut frees the shortcut again.
func (a *App) unregisterShortcut() {
	if !a.shortcutOn {
		return
	}
	mygo.GlobalShortcut.Unregister(a.shortcutAcc)
	a.shortcutOn = false
}

// onBreakShortcut is the shortcut the break window names.
func (a *App) onBreakShortcut() {
	if a.breaks == nil {
		return
	}
	action, ok := a.breaks.ActionAt(time.Now())
	if !ok {
		return
	}
	a.onBreakAction(action)
}

// onBreakAction runs the break's one exit.
func (a *App) onBreakAction(action breakwin.Action) {
	now := time.Now()
	switch action {
	case breakwin.Postpone:
		a.step(planner.Postpone{Now: now})
	case breakwin.Skip:
		a.step(planner.Skip{Now: now})
	case breakwin.Finish:
		a.step(planner.Finish{Now: now})
	}
}

// syncWindows tells the break windows what the planner now says.
func (a *App) syncWindows() {
	if a.breaks == nil || !a.breaks.Visible() || a.pl == nil {
		return
	}
	st, _ := a.pl.Snapshot()
	a.breaks.Sync(st.Phase, st.Postpones, a.danger)
}

// refreshTray computes the tray's words, icon and menu.
func (a *App) refreshTray() {
	if a.holdReason != "" {
		if st, _ := a.pl.Snapshot(); st.Phase != planner.Waiting && st.Phase != planner.Notified {
			a.holdReason = ""
		}
	}
	if a.tray == nil || a.pl == nil {
		return
	}
	now := time.Now()
	st, spec := a.pl.Snapshot()
	cfg := a.pl.Config()
	s := a.store.Get()
	lines := a.trayLines(now)
	style := s.General.MenuBarStyle
	icon, iconKey := a.trayIcon(style, st, spec, cfg, now)
	a.tray.Update(tray.Snapshot{
		State:          lines,
		Style:          style,
		Title:          trayTitle(st, now),
		Icon:           icon,
		IconKey:        iconKey,
		Phase:          st.Phase,
		Strict:         st.Strict,
		CanPause:       a.pl.CanPause(),
		CanResume:      a.pl.CanResume(),
		CanReset:       a.pl.CanReset(),
		CanSkipTo:      a.pl.CanSkipTo(),
		MiniEnabled:    cfg.Mini.Enabled,
		LongEnabled:    cfg.Long.Enabled,
		KeepInStrict:   s.General.ShowTrayMenuInStrictMode,
		PauseDurations: a.pauseDurations(),
	})
}

// trayLines is the status text, with a held break's own wording.
func (a *App) trayLines(now time.Time) status.Lines {
	st, _ := a.pl.Snapshot()
	cfg := a.pl.Config()
	lines := status.Format(st, now, status.Extra{LongAfterMinis: longAfterMinis(cfg, st)}, messages{})
	if a.holdReason == holdFullscreen {
		lines.Status = locales.MenuHeldFullscreen()
		lines.Detail = ""
		lines.Tooltip = locales.TrayTooltip() + "\n\n" + lines.Status
	}
	return lines
}

// trayTitle is the menu bar text of the time style.
func trayTitle(st planner.State, now time.Time) string {
	switch st.Phase {
	case planner.Waiting:
		minutes := int(math.Ceil(st.BreakAt.Sub(now).Minutes()))
		return fmt.Sprintf("%dm", min(max(minutes, 0), 99))
	case planner.Notified, planner.Breaking, planner.ManualAwait:
		return "0m"
	default:
		return ""
	}
}

// trayIcon chooses the icon and a key that avoids setting it twice.
func (a *App) trayIcon(style string, st planner.State, spec planner.BreakSpec, cfg planner.Config, now time.Time) ([]byte, string) {
	const size = 36
	switch {
	case st.Phase == planner.Paused:
		return trayicon.Paused(size), "paused"
	case style == settings.StyleProgress:
		p := int(math.Round(progressShare(st, spec, cfg, now) * 100))
		return trayicon.Ring(size, float64(p)/100), fmt.Sprintf("ring-%d", p)
	default:
		return trayicon.Mark(size), "mark"
	}
}

// progressShare is how much of the current wait or break has passed.
func progressShare(st planner.State, spec planner.BreakSpec, cfg planner.Config, now time.Time) float64 {
	switch st.Phase {
	case planner.Breaking:
		return spec.Progress(now)
	case planner.ManualAwait:
		return 1
	case planner.Waiting, planner.Notified:
		gap := cfg.Gap()
		if gap <= 0 {
			return 0
		}
		left := st.BreakAt.Sub(now)
		return min(max(1-float64(left)/float64(gap), 0), 1)
	default:
		return 0
	}
}

// longAfterMinis is the second status line's count.
func longAfterMinis(cfg planner.Config, st planner.State) int {
	if !cfg.Mini.Enabled || !cfg.Long.Enabled || st.N < 1 {
		return 0
	}
	if st.Phase != planner.Waiting && st.Phase != planner.Notified {
		return 0
	}
	if st.Kind != planner.Mini {
		return 0
	}
	n := cfg.EveryN() + 1
	remain := n - st.N%n
	if remain <= 0 || remain > cfg.EveryN() {
		return 0
	}
	return remain
}

// ensureTrayRate changes how often the tray's countdown is refreshed.
func (a *App) ensureTrayRate() {
	rate := time.Minute
	if st, _ := a.pl.Snapshot(); st.Phase == planner.Breaking || st.Phase == planner.ManualAwait {
		rate = time.Second
	}
	if rate == a.trayRefresh && a.trayTimer != nil {
		return
	}
	a.scheduleTrayRefresh()
}

// scheduleTrayRefresh runs the tray's own clock: once a second during a break,
// once a minute otherwise.
func (a *App) scheduleTrayRefresh() {
	if a.trayTimer != nil {
		a.trayTimer.Stop()
	}
	rate := time.Minute
	if a.pl != nil {
		if st, _ := a.pl.Snapshot(); st.Phase == planner.Breaking || st.Phase == planner.ManualAwait {
			rate = time.Second
		}
	}
	a.trayRefresh = rate
	a.trayTimer = time.AfterFunc(rate, func() {
		mygo.RunOnMain(func() {
			if a.quitting {
				return
			}
			if st, _ := a.pl.Snapshot(); st.Phase == planner.Waiting || st.Phase == planner.Notified {
				a.persistSchedule()
			}
			a.refreshTray()
			a.scheduleTrayRefresh()
		})
	})
}

// backgroundImage picks a background for the break, if the setting asks for
// one: a random picture from the user's backgrounds folder, never the same
// twice in a row.
func (a *App) backgroundImage() []byte {
	s := a.store.Get()
	if !s.BreakWindow.BackgroundImage {
		return nil
	}
	if len(a.bgPool) == 0 {
		dir, err := paths.Backgrounds()
		if err != nil {
			return nil
		}
		for _, pattern := range []string{"*.png", "*.jpg", "*.jpeg", "*.webp", "*.gif"} {
			matches, _ := filepath.Glob(filepath.Join(dir, pattern))
			a.bgPool = append(a.bgPool, matches...)
		}
		sort.Strings(a.bgPool)
	}
	if len(a.bgPool) == 0 {
		a.log.Info("no background images found", "folder", "backgrounds")
		return nil
	}
	path := a.bgPool[rand.IntN(len(a.bgPool))]
	for range 4 {
		if path != a.lastBg {
			break
		}
		path = a.bgPool[rand.IntN(len(a.bgPool))]
	}
	data, err := os.ReadFile(path)
	if err != nil {
		a.log.Warn("background image", "path", path, "error", err)
		return nil
	}
	a.lastBg = path
	return data
}
