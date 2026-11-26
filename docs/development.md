# Development guide

How to run, build, test, localise and package Fermata. The code itself is
described in [architecture.md](architecture.md); the rules the app must satisfy
are in [functional-spec.md](functional-spec.md).

## Requirements

| Tool | Version | Needed for |
|---|---|---|
| macOS | 13+ | Everything (the only shipping platform) |
| Go | 1.27+ | Build, test, tools |
| Xcode | 26+ | `make icons` (`actool` compiles the app icon) |
| `rsvg-convert` | any recent | `make assets` (`brew install librsvg`) |
| Python + fontTools | 3.11+ | Rebuilding the bundled serif subsets |

The MyGo toolchain is a Go tool dependency in `go.mod`; there is nothing to
install separately:

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

## GitHub assets

```sh
make assets
```

`tools/banner/render.sh` renders `tools/banner/banner.svg` into
`.github/assets/banner.png` (2560×1280) and `social-preview.png` (1280×640). It
points `fontconfig` at `internal/fonts/assets` so the wordmark is set in
Spectral and 停琴 in Noto Serif SC, the same faces the break window uses.
`rsvg-convert` is the only extra tool.

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
| `release.yml` | Push a tag matching `v*` | Injects the tag's version and commit into `main.go`/`mygo.json`, runs `make check`, builds the universal ad-hoc DMG, and creates the GitHub release with `Fermata-<version>.dmg`, a rolling `Fermata.dmg` and `SHA256SUMS` |
| `nightly.yml` | Daily at 17:00 UTC, manual | Injects `<version>-nightly.<date>.<sha>` into `main.go`/`mygo.json`, runs `make check`, builds the universal ad-hoc DMG, and refreshes the rolling `nightly` prerelease with `Fermata-nightly.dmg` and `SHA256SUMS` |

The release is ad-hoc signed: no Apple certificate, no notarization. CI has no
secrets to configure.

The nightly is one rolling prerelease: its tag moves to the head of `main`, so
`releases/download/nightly/Fermata-nightly.dmg` always points at the latest
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
4. Tag and push: `git tag vX.Y.Z && git push origin vX.Y.Z`. The workflow builds,
   tests and publishes the DMG and checksums.
5. Install from the published DMG on a clean account once: first run,
   notification permission, login item, a break on each display.
6. Check the tap: `brew upgrade --cask fermata`, or run the tap's update
   workflow manually.
