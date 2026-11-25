// Package cli parses Fermata's command line. It is a pure function: the app
// decides what a command does, and which commands a strict break refuses.
package cli

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// ErrUsage marks a command line that cannot be parsed: the process exits with
// code 2.
var ErrUsage = errors.New("usage")

// The commands the app understands.
const (
	CmdStart       = "start"
	CmdHelp        = "help"
	CmdVersion     = "version"
	CmdPause       = "pause"
	CmdResume      = "resume"
	CmdToggle      = "toggle"
	CmdReset       = "reset"
	CmdSkip        = "skip"
	CmdPostpone    = "postpone"
	CmdFinish      = "finish"
	CmdMini        = "mini"
	CmdLong        = "long"
	CmdLast        = "last"
	CmdPreferences = "preferences"
)

// Command is one parsed command line.
type Command struct {
	Name string
	// Duration is the pause's length, for Pause.
	Duration time.Duration
	// Indefinite pauses without an end.
	Indefinite bool
	// UntilMorning pauses until the next morning hour.
	UntilMorning bool
	// Title and Text override the next break's idea.
	Title, Text string
	// NoSkip sets the next break's text without starting the break.
	NoSkip bool
	// Wait delays starting the break.
	Wait time.Duration
}

// Parse turns the arguments after the program name into a command.
func Parse(args []string) (Command, error) {
	if len(args) == 0 {
		return Command{Name: CmdStart}, nil
	}
	name := args[0]
	switch name {
	case "help", "-h", "--help":
		return Command{Name: CmdHelp}, nil
	case "version", "-v", "--version":
		return Command{Name: CmdVersion}, nil
	}
	cmd := Command{Name: name}
	rest := args[1:]
	switch name {
	case CmdStart:
		return cmd, nil
	case CmdPause:
		return parsePause(rest)
	case CmdResume, CmdToggle, CmdReset, CmdSkip, CmdPostpone, CmdFinish, CmdLast, CmdPreferences:
		if err := noArgs(name, rest); err != nil {
			return Command{}, err
		}
		return cmd, nil
	case CmdMini, CmdLong:
		return parseBreak(cmd, rest)
	default:
		return Command{}, usagef("unknown command %q", name)
	}
}

func parsePause(args []string) (Command, error) {
	cmd := Command{Name: CmdPause, Indefinite: true}
	positional := false
	set := func(value string) error {
		switch value {
		case "indefinitely":
			cmd.Indefinite = true
		case "until-morning":
			cmd.UntilMorning = true
			cmd.Indefinite = false
		default:
			d, err := Duration(value)
			if err != nil {
				return err
			}
			cmd.Duration = d
			cmd.Indefinite = false
		}
		return nil
	}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		var value string
		switch {
		case arg == "-d" || arg == "--duration":
			if i+1 >= len(args) {
				return Command{}, usagef("%s needs a duration", arg)
			}
			i++
			value = args[i]
		case strings.HasPrefix(arg, "-d="):
			value = strings.TrimPrefix(arg, "-d=")
		case strings.HasPrefix(arg, "--duration="):
			value = strings.TrimPrefix(arg, "--duration=")
		case !strings.HasPrefix(arg, "-") && !positional:
			// "fermata pause 30", as the command line has always read.
			positional = true
			value = arg
		default:
			return Command{}, usagef("pause does not take %q", arg)
		}
		if err := set(value); err != nil {
			return Command{}, err
		}
	}
	return cmd, nil
}

func parseBreak(cmd Command, args []string) (Command, error) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		needValue := func() (string, error) {
			if i+1 >= len(args) {
				return "", usagef("%s needs a value", arg)
			}
			i++
			return args[i], nil
		}
		switch arg {
		case "-n", "--no-skip":
			cmd.NoSkip = true
		case "-T", "--title":
			v, err := needValue()
			if err != nil {
				return Command{}, err
			}
			cmd.Title = v
		case "-t", "--text":
			v, err := needValue()
			if err != nil {
				return Command{}, err
			}
			cmd.Text = v
		case "-w", "--wait":
			v, err := needValue()
			if err != nil {
				return Command{}, err
			}
			d, err := Duration(v)
			if err != nil {
				return Command{}, err
			}
			cmd.Wait = d
		default:
			return Command{}, usagef("%s does not take %q", cmd.Name, arg)
		}
	}
	if cmd.NoSkip && cmd.Wait > 0 {
		return Command{}, usagef("--no-skip and --wait cannot be combined")
	}
	if cmd.Name == CmdMini && cmd.Text != "" {
		return Command{}, usagef("a mini break has no text: only --title")
	}
	return cmd, nil
}

func noArgs(name string, args []string) error {
	if len(args) > 0 {
		return usagef("%s takes no arguments", name)
	}
	return nil
}

func usagef(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrUsage, fmt.Sprintf(format, args...))
}

// Version returns the line the version command prints.
func Version(version, commit string) string {
	if commit == "" {
		return "Fermata " + version
	}
	return fmt.Sprintf("Fermata %s (%s)", version, commit)
}

// Help returns the usage text.
func Help() string {
	return `Fermata — a held note in the workday.

Usage:
  fermata                          start the app, or send it a command
  fermata help                     print this help
  fermata version                  print the version
  fermata pause [duration]        pause breaks; the default is indefinite
  fermata resume                   resume breaks
  fermata toggle                   pause or resume
  fermata reset                    restart the rhythm
  fermata skip                     skip the running break
  fermata postpone                 postpone the running break
  fermata finish                   finish a break waiting for you
  fermata mini [-T title] [-w d]   start a mini break
  fermata long [-T title] [-t text] [-w d]
                                     start a long break
  fermata last                     show the last break's idea again
  fermata preferences              open the settings window

Durations:
  30          minutes
  90m         90 minutes
  1h20m       1 hour and 20 minutes
  indefinitely, until-morning   for pause only

Flags:
  -T, --title <text>   override the next break's title
  -t, --text <text>    override a long break's text
  -w, --wait <time>    wait before the break starts
  -n, --no-skip        set the next break's text without starting it

A running app receives the command; a strict break refuses the commands that
would end it.
`
}
