package app

import (
	"fmt"
	"time"

	"github.com/gythialy/fermata/internal/cli"
	"github.com/gythialy/fermata/internal/planner"
)

// HandleCommand runs one command line inside the running app. It returns the
// process's exit code: 0 for success, 1 for a command the state refuses.
func (a *App) HandleCommand(cmd cli.Command) int {
	if a.store == nil || a.pl == nil {
		return 1
	}
	if a.welcomeOpen {
		// The welcome window is one decision; the command waits for it.
		a.pending = append(a.pending, cmd)
		return 0
	}
	a.log.Info("command", "name", cmd.Name)
	switch cmd.Name {
	case cli.CmdStart:
		return 0
	case cli.CmdPause:
		return a.commandPause(cmd)
	case cli.CmdResume:
		a.resume()
		return 0
	case cli.CmdToggle:
		if a.strictRunning() {
			return a.refuse("toggle")
		}
		if a.pl.CanResume() {
			a.resume()
		} else {
			a.pauseFor(0, true)
		}
		return 0
	case cli.CmdReset:
		if a.strictRunning() {
			return a.refuse("reset")
		}
		a.reset()
		return 0
	case cli.CmdSkip:
		if a.strictRunning() {
			return a.refuse("skip")
		}
		if !a.pl.CanSkip(time.Now()) {
			return a.refuse("skip")
		}
		a.step(planner.Skip{Now: time.Now()})
		return 0
	case cli.CmdPostpone:
		if a.strictRunning() {
			return a.refuse("postpone")
		}
		if !a.pl.CanPostpone(time.Now()) {
			return a.refuse("postpone")
		}
		a.step(planner.Postpone{Now: time.Now()})
		return 0
	case cli.CmdFinish:
		if !a.pl.CanFinish() {
			return a.refuse("finish")
		}
		a.step(planner.Finish{Now: time.Now()})
		return 0
	case cli.CmdMini:
		return a.commandBreak(cmd, planner.Mini)
	case cli.CmdLong:
		return a.commandBreak(cmd, planner.Long)
	case cli.CmdLast:
		return a.commandLast()
	case cli.CmdPopover:
		if !cli.Debug {
			return a.refuse("popover")
		}
		a.commandPopover(cmd)
		return 0
	case cli.CmdPreferences:
		if a.strictRunning() {
			return a.refuse("preferences")
		}
		a.openSettings()
		return 0
	case cli.CmdHelp:
		fmt.Print(cli.Help())
		return 0
	case cli.CmdVersion:
		fmt.Println(cli.Version(a.opts.Version, a.opts.Commit))
		return 0
	default:
		a.log.Warn("unknown command", "name", cmd.Name)
		return 1
	}
}

// commandPause pauses, unless a strict break holds the app.
func (a *App) commandPause(cmd cli.Command) int {
	if a.strictRunning() {
		return a.refuse("pause")
	}
	now := time.Now()
	var until time.Time
	switch {
	case cmd.UntilMorning:
		until = cli.UntilMorning(now, a.store.Get().Schedule.MorningHour)
	case !cmd.Indefinite && cmd.Duration > 0:
		until = now.Add(cmd.Duration)
	}
	a.step(planner.Pause{Now: now, Reason: planner.Manual, Until: until})
	return 0
}

// commandBreak starts a break the command line asked for.
func (a *App) commandBreak(cmd cli.Command, kind planner.Kind) int {
	cfg := a.pl.Config()
	if !cfg.For(kind).Enabled {
		a.log.Warn("that break kind is off", "kind", kind.String())
		return 1
	}
	if a.strictRunning() {
		return a.refuse(cmd.Name)
	}
	now := time.Now()
	if cmd.NoSkip {
		a.step(planner.NextText{Now: now, Title: cmd.Title, Text: cmd.Text})
		return 0
	}
	a.step(planner.SkipTo{Now: now, Kind: kind, Delay: cmd.Wait, Title: cmd.Title, Text: cmd.Text})
	return 0
}

// commandPopover opens or closes the popover as a test asks, which cannot
// click the tray icon over the automation protocol.
func (a *App) commandPopover(cmd cli.Command) {
	switch cmd.Popover {
	case "show":
		a.showPopover()
	case "hide":
		a.hidePopover()
	default:
		a.togglePopover()
	}
}

// commandLast starts the last break's idea again.
func (a *App) commandLast() int {
	st, _ := a.pl.Snapshot()
	if st.Phase == planner.Breaking || st.Phase == planner.ManualAwait || st.Phase == planner.Paused {
		return a.refuse("last")
	}
	if a.lastSpec.Kind == 0 || a.lastIdea.Text == "" {
		return a.refuse("last")
	}
	cfg := a.pl.Config()
	if !cfg.For(a.lastSpec.Kind).Enabled {
		return a.refuse("last")
	}
	a.step(planner.SkipTo{
		Now:   time.Now(),
		Kind:  a.lastSpec.Kind,
		Title: a.lastIdea.Title,
		Text:  a.lastIdea.Text,
	})
	return 0
}

// strictRunning reports whether a strict break is counting down.
func (a *App) strictRunning() bool {
	st, _ := a.pl.Snapshot()
	if !st.Strict {
		return false
	}
	return st.Phase == planner.Breaking || st.Phase == planner.ManualAwait
}

// refuse logs a command the state does not allow.
func (a *App) refuse(name string) int {
	a.log.Warn("command refused", "name", name, "reason", "current state")
	return 1
}
