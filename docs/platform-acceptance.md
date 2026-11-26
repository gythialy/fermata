# Platform acceptance

How the Windows and Linux apps were checked against the macOS one, what passed,
and what still needs a person. macOS is the shipping platform; the other two
are judged by this record.

## What was run

| Platform | Machine | How |
|---|---|---|
| Windows 11 (25H2, build 26200) | Remote desktop over RDP | `mygo` automation plugin over TCP (`MYGO_AUTOMATION=tcp::9100`), driven by `tools/e2e` |
| Linux (Ubuntu 24.04.5 LTS, arm64) | ArcBox machine, headless | `gofmt`, `go vet`, `go test`, `xvfb-run go test`, `mygo build`, `dpkg` |

Both used an e2e build: `GOFLAGS=-tags=e2e MYGO_INSPECTOR=1 go tool mygo build`.
Release builds carry neither the plugin nor the socket.

## Results

| Behaviour | Windows | Linux |
|---|---|---|
| `gofmt -l .`, `go vet ./...` | clean | clean |
| `go test ./...` | — | pass (bare and under `xvfb-run`) |
| Settings pages (four, both languages) | pass | — |
| Language switching (system / English / 简体中文) | pass | — |
| Toggle switch drives state | pass | — |
| Stepper bounds (disabled at minimum) | pass | — |
| Mini break window, countdown ticking | pass | — |
| Postpone closes the break | pass | — |
| Strict break offers no exit and ends by itself | pass | — |
| Start at login writes and removes the Run entry | pass | — |
| Electron-free packaging (deb/rpm/tar.gz/PKGBUILD) | — | pass |
| App starts headless (software drawing) | — | pass |
| Tray icon reads at tray size | pass (screenshot) | not checked |
| App exclusions hidden where processes cannot be listed | pass, after the fix | — |
| Switches carry accessible names | pass, after the fix | — |

## Defects found

1. **Application exclusions are offered on Windows and can never match.**
   `prefs.globalPage` draws the section on every platform, but
   `presence.RunningProcesses` is `//go:build !unix` and returns an error, so a
   rule can be added and never fires. The Focus row already shows the pattern
   to follow: a capability on `prefs.Options` and a guard in the view.
2. **Switches have no accessible name.** Every `switch` node in the
   accessibility tree carries states only, while tabs, popups and steppers
   carry names. A screen reader announces "switch" with no subject, which
   also leaves automation nothing to address but an ID.

Both are fixed: `presence.ProcessesReadable()` gates the exclusions section
through `prefs.Options`, and every switch carries the text of its row as its
label. `internal/prefs/view_test.go` and `internal/welcome/view_test.go` fail
if a switch is unnamed again.

Checked again on the same machine after the fix: the six switches of the
General page report their row's text (`登录时启动`, `离开时暂停`, …), the
exclusions section is gone, and three captures of the page taken two seconds
apart are byte-identical, so naming an element changes no pixels.

## Only a person can check

- The tray icon and its menu on Windows, and the notification-area behaviour
  (the automation protocol drives windows, not the system tray).
- System notifications: whether the notice appears, and whether clicking it
  opens the popover.
- The global end-break shortcut (`Ctrl+X`): it is an OS hotkey, so a window
  event cannot reach it.
- Display scaling above 100%, multiple displays, and a dark taskbar.
  `MyGo` whitens a template tray icon on a dark taskbar
  (`internal/windows/tray.go`, `SetImage`), which the code review confirms and
  the run did not exercise.

## Re-running it

The repeatable half is now a test suite; see
[development](development.md#end-to-end-tests).

```sh
# on this machine: build, run, test, stop
make e2e

# on another one: build the app there, then drive it from here
GOFLAGS=-tags=e2e MYGO_INSPECTOR=1 go tool mygo build -platform windows/amd64 -skip-installer
set MYGO_AUTOMATION=tcp::9100
set FERMATA_E2E_DATA=%TEMP%\fermata-e2e
Fermata.exe preferences

FERMATA_E2E_ADDR=<host>:9100 FERMATA_E2E_APP=windows go test -tags e2e ./internal/e2e
go run ./tools/e2e -addr <host>:9100 shotall -out build/e2e
```

A break or a settings window has to be opened on the machine itself
(`Fermata.exe mini`, `Fermata.exe preferences`): the app forwards a second
launch's arguments, and the driver cannot start a process on another host.
