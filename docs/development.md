# Development guide

How to run, build, test, localise and package Fermata. The code itself is
described in [architecture.md](architecture.md); the rules the app must satisfy
are in [functional-spec.md](functional-spec.md).

## Requirements

| Tool | Version | Needed for |
|---|---|---|
| macOS 13+, Windows 11, Linux | — | The three platforms the app ships on |
| Go | 1.27+ | Build, test, tools |
| NSIS | any recent | The Windows installer, when building off Windows |
| Xcode | 26+ | `make icons` (`actool` compiles the app icon) |
| `rsvg-convert` | any recent | `make assets` (`brew install librsvg`) |
| Python + fontTools | 3.11+ | Rebuilding the bundled serif subsets |

The app needs no cgo, so any machine builds for all three platforms; only the
tools of a platform sign and package their own artifacts. The MyGo toolchain is
a Go tool dependency in `go.mod`; there is nothing to install separately:

```sh
go tool mygo version
```

## Run

```sh
go run .
```

The app has no main window: look at the menu bar. The first launch shows the
welcome window, and closing it starts the schedule too. Quit from the menu bar
menu.

## Everyday targets

```sh
make run          # go run .
make build        # go tool mygo build -skip-dmg, output in build/
make dmg          # full build with a disk image
make test         # go test ./...
make test-race    # go test -race ./...
make vet          # go vet ./...
make fmt          # gofmt -w .
make check        # gofmt check, vet, tests — the gate before a commit
make help         # print the targets
```

`make check` is the minimum for a commit. The CI-less project relies on it,
plus the on-device acceptance cases in the functional spec.

### Build facts

- Output: `build/darwin-universal/Fermata.app` (arm64 + x86_64). The Makefile
  passes `-platform darwin/universal`; `PLATFORM=darwin/arm64 make build` gives a
  quick single-architecture build for local work.
- Signing is **ad-hoc**: no Developer ID, no notarization. macOS asks the user
  to confirm the first launch.
- App identity (`name`, `slug`, `identifier`, `version`, icon, i18n source)
  lives in `mygo.json`.
- The link-time identity lives in `main.go`:
  `-ldflags "-X main.version=… -X main.build=… -X main.commit=…"`.

## Install

### Homebrew

```sh
brew tap gythialy/tap
brew install --cask fermata
```

