// Command e2e drives a running Fermata from outside, through the MyGo
// automation protocol. It is the tool a person reaches for while debugging
// what a platform shows; the tests in internal/e2e are the ones that run
// unattended.
//
// The app must be an e2e build (built with -tags e2e, keeping the inspector)
// started with MYGO_AUTOMATION naming a socket, e.g. tcp::9100:
//
//	GOFLAGS=-tags=e2e MYGO_INSPECTOR=1 go tool mygo build -platform windows/amd64 -skip-installer
//	MYGO_AUTOMATION=tcp::9100 ./Fermata.exe preferences
//
// Examples:
//
//	e2e windows
//	e2e tree -window 1
//	e2e assert -window 1 -text "General;Schedule;Appearance;About"
//	e2e press -window 1 -label "Appearance"
//	e2e capture -window 1 -out build/e2e/settings.png
//	e2e shotall -out build/e2e
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gythialy/fermata/internal/automation"
)

// settle is how long a read or an act waits for the window to stop moving.
const settle = 3 * time.Second

func main() {
	logf := func(format string, args ...any) { fmt.Fprintf(os.Stderr, "e2e: "+format+"\n", args...) }

	addr := flag.String("addr", envOr("FERMATA_E2E_ADDR", "127.0.0.1:9100"), "the app's automation address, host:port")
	timeout := flag.Duration("timeout", 10*time.Second, "how long to wait for an answer")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		usage()
		os.Exit(2)
	}
	if err := run(args[0], args[1:], *addr, *timeout, logf); err != nil {
		logf("%v", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `usage: e2e [-addr host:port] <command> [flags]

  windows                       list the app's windows
  tree    -window N [-json]     print the window's elements
  find    -window N -text T     print the elements whose text holds T
  assert  -window N -text LIST  check that every "; "-separated text is there
  press   -window N [-id I | -key K | -label L] [-action press]
  input   -window N [-key ACC] [-text T]   send a key or typed text to the window
  capture -window N -out FILE   write the window's image as PNG
  shotall -out DIR              write every window's image
  wait    -window N -text T     wait until the text appears
`)
}

