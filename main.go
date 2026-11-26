// Command fermata is a menu bar break reminder.
package main

import (
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/egoist/mygo"
	"github.com/gythialy/fermata/internal/app"
	"github.com/gythialy/fermata/internal/cli"

	// The catalogs make the app's languages its own, and give every view its
	// words.
	_ "github.com/gythialy/fermata/locales"
)

// version, build and commit are the app's identity; a release build can set
// them with -ldflags "-X main.version=…".
var (
	version = "1.0.0"
	build   = "1"
	commit  = ""
)

func main() {
	mygo.App.SetName("Fermata")
	mygo.App.SetActivationPolicy(mygo.ActivationPolicyAccessory)
	if runtime.GOOS == "darwin" {
		// A menu bar app needs no About item: the About page lives in
		// Settings. Keep the standard editor and window menus for the
		// settings fields, and Hide/Quit for the app.
		mygo.App.SetMenu(mygo.NewMenu([]*mygo.MenuItem{
			{Label: "Fermata", Submenu: []*mygo.MenuItem{
				{Role: mygo.RoleHide},
				{Role: mygo.RoleHideOthers},
				{Role: mygo.RoleUnhide},
				mygo.Separator(),
				{Role: mygo.RoleQuit},
			}},
			{Role: mygo.RoleEditMenu},
			{Role: mygo.RoleWindowMenu},
		}))
	}

	cmd, err := cli.Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		fmt.Fprint(os.Stderr, cli.Help())
		os.Exit(2)
	}
	switch cmd.Name {
	case cli.CmdHelp:
		fmt.Print(cli.Help())
		return
	case cli.CmdVersion:
		fmt.Println(cli.Version(version, commit))
		return
	}

	if !mygo.App.RequestSingleInstanceLock() {
		fmt.Println("Forwarded to Fermata.")
		return
	}

	a := app.New(app.Options{Version: version, Build: build, Commit: commit, Initial: cmd})
	mygo.App.OnSecondInstance(func(args []string, wd string) {
		cmd, err := cli.Parse(args)
		if err != nil {
			log.Printf("fermata: %v", err)
			return
		}
		if cmd.Name == cli.CmdHelp || cmd.Name == cli.CmdVersion {
			return
		}
		mygo.RunOnMain(func() { a.HandleCommand(cmd) })
	})

	mygo.App.WhenReady(a.Start)
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