The cask lives in [gythialy/homebrew-tap](https://github.com/gythialy/homebrew-tap).
It clears the download quarantine; Fermata is ad-hoc signed and not notarized,
so if macOS still blocks the first launch, right-click the app and choose Open.
Update with `brew upgrade --cask fermata`.

### From source

```sh
make build
open build/
```

Drag `Fermata.app` to `/Applications`. Because builds are ad-hoc signed, macOS
asks for confirmation on the first launch (System Settings → Privacy & Security
→ Open Anyway). Notifications need permission; the login item needs the app to
be packaged, not run from source.

## Command line

A second launch forwards its arguments to the running app:

```text
fermata                          start the app, or send it a command
fermata help                     print help
fermata version                  print the version
fermata pause [duration]         pause breaks; the default is indefinite
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
```

Durations read `30` (minutes), `90m`, or `1h20m`; `indefinitely` and
`until-morning` apply to `pause`. A strict break refuses the commands that
would end it.

## Localisation

The interface is written in English and Simplified Chinese. The catalogs are
the source of truth:

```
locales/en.json        source language
locales/zh-Hans.json   translation
locales/catalog_gen.go generated accessors — do not edit
```

Workflow:

1. Add or change the message in **both** JSON files. Keep placeholder names and
   plural forms identical.
2. Regenerate the accessors:

   ```sh
   go tool mygo generate
   ```

3. Call the generated accessor from the view, e.g. `locales.SettingsTitle()`.
   Never build a `map[string]string` by hand.
4. Check the catalogs:

   ```sh
   go tool mygo i18n check
   ```

`mygo.json` declares `"i18n": {"dir": "locales", "source": "en"}`, so the
accessors and the app's available languages follow the catalogs.

Break suggestions are not UI strings: they are a shuffled deck in
`internal/ideas`, one JSON file per language, and they fall back to the English
deck when a language has none.

## Fonts

The break voice uses two OFL families, committed under
`internal/fonts/assets/`:

| File | Family | Weight |
|---|---|---|
| `spectral-light.ttf` | Spectral | 300 |
| `spectral-regular.ttf` | Spectral | 400 |
| `noto-serif-sc-light.otf` | Noto Serif SC (subsets) | 300 |
| `noto-serif-sc-regular.otf` | Noto Serif SC (subsets) | 400 |

The Noto files are cut down to the characters the catalogs and suggestions can
draw. Rebuild them after changing either:

```sh
python3 tools/fonts/subset.py \
  --spectral-light /path/to/Spectral-Light.ttf \
  --spectral-regular /path/to/Spectral-Regular.ttf \
  --noto-serif /path/to/NotoSerifCJK.ttc
```

`internal/fonts` tests that every character in the catalogs and suggestions is
covered by the shipped fonts; a missing glyph fails the build instead of
silently falling back.

## Icons

```sh
make icons     # go run ./tools/genicons
```

The tool draws the app icon in light and dark appearances, the menu bar
template, and compiles `icons/AppIcon.icon` into `icons/AppIcon.car` with
`xcrun actool`. It writes `resources/icon.png` (the `.icns` fallback) and
`resources/tray.png` too. The geometry comes from `internal/mark`, the same
mark the interface draws — never trace a new one.

`tools/genicons` needs Xcode 26+; the compiled `.car` is committed so machines
without Xcode can still build.

## Sounds

```sh
go run ./tools/gensounds
```

This writes the four original 16-bit mono WAVs into
`internal/sounds/assets/`, embedded with `go:embed`. They are synthesized, not
recorded; keep them short, quiet and free of clipping.

## Screenshots

```sh
make screenshots
```

This runs build-tagged tests (`-tags screenshots`) in `internal/breakwin`,
`internal/prefs`, `internal/welcome` and `internal/popover`. Each view is
rendered with `ui.NewTester` at 2×, once in English and once in `zh-Hans`:
English writes to `docs/images/` and Chinese to `docs/images/zh-Hans/`, so each
README shows its own language.

| File | View |
|---|---|
| `break-mini.png`, `break-long.png` | The break window, short and long |
| `settings-general.png`, `settings-schedule.png`, `settings-appearance.png`, `settings-about.png` | The four settings pages |
| `welcome.png` | First run |
| `popover.png` | The status popover |

Each file above exists in both folders. The tests draw fixed suggestions, so a
re-run only changes an image when a surface changes. These are real renders of
the app's views — not mockups: when a surface changes, re-run the target so the
README and the docs do not drift. The test files are excluded from normal runs
by the build tag.

The break window's two stills are written with the field's rounded corners
transparent, so they sit on GitHub's light and dark themes without a white
halo. The headless renderer starts from a white canvas, which its anti-aliased
corner is blended with; the test un-blends that white out of each corner and
writes the field's colour at the coverage the renderer drew, so the shape is the
renderer's own and not a guessed radius.

## Animation

```sh
make animation
```

`TestAnimation` in `internal/breakwin` draws the break window's first seconds
frame by frame: the content settles, the hairline fills with the break, the
countdown steps down, and the exit changes from `Postpone` to `Skip` once thirty
percent of the break has passed. Time comes from `ui.Tester.Advance`, so the
clip is the same on every run and nothing is recorded from a screen.

The frames land in `build/screens/break-loop/<locale>/`, which git ignores, and
`tools/screens/animate.sh` encodes them with `ffmpeg`:

| File | Use |
|---|---|
| `break-loop.png` | The README's loop: an animated PNG, 1680×1120, 24 frames a second, with the field's corners transparent |
| `break-loop.webm`, `break-loop.mp4` | The hero of `docs/index.html`, played once and resting on its last frame |

The loop is an APNG that keeps its `.png` name: GitHub plays an animated PNG
with that extension, and unlike a GIF it holds 8-bit alpha for the corners at a
quarter of the bytes. A client that does not animate it shows the first frame,
which is the same field the stills show.

The video is H.264 and VP9, neither of which carries alpha through a plain
RGBA-to-YUV conversion (the corners would come out black), so `animate.sh`
mattes each frame on the field's own colour, sampled from its left edge.

Each file exists in both folders, like the stills. Two rules come from the
design system: the mark never moves, and motion only carries a state change —
no pulses, no loops for their own sake. The window's own fade-in and closing
belong to the window layer, not to the view, so the clip starts with the content
rising and ends on the field.

`ffmpeg` is the only extra tool (`brew install ffmpeg`), and only `make
animation` needs it; `make screenshots` and `make check` are unchanged. The
encode is pinned to one thread and bit-exact output, so an unchanged view gives
byte-identical files on a re-run.

## GitHub assets

```sh
make assets
```

`tools/banner/render.sh` renders `tools/banner/banner.svg` into
`.github/assets/banner.png` (2560×1280) and `social-preview.png` (1280×640). It
points `fontconfig` at `internal/fonts/assets` so the wordmark is set in
Spectral and 停琴 in Noto Serif SC, the same faces the break window uses.
`rsvg-convert` is the only extra tool here.

The repository page uses these files as follows:

| File | Use |
|---|---|
| `.github/assets/banner.png` | README hero |
| `.github/assets/social-preview.png` | Settings → Social preview |
| `.github/assets/mark.svg` | Vector mark |
| `.github/assets/icon.png` | Repository avatar |

## Tests

| Area | Where | How |
|---|---|---|
| Planner state machine | `internal/planner` | Table-driven cases plus a long random walk that checks invariants |
| Settings | `internal/settings` | Defaults, validation, corruption backup, atomic and debounced writes |
| CLI | `internal/cli` | Parse table |
| Presence | `internal/presence` | Idle, Focus, lock, exclusions with fake readers |
| Tray and icons | `internal/tray`, `internal/trayicon`, `internal/status` | Menu matrix, drawn icon pixels |
| Views | `internal/breakwin`, `internal/prefs`, `internal/welcome`, `internal/popover` | `ui.NewTester`: texts, clicks, smallest window, frame settling |
| Fonts | `internal/fonts` | Glyph coverage of the shipped subsets |

The general rule: state lives in plain structs; the view is a pure function of
state; anything that needs the system goes through a narrow interface with a
fake in tests.

### Release artifacts

Every platform names its files the same way, in lower case, with the platform
and the architecture spelled out (Go's `amd64` and `arm64`; a `PKGBUILD` keeps
the name pacman expects):

```
fermata-<version>-macos-universal.dmg
fermata-<version>-windows-amd64-setup.exe | -portable.zip
fermata-<version>-linux-amd64.deb | .rpm | .tar.gz
install.sh, PKGBUILD-x86_64, PKGBUILD-aarch64
```

Nothing on a release is unversioned: the landing page links to the release
page, which lists the files of that tag. The nightly keeps stable names
because it is one rolling release, with the date and commit in its notes.

The Homebrew cask in `gythialy/homebrew-tap` builds its URL from the version,
so a rename reaches it only with a release that already carries the new name:
update `scripts/update.py` and the cask in the same cycle as that tag, never
before it.

### End-to-end tests

The tests above run a view in memory. The end-to-end suite drives the **built
app** through MyGo's automation protocol, so the same cases run on macOS,
Windows and Linux:

```sh
make e2e                 # build here, run, stop
go test -tags e2e ./internal/e2e   # against an app already running
```

`tools/e2e/run.sh` builds an e2e app (`-tags e2e`, inspector kept), starts it
in a temporary `FERMATA_E2E_DATA` directory with `MYGO_AUTOMATION=tcp::9100`,
waits for the socket, and stops it after the run. That directory and the name
`Fermata E2E` keep the run from touching an installed copy, and from being
forwarded to it.

Against an app on another machine, build there and point the suite at it:

```sh
FERMATA_E2E_ADDR=192.0.2.10:9100 FERMATA_E2E_APP=windows go test -tags e2e ./internal/e2e
```

`FERMATA_E2E_APP` names the platform **under test**, which is not the test's
own when the app runs elsewhere; `TestExclusionsFollowThePlatform` needs it.

A test skips when nothing answers, and `TestBreakWindowIsReadable` skips until
a break window exists (start one with `Fermata mini` on the app's machine).
Tray menus, notifications and the global end-break shortcut are out of the
protocol's reach; they stay in [platform acceptance](platform-acceptance.md).

`tools/e2e` is the same protocol as a command, for looking at what a platform
shows while debugging it:

```sh
go run ./tools/e2e windows
go run ./tools/e2e tree -window 1
go run ./tools/e2e shotall -out build/e2e
```

## Code conventions

- Comments and identifiers in English; comments explain **why**, briefly.
- No `time.Now` or `rand` in the planner: time and the deck are injected, so
  tests are deterministic.
- Never persist `ui` elements across frames; keep handles, not elements.
- Keep the platform shell native: no hand-drawn menus or title bars.
- New user-visible behaviour needs a spec line and a test; new visuals need a
  design-system token and, when visible, a fresh screenshot.
- Do not commit build output (`build/`), local agent files, or secrets.

## CI and releases

Two GitHub Actions workflows live in `.github/workflows/`:

| Workflow | Trigger | What it does |
|---|---|---|
| `ci.yml` | Push to `main`, pull requests, manual | `make check`, then a universal `make build` on a macOS runner |
| `release.yml` | Push a tag matching `v*` | Opens a draft release, then builds the universal ad-hoc DMG, the Windows installers and portable archives, and the Linux packages on their own runners; merges the per-platform checksums and publishes |
| `nightly.yml` | Daily at 17:00 UTC, manual | Stamps `<version>-nightly.<date>.<sha>`, builds the same three platforms and refreshes the rolling `nightly` prerelease, keeping only the names it still ships |

The release is ad-hoc signed: no Apple certificate, no notarization. CI has no
secrets to configure.

The nightly is one rolling prerelease: its tag moves to the head of `main`, so
`releases/download/nightly/fermata-nightly-macos-universal.dmg` always points at the latest
build. It is unsigned and may be broken; it never replaces a stable release.

Dependabot checks Go modules and GitHub Actions weekly (`.github/dependabot.yml`):
minor and patch bumps arrive grouped, majors separately.

The Homebrew tap ([gythialy/homebrew-tap](https://github.com/gythialy/homebrew-tap))
runs its own scheduled workflow (`update.yml`, daily and manual). It reads the
latest stable release of Fermata and of [exDesktop](https://github.com/metawka/exDesktop),
takes each asset's SHA-256 digest from the GitHub API, and commits refreshed
casks. The Fermata cask starts as a rolling `:latest` cask and becomes versioned
once the first release exists.

## Release checklist

1. `make check` passes.
2. `make screenshots` and `make assets` are current if a surface changed.
3. Bump the version in `main.go` and `mygo.json` so a local build reports the
   same number; the release workflow overrides them from the tag.
4. Tag and push: `git tag vX.Y.Z && git push origin vX.Y.Z`. The workflow builds
   every platform, merges the checksums and publishes the release.
5. Install from the published DMG on a clean account once: first run,
   notification permission, login item, a break on each display.
6. Check the tap: `brew upgrade --cask fermata`, or run the tap's update
   workflow manually. When the artifact names changed, update the tap's
   `scripts/update.py` and the cask after the release exists, never before:
   the cask builds its URL from the version.