func run(cmd string, args []string, addr string, timeout time.Duration, logf func(string, ...any)) error {
	switch cmd {
	case "windows":
		if err := flags("windows").Parse(args); err != nil {
			return err
		}
		c, err := automation.Dial(addr, timeout)
		if err != nil {
			return err
		}
		defer c.Close()
		windows, err := c.Windows()
		if err != nil {
			return err
		}
		for _, w := range windows {
			mark := " "
			if w.Focused {
				mark = "*"
			}
			hidden := ""
			if !w.Visible {
				hidden = " (hidden)"
			}
			fmt.Printf("%s%d\t%s\t%dx%d%s\n", mark, w.ID, w.Title, int(w.Bounds.Width), int(w.Bounds.Height), hidden)
		}
		if len(windows) == 0 {
			fmt.Println("no windows")
		}
		return nil

	case "tree", "find":
		fs := flags(cmd)
		window := fs.Int("window", 1, "the window")
		text := fs.String("text", "", "only the elements whose text holds this")
		asJSON := fs.Bool("json", false, "print the protocol's answer as it is")
		if err := fs.Parse(args); err != nil {
			return err
		}
		c, err := automation.Dial(addr, timeout)
		if err != nil {
			return err
		}
		defer c.Close()
		if err := c.Idle(*window, settle); err != nil {
			logf("window %d: %v", *window, err)
		}
		nodes, raw, err := c.Tree(*window)
		if err != nil {
			return err
		}
		if *asJSON {
			os.Stdout.Write(raw)
			fmt.Println()
			return nil
		}
		for _, n := range nodes {
			if *text != "" && !strings.Contains(strings.ToLower(n.Text()), strings.ToLower(*text)) {
				continue
			}
			fmt.Println(n.Line())
		}
		return nil

	case "assert":
		fs := flags(cmd)
		window := fs.Int("window", 1, "the window")
		text := fs.String("text", "", `the texts to find, separated by ";"`)
		if err := fs.Parse(args); err != nil {
			return err
		}
		c, err := automation.Dial(addr, timeout)
		if err != nil {
			return err
		}
		defer c.Close()
		nodes, _, err := c.Tree(*window)
		if err != nil {
			return err
		}
		var all strings.Builder
		for _, n := range nodes {
			all.WriteString(n.Text())
			all.WriteString("\n")
		}
		haystack := strings.ToLower(all.String())
		missing := 0
		for _, want := range strings.Split(*text, ";") {
			want = strings.TrimSpace(want)
			if want == "" {
				continue
			}
			if !strings.Contains(haystack, strings.ToLower(want)) {
				fmt.Printf("MISSING %q\n", want)
				missing++
				continue
			}
			fmt.Printf("ok      %q\n", want)
		}
		if missing > 0 {
			return fmt.Errorf("%d text(s) missing from window %d", missing, *window)
		}
		return nil

	case "press":
		fs := flags(cmd)
		window := fs.Int("window", 1, "the window")
		id := fs.String("id", "", "the element's ID, from an earlier tree")
		key := fs.String("key", "", "the element's key")
		label := fs.String("label", "", "the element's label, when it has no ID")
		action := fs.String("action", "press", "the action: press, focus, increment, ...")
		text := fs.String("text", "", "the text of setValue and insertText")
		dry := fs.Bool("dry", false, "only name the element the press would hit")
		if err := fs.Parse(args); err != nil {
			return err
		}
		if *id == "" && *key == "" && *label == "" {
			return errors.New("-id, -key or -label names the element")
		}
		c, err := automation.Dial(addr, timeout)
		if err != nil {
			return err
		}
		defer c.Close()

		var target automation.Node
		if *id != "" {
			nodes, _, err := c.Tree(*window)
			if err != nil {
				return err
			}
			for _, n := range nodes {
				if n.ID == *id {
					target = n
					break
				}
			}
			if target.ID == "" {
				return fmt.Errorf("no element with ID %s", *id)
			}
		} else {
			nodes, _, err := c.Tree(*window)
			if err != nil {
				return err
			}
			target, err = automation.Find(nodes, *key, *label, *action)
			if err != nil {
				return err
			}
		}
		if *dry {
			fmt.Printf("would %s %s\n", *action, target.Line())
			return nil
		}
		if err := c.Act(*window, target.ID, target.Key, *action, *text); err != nil {
			return err
		}
		if err := c.Idle(*window, settle); err != nil {
			logf("window %d: %v", *window, err)
		}
		fmt.Printf("%s %s\n", *action, target.Line())
		return nil

	case "input":
		fs := flags(cmd)
		window := fs.Int("window", 1, "the window")
		key := fs.String("key", "", "the key and its modifiers, as Ctrl+X or Enter")
		text := fs.String("text", "", "the text to type")
		if err := fs.Parse(args); err != nil {
			return err
		}
		if *key == "" && *text == "" {
			return errors.New("-key or -text names what to send")
		}
		var events []automation.Event
		if *key != "" {
			events = append(events, automation.Event{"type": "key", "key": *key})
		}
		if *text != "" {
			events = append(events, automation.Event{"type": "text", "text": *text})
		}
		c, err := automation.Dial(addr, timeout)
		if err != nil {
			return err
		}
		defer c.Close()
		if err := c.Input(*window, events); err != nil {
			return err
		}
		if err := c.Idle(*window, settle); err != nil {
			logf("window %d: %v", *window, err)
		}
		fmt.Printf("sent %d event(s) to window %d\n", len(events), *window)
		return nil

	case "capture":
		fs := flags(cmd)
		window := fs.Int("window", 1, "the window")
		out := fs.String("out", "", "the PNG to write")
		if err := fs.Parse(args); err != nil {
			return err
		}
		if *out == "" {
			return errors.New("-out names the file to write")
		}
		c, err := automation.Dial(addr, timeout)
		if err != nil {
			return err
		}
		defer c.Close()
		if err := c.Idle(*window, settle); err != nil {
			logf("window %d: %v", *window, err)
		}
		n, err := c.Capture(*window, *out)
		if err != nil {
			return err
		}
		fmt.Printf("wrote %s (%d bytes)\n", *out, n)
		return nil

	case "shotall":
		fs := flags(cmd)
		out := fs.String("out", "build/e2e", "the directory to write the PNGs to")
		if err := fs.Parse(args); err != nil {
			return err
		}
		c, err := automation.Dial(addr, timeout)
		if err != nil {
			return err
		}
		defer c.Close()
		windows, err := c.Windows()
		if err != nil {
			return err
		}
		if len(windows) == 0 {
			return errors.New("no windows to capture")
		}
		for _, w := range windows {
			path := filepath.Join(*out, fmt.Sprintf("%02d-%s.png", w.ID, slug(w.Title)))
			n, err := c.Capture(w.ID, path)
			if err != nil {
				logf("window %d: %v", w.ID, err)
				continue
			}
			fmt.Printf("%s\t%s\t%d bytes\n", w.Title, path, n)
		}
		return nil

	case "wait":
		fs := flags(cmd)
		window := fs.Int("window", 1, "the window")
		text := fs.String("text", "", "the text to wait for")
		within := fs.Duration("within", 15*time.Second, "how long to wait")
		if err := fs.Parse(args); err != nil {
			return err
		}
		c, err := automation.Dial(addr, timeout)
		if err != nil {
			return err
		}
		defer c.Close()
		deadline := time.Now().Add(*within)
		for {
			nodes, _, err := c.Tree(*window)
			if err != nil {
				return err
			}
			for _, n := range nodes {
				if strings.Contains(strings.ToLower(n.Text()), strings.ToLower(*text)) {
					fmt.Printf("found %q\n", *text)
					return nil
				}
			}
			if time.Now().After(deadline) {
				return fmt.Errorf("%q did not appear in window %d within %s", *text, *window, *within)
			}
			time.Sleep(200 * time.Millisecond)
		}

	default:
		usage()
		return fmt.Errorf("no such command %q", cmd)
	}
}

func flags(name string) *flag.FlagSet { return flag.NewFlagSet(name, flag.ExitOnError) }

func slug(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == '_':
			b.WriteByte('-')
		}
	}
	if out := strings.Trim(b.String(), "-"); out != "" {
		return out
	}
	return "window"
}

func envOr(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}
