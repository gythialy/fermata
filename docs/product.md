# Fermata product document

- **Version** 1.0, as built (v1.0.0)
- **Platform** macOS 13+, Windows 11 and Linux ship from the same code
- **Foundation** MyGo 0.10.0, Go 1.27, single process, no webview
- **Companions** [functional-spec.md](functional-spec.md) owns behaviour,
  [design-system.md](design-system.md) owns visuals,
  [architecture.md](architecture.md) owns code structure, and
  [development.md](development.md) owns the toolchain

This document keeps the product-level material only: what Fermata is for, what
is in scope, which decisions are settled, what is still open, and where the
project stands.

## 1. What Fermata is

A menu bar app that interrupts work on a rhythm. It is inspired by
[Stretchly](https://github.com/hovancik/stretchly) and follows its user-facing
behaviour — no shared code.

### 1.1 The name

*Fermata* is the Italian musical term for a hold: the sign above a note that
tells a player to sustain it beyond its written value, then move on. 停琴, "the
qin stops" (the qin is a Chinese zither), comes from a line by the Southern Qi
poet Xie Tiao in 《移病还园示亲属诗》: 「停琴伫凉月，灭烛听归鸿」 — the player
stops the strings and stands in the cool moonlight, blows out the candle to
hear the returning geese.

The two names describe the same pause, and the mark carries it: the arc is the
fermata, the dot beneath is the finger resting on the string.

Three moments carry the product:

| Moment | Goal | Tone |
|---|---|---|
| The break window | Experience | Immersive, one field, a serif suggestion, no app name |
| Settings and menu | Operate | Restrained paper-and-ink neutrals, desktop density, native shell |
| First run | Persuade | One page, one decision, a real preview |

The one-sentence design stance: **a break is a note held longer.** No clock
faces, breathing circles, clouds, streaks or badges — one fermata, one field,
one exit.

## 2. Goals

1. Reproduce Stretchly's core experience: a resident menu bar item, breaks on
   schedule, a break window, postpone / skip / pause / strict mode.
2. Native desktop integration on macOS: menu bar, system notifications, login
   item, Focus and lock/sleep awareness, light and dark appearance, multiple
   displays.
3. Far lower resource use than the Electron original: one process, no webview,
   idle CPU near zero.
4. Testable: scheduling is a pure Go state machine; views are verified
   headlessly with `ui.NewTester`.
5. Executable design: every surface, state, string and size is specified, so
   implementation keeps no room for visual improvisation.

## 3. Non-goals

- Windows and Linux **releases** in v1. The code builds there; platform
  adaptation and on-device acceptance are a later milestone.
- Accounts, sync, contributor preferences, sponsorship and other Stretchly
  extras.
- Silent auto-update.
- Importing Stretchly's configuration (the two apps coexist, isolated).
- 54 languages: v1 ships English and Simplified Chinese, with a catalog shaped
  for more.
- A webview, a frontend build step, or a plugin system.

## 4. Success criteria

| Area | Criterion | State |
|---|---|---|
| Behaviour | Default rhythm, notices, postpone and skip match the functional spec; T01–T36 pass | Unit-tested; on-device cases listed in the spec |
| Resources | Idle RSS ≤ 80 MB, idle CPU 0.0–0.1%; ≤ 120 MB while a break is shown | Budget; not yet recorded on hardware |
| Stability | 8 hours across sleep, wake, lock and display changes without leaks, missed or replayed breaks | Manual soak pending |
| Distributability | universal2 `.app` + `.dmg`, ad-hoc signed, installs and starts at login | Builds and DMGs; no notarized download |
| Design consistency | Break window, settings, welcome, tray and popover match the design system | Implemented on macOS |

The resource numbers are budgets from the design, not measurements. When a
measurement run exists, it belongs in this table, with its method.

## 5. Scope

| # | Feature | Priority | State |
|---|---|---|---|
| 1 | Mini and long break scheduling, notices, postpone, skip, strict mode | P0 | Done |
| 2 | Break window: multi-display, 85% field, countdown, suggestion, one exit | P0 | Done |
| 3 | Menu bar: status text, menu, three icon styles, dark appearance | P0 | Done |
| 4 | Manual pause (30m / 1h / 2h / 5h / until morning / indefinite), resume, reset, skip ahead | P0 | Done |
| 5 | Settings: General, Schedule, Appearance, About, restore defaults | P0 | Done |
| 6 | System notifications, advance notice, muted, no action buttons | P0 | Done |
| 7 | Sounds: start and end, four originals, volume, mute | P0 | Done |
| 8 | Automatic pauses: idle, Focus / Do Not Disturb, lock and sleep | P0 | Done |
| 9 | Start at login, single instance | P0 | Done |
| 10 | Command line: pause, resume, toggle, reset, skip, postpone, finish, mini, long, last, preferences, version, help | P0 | Done |
| 11 | i18n (en, zh-Hans), light and dark | P0 | Done |
| 12 | First-run welcome window | P0 | Done |
| 13 | Manual finish mode, current time in the break window | P0 | Done |
| 14 | App exclusions (pause or resume while an app runs) | P1 | Done |
| 15 | Background image and transparency | P1 | Configurable in `config.json`; no UI. Opacity is a P0 Appearance slider |
| 16 | Update check against GitHub Releases | P1 | Done |
| 17 | Break health counter | P1 | Done |
| 18 | Custom break suggestions | P1 | Not done |
| 19 | Sunrise/sunset times from coordinates | P2 | Not done |
| 20 | 54 languages (Stretchly translations, with attribution) | P2 | Not done |
| 21 | Developer ID signing, notarization, auto-update | P2 | Not done |

Priority means: **P0** shipped in v1, **P1** the first round after v1, **P2**
deliberately deferred.

## 6. Decisions

Settled decisions, kept for the record. Numbers match the design notes;
implementation choices (A1–A10) follow in §6.1.

| # | Decision | Consequence |
|---|---|---|
| D1 | Build on MyGo, not NSGo | MyGo has the four things this app cannot fake — system-wide shortcuts, idle time, a non-activating panel and a headless UI tester — plus one codebase for macOS, Windows and Linux |
| D2 | Do not import Stretchly's configuration | Separate data directories; the two apps can coexist |
| D3 | Ship the P0 scope first | P1/P2 stay visible in §5 rather than creeping into v1 |
| D4 | No Apple Developer account | Ad-hoc signing; the user confirms the first launch; no notarization |
| D5 | Repository `github.com/gythialy/fermata` | Module path, update endpoint and About links follow it |
| D6 | Design direction *Held Note v2* | Paper and ink, one vermilion signal, Material 3 tokens as structure only, platform shells native |
| D7 | One owner per document | Behaviour → functional spec; visuals → design system; structure → architecture; scope → this document |
| D8 | One graphic asset: the fermata | App icon, menu bar, break window, welcome, sidebar and About share one geometry (`internal/mark`); no hourglass |
| D9 | Bundle the break serif | Spectral (Latin) + Noto Serif SC subset (Han), OFL; UI stays on the system sans; coverage is tested |
| D10 | Respect the system text scale | `theme.Px` applies it to UI text, capped at 1.5; the scale preview is exempt |
| D11 | English name *Fermata*, Chinese 停琴 | Bundle `dev.gythialy.fermata`, module `github.com/gythialy/fermata` |

Behavioural choices that differ from Stretchly are settled in the spec, not
re-argued here: strict mode allows neither postpone nor skip; sleep and wake do
not replay a missed break; a break has one exit at a time; long breaks have
their own interval; manual finish is P0.

### 6.1 Implementation decisions

The code-shaping decisions behind those product choices:

| # | Decision | Why |
|---|---|---|
| A1 | The planner is a pure function of inputs and time | The whole schedule, sleep and missed breaks included, is exhaustively testable; the clock and the OS stay in one driver |
| A2 | One main thread, `RunOnMain` for everything else | Matches MyGo's model; no locks around UI or planner state, no lock-order bugs |
| A3 | Pausing is a set of reasons, not a boolean | Focus, idle, lock and exclusions overlap; the tray can name the reason that matters, and removing one does not resume while another holds |
| A4 | One break window per display, one manager owning them | Display changes and rebuilds stay in one type; views stay stateless and per-frame |
| A5 | The planner owns the end time | A window cannot close a break by mistake; the view displays and reports intent only |
| A6 | Debounced, single-writer settings | A slider writes once; atomic replace means a crash cannot truncate `config.json` |
| A7 | Tray menu updated in place | An open menu survives a state change, which matters while the user reads the changing status line |
| A8 | Icons drawn at runtime, cached | No PNG atlas, exact template alphas, a ring at any share, and the result is decodable in a test |
| A9 | Theme, fonts and wording assembled in the app layer | Views take a theme and strings per frame, so a test can render them with markers and a language change needs no restart |
| A10 | No file watching of `config.json` | One source of truth and no half-applied external edit; an external change applies at the next launch |

Two later decisions sit outside the table: `savedstate` (a waiting countdown
survives a quick restart) and the health counter (a soft, opt-in signal).

## 7. Compatibility, data and permissions

- **System** macOS 13.0 or later, universal2.
- **Coexistence** bundle id `dev.gythialy.fermata`, data directory
  `~/Library/Application Support/Fermata`, log `~/Library/Logs/Fermata/fermata.log`;
  isolated from Stretchly.
- **Shortcuts** Electron-style accelerators (`CmdOrCtrl+X`), native to MyGo.
- **Command line** aligned with Stretchly's subcommands; semantics in the spec.
- **Permissions** notifications must be granted (a refusal is logged, not
  nagged); the login item needs a packaged `.app`; Focus detection reads a user
  file and degrades to "unavailable" without extra access. Fullscreen detection
  reads window bounds only and needs no screen-recording permission.
- **Signing** ad-hoc. If a Developer ID appears later, set
  `macos.signingIdentity` in `mygo.json`; nothing else changes.
- **Windows/Linux** build from the same code; tray titles, panel focus, virtual
  desktops, Wayland and notification packaging are platform work, not
  assumptions (see the design system's platform section).

## 8. Milestones

| Milestone | Deliverable | State |
|---|---|---|
| M0 | Repository skeleton, `mygo.json`, tray icon, build | Done |
| Design | Functional spec, design system v2, canvas, architecture | Done |
| M1 | Settings, planner, scheduler, CLI, tray; unit tests | Done |
| M2 | Break window, notifications, sounds, `⌘X`, manual finish, strict mode, multi-display | Done |
| M3 | Settings window, welcome window, i18n, login item, idle / DND / lock | Done |
| M4 | Universal packaging, DMG, ad-hoc signing, README, measurement | Packaging, signing and README done; measurement not recorded |
| M5 | Windows and Linux builds, platform adaptation, installers | Not started |

All four design milestones landed in v0.1.0: the planner, scheduler, settings,
CLI and tray (M1); break windows on one and many displays, notifications,
sounds, the shortcut, manual finish and strict mode (M2); the settings pages,
welcome window, presence and localisation (M3); packaging, the DMG, ad-hoc
signing, resources and the documentation images (M4). What remains is the
on-device verification, not unimplemented scope.

## 9. Risks

| Risk | Mitigation | State |
|---|---|---|
| MyGo is young; a `Panel` may take focus when a break appears | Verify on device (S1); fall back to a plain always-on-top window plus `ShowInactive` | Open |
| Focus detection reads a private file; macOS versions differ | Probe once at launch; when unreadable, hide the row and stop polling | Handled by design |
| No Developer ID | Ad-hoc signing plus install notes | Accepted |
| Timer drift across sleep | Monotonic clock, wake events, 2-second lateness threshold; never replay a missed break | Handled by design |
| Notifications denied | Log, do not block, explain in settings; no pre-query | Handled by design |
| The mark at 18 px loses legibility | Optical sizes in the design system; menu bar template checked at 1× and 2× | On-device check pending |
| Fonts and Han fallback | Bundled subsets with a coverage test; no runtime download | Done |
| The break window can steal focus | `Panel` + `ShowInactive`; on-device check | Open |
| Windows/Linux platform gaps (tray title, panel focus, Wayland, DND) | Same stack builds; adapt per platform and verify on hardware before promising a release | Deferred to M5 |
| GUI cannot be checked in an automated session | Headless view tests plus manual on-device acceptance | Ongoing |

## 10. Versioning and release

The app version lives in `main.go` (`version`, `build`, `commit`) and can be set
at link time. The bundle version and identifier live in `mygo.json`. A release
is cut by pushing a `v*` tag: GitHub Actions injects that version, runs the
checks, builds the universal ad-hoc DMG and attaches it to the release with a
`SHA256SUMS` file. The update check reads `releases/latest` and never installs
anything. Homebrew users install through `gythialy/homebrew-tap`
(`brew tap gythialy/tap`, then `brew install --cask fermata`); that tap also
carries exDesktop and refreshes both casks from the latest upstream releases.
See [development.md](development.md) for the commands.

## 11. Visual canvas

The visual reference is an Open Artifact canvas: 6 pages, 26 boards, kept in
`artifacts/fermata-ui-ux-design/`. It shows the design system as drawn;
[design-system.md](design-system.md) remains the specification, and when the
two disagree the design system wins. The displayed title is 停琴 (Fermata).

```
artifacts/fermata-ui-ux-design/
├── project/
│   ├── canvas.json          # pages, boards, positions, order
│   ├── Main.dc.html         # cover
│   ├── overview/            # platforms, principles, state model
│   ├── language/            # colour, type, shape/motion, components, icons
│   ├── break/               # mini, long, the four moments
│   ├── settings/            # general, schedule, appearance, about
│   ├── onboarding/          # first run
│   ├── system/              # tray, system states, platform spec, accessibility, audit
│   └── popover/             # status flyout: waiting, moments, implementation, spec
└── .gitignore               # excludes _agent/, history/, .artifact*
```

| Page | Boards |
|---|---|
| 01 Overview | Cover; the three platforms and their menus; principles, Material 3 trade-offs and the state model |
| 02 Design language | Colour roles and palette; type and sizes; shape, elevation and motion; desktop components; icons |
| 03 Break window | Mini break; long break; the four moments and the light field |
| 04 Settings and first run | General; Schedule; Appearance; About; first run |
| 05 System and delivery | Menu bar / tray / indicators; pause, notices and strict mode; platform spec; accessibility; document ↔ implementation audit |
| 06 Status popover | Waiting (macOS material / paper-ink); the four moments; three countdown variants; implementation diagram; spec |

The canvas needs the Open Artifact host to render its `.dc.html` boards; opening
a board directly in a browser does not work. In the host, the canvas view pans
and zooms, the sidebar switches pages, and a board opens full screen. The
switches, sliders and segmented controls in `language/Components.dc.html` are
live; `system/Audit.dc.html` is the visual ↔ document audit. Comments and
"Send to Agent" are forwarded to the session. Local URLs are machine-local and
are not committed.

Maintenance:

1. Change the specification before the canvas: visual decisions land in
   [design-system.md](design-system.md), then the canvas follows.
2. The canvas shows visuals and states, never scheduling behaviour; behaviour
   belongs to [functional-spec.md](functional-spec.md).
3. A new board needs both `boards` and `order` in `project/canvas.json`; board
   file names are unique under `project/`.
4. Do not commit `_agent/`, `history/`, `.artifact*`, `.sharing.json` or local
   logs.
