# Fermata design system v2 · Held Note (cross-platform)

- **Version:** v2.0 (2026-11-25)
- **Status:** Design baseline v2 — implemented on macOS, Windows and Linux
- **Product:** **Fermata** (停琴) — a break reminder that lives in the menu bar or the system tray
- **Platforms:** macOS 13+, Windows 11, Linux (GNOME / KDE). The same MyGo codebase covers all three since v1.1.0; the shell differences each keeps are recorded in [platform acceptance](platform-acceptance.md)
- **Documents:** [product](product.md) · [functional spec](functional-spec.md) · [architecture](architecture.md) · [visual canvas](product.md#11-visual-canvas) · [development](development.md)

Precedence: **behavior** follows the functional spec; **visuals, sizes, controls and the platform shell** follow this document; **code structure** follows architecture. Any conflict is corrected in the documents first, then in the code.

## 1. Direction

### 1.1 In one line

**A break is a held note.** Fermata does not draw clocks, breathing circles or clouds. It puts one fermata where the day stops for a moment, then steps back into the menu bar or the tray.

### 1.2 Three scenarios

| Scenario | Goal | Design voice |
|---|---|---|
| Break window | Experience | Immersive, one full field, serif suggestion, no app name |
| Settings / menu | Operate | Restrained, paper-and-ink neutrals, desktop density, native shell |
| Welcome window | Persuade | One page, one decision, a real preview |

### 1.3 Five principles

1. **One thing at a time.** The break window has a single exit at any moment; first launch has a single decision.
2. **Exist quietly.** No focus stealing, no chimes, no streaks, no badges; the brand only takes over during a break.
3. **Same content, different shell.** Menus, window decoration, shortcuts and notifications follow the platform; content and tokens are not duplicated per platform.
4. **During a break, the brand takes over.** One full field, a serif suggestion, one hairline of progress; no Material components and no app name.
5. **Predictable.** Every state has words; strict is not a red alarm, it is an explicit absence of an exit.

### 1.4 What we take from Material 3, and what we do not

| Take | Do not take |
|---|---|
| Role colors (primary / container / surface / outline) | Dynamic color from the wallpaper |
| Typescale (display → caption) | FAB |
| Shape scale (4 / 8 / 12 / 20) | Bottom navigation bar |
| State layers (hover 8% / pressed 12%) | Everything as a pill |
| Motion tokens (120–300 ms) | Mobile 48–56 px whitespace |
| Desktop density (32 px controls) | Surface tonal wash |
| 2 px focus ring | Default purple |
| Elevation 0–3 | Cards inside cards |

Material 3 is **structure** here, not a skin. No Material component's appearance is copied onto the interface as-is.

### 1.5 Anti-references (deliberately not done)

- Teal-to-green gradients, clouds, leaves, landscapes, breathing circles, meditation illustration
- Streaks, badges, points, leaderboards, pets
- Glassmorphism, colored glows, neon outlines, gradient text
- Staves, clefs, note heads, piano keys, metronomes, gramophones, waveforms
- Rounded cards inside rounded cards; one card per setting
- Decorative animation, looping pulses, staged entrances
- Three hand-drawn menus, three hand-drawn title bars

### 1.6 Cross-platform stance

- **Identical content:** break window, settings pages, welcome window, copy and tokens are the same everywhere.
- **The shell belongs to the system:** menus, window decoration, notifications, login items and file dialogs are implemented per platform. Do not imitate native either: no self-drawn menus or title bars, and no macOS traffic lights painted onto Windows.
- **One implementation stack:** MyGo 0.10.0 supports macOS, Windows and Linux (and iOS/Android); MyGo draws the native UI on all three. v1 ships macOS; Windows and Linux are platform-adaptation and acceptance milestones on the same codebase, with no second renderer.
- **Platform semantics differ and must be designed, not assumed:** tray titles, panel focus and virtual desktops, Wayland, notification packaging and Do Not Disturb detection are not equivalent across the three platforms; see §6.

## 2. Identity

### 2.1 The mark

A fermata: one shallow arc with a dot beneath it. It is the product's **only** graphic asset, and there is exactly one geometry in code (`internal/mark`). The app icon, the menu bar, the break window, the welcome window, the sidebar and the About page all take their shape from it; no second mark (an hourglass, for example) may appear.

```
viewBox 0 0 32 32
arc:   M 6 18 C 8 8.5, 24 8.5, 26 18    round cap
dot:   cx 16, cy 25                     fill
visual bounding box: x 4.9–27.1, y 9.8–27.6
```

The same 32-unit geometry is scaled by `side / 32` at every size, and a small copy takes a heavier stroke and dot so it holds. Optical sizes (one curve, thickened as it shrinks; units are the 32 canvas unless stated):

| Surface | Canvas | Arc width | Dot radius | Notes |
|---|---|---|---|---|
| Menu bar | 18 | 2.5 | 2.2 | Optical values for an 18 canvas (≈ 4.4 / 3.9 on the 32 canvas); template image; arc control points `(1.7,10.4) (4.1,2.7) (13.9,2.7) (16.3,10.4)`, dot `(9,14.5)` |
| Break window | 48 | 2.6 | 2.9 | Vermilion `#F0836C` on a dark field, `#B8432F` on a light one |
| Welcome preview / About | 44 / 49 | 2.6 | 2.9 | Same shape as the break window |
| Sidebar / brand row | 22 / 30 | 3.2–4.2 | 3.0–3.6 | Bolder at small sizes, so it still reads on paper |
| App icon | 700 | 2.2 | 2.6 | Centered in the 824 safe area; ink arc + vermilion dot |

- The mark never rotates, never takes a shadow and is never animated. It is a printed mark, not a loading indicator.
- On a light field the mark is ink `#171A17` with vermilion `#B8432F`; on a dark field it is paper `#F3F2EA` with vermilion `#F0836C`.
- The menu bar template image is tinted by the system; the in-app copies use the colors in this table and do not depend on a system tint.

### 2.2 Tray / menu bar icon states

| State | Graphic | Notes |
|---|---|---|
| Normal | The mark | One-color template image; the system tints it on macOS, `Tray.Dark()` drives a repaint on Windows/Linux |
| Time style | The mark + `9m` | **macOS menu bar title; the Linux tray label depends on the host (AppIndicator); Windows has no title** (MyGo `SetTitle` is a no-op there) — the time goes to the tooltip and the menu |
| Progress style | 16–18 pt ring + center dot | Track at 25% alpha, used arc at 100%; ring 1.6 pt wide, center dot r 1.3 |
| Paused / DND / idle / locked | Two bars (1.7×7.4, radius 0.8) | The generic pause meaning; no moon, no ZZZ; bar centers 5.2 apart |
| Strict break | The mark | No lock; strict is carried by the menu copy and the break window's label |

- Icon size: v1 hands one 36 px template (18 pt at 2×) to every platform; the per-platform sizes (Windows 16/20/24 px, Linux 22 px symbolic) are platform work for their release, and the system scales the template.
- Title: `9m` (no space), `0m`, at most `99m`; never `9′`. Icon updates do not flicker: the same state never calls `SetIcon` twice.

### 2.3 App icon

- 1024×1024 canvas; the content sits inside the macOS 824 squircle safe area (100 pt margin all round), and the system draws the corners and Liquid Glass.
- Graphic: **the same fermata as the interface**, not an hourglass, not a clock. The mark sits centered in a 700×700 box, ink arc over a vermilion dot; because the arc is above the dot the optical center is lifted slightly above the box center.
- The name and the graphic share one origin: the English name **Fermata** is the mark itself, and the Chinese name is the same gesture — a finger stopping the string. No strings and no staff are drawn: at 16–32 px they read as a face, or flatten the mark into a background texture.
- Field: a vertical sand gradient, light `#F2EAD9` → `#E6DAC2`; dark `#2A241C` → `#161210`.
- Mark: ink `#1B1A14` (paper `#F3EEE2` in dark); dot `#C94B33` (dark `#F0836C`).
- The About page, welcome window, sidebar and menu bar template all use the same geometry (`theme.PaintMark` / `trayicon.Mark` / `internal/mark`), adjusting only the optical weight from the table above.
- **Delivery:** an Icon Composer `.icon` with two layers (sand field + mark) and light/dark specializations, compiled into `Assets.car`; older systems fall back to a pre-rounded `.icns`. No text, no drop shadow.
- Regenerate with `make icons` (needs `actool` from Xcode 26+), which also writes `resources/icon.png` and `resources/tray.png`.

### 2.4 Wordmark

The wordmark is the product name in the system UI font, Semibold, tracking −0.01em. English is `Fermata`; the Simplified Chinese localization sets the product's Chinese name in the same font and weight. It appears only in the welcome window, Settings › About and system notifications (as the app name). **The break window never shows the wordmark.**

## 3. Design tokens

### 3.1 Color

**Principle:** restrained in preferences and settings, immersive during a break, never color as the only signal, and no dynamic color read from the wallpaper.

#### Role tokens

| Token | Light | Dark | Use |
|---|---|---|---|
| `primary` | `#B8432F` | `#F0836C` | Primary button, selection, the mark's dot |
| `on-primary` | `#FFFFFF` | `#451009` | Text on an accent fill |
| `primary-container` | `#F6D9D2` | `#5C2418` | Secondary accent fill, hints |
| `secondary` | `#3E5F4E` | `#A9C8B4` | Quiet state, success meaning |
| `secondary-container` | `#D7E3D9` | `#2A3E32` | Rhythm chips, tonal buttons |
| `surface` | `#F6F4ED` | `#121513` | Window background |
| `surface-container` | `#EAE8DF` | `#1B211D` | Group background, input slots |
| `surface-high` | `#E2E0D7` | `#232A25` | Overlay background, selected row |
| `surface-sidebar` | `#F0EEE6` | `#171B18` | Settings sidebar |
| `surface-pressed` | `#DAD8CE` | `#2B322C` | Pressed controls |
| `on-surface` | `#171A17` | `#ECEDE5` | Body text |
| `on-surface-variant` | `#5D635B` | `#A8AEA4` | Descriptions, subtitles |
| `outline` | `#7C8178` | `#8E938B` | Input borders, icons |
| `outline-variant` | `#D6D4CA` | `#2E342F` | 1 px separators |
| `error` | `#B3261E` | `#F2B8B5` | Errors, strict notices |
| `warning` | `#8A5A00` | `#F0C36A` | Permissions, write failures |
| `focus` | `rgba(184,67,47,.55)` | `rgba(226,103,78,.60)` | 2 px focus ring |

Contrast: body and description text ≥ 4.5:1; large text and graphics ≥ 3:1; the focus ring ≥ 3:1 against its neighbors. Placeholder text may not rely on pale gray.

#### Break field

| Preset | Light | Dark | Long break deepens to | Foreground |
|---|---|---|---|---|
| Pine (default) | `#2B4A41` | `#24413A` | `#1B322C` | paper `#F3F2EA` |
| Ink | `#141A18` | `#141A18` | `#0E1311` | paper |
| Oxblood | `#4A2A2E` | `#43262A` | `#351D20` | paper |
| Slate | `#2E383F` | `#2A3339` | `#20282D` | paper |
| Sand (light) | `#CFC5B2` | `#CFC5B2` | `#C2B7A3` | ink `#171A17` |
| Paper (light) | `#EFEEE6` | `#EFEEE6` | `#E6E4DA` | ink |

- The foreground is chosen automatically from luminance, paper or ink; custom colors are treated the same way. A light field takes no vignette and no texture.
- The mark: vermilion `#F0836C` on a dark field, `#B8432F` on a light one.
- Progress track = foreground at 25%; used = foreground at 85%; secondary text = foreground at 62–72%.
- A long break does not change hue, only deepens one step; type is one step larger. The user picks one field color.

#### Color rules

- Strict never uses red; it is expressed by copy and by the absence of an exit. Strict is not an error.
- Errors use only `error`; permissions use `warning`; success uses `secondary` sparingly.
- No gradients, colored shadows or decorative color blocks; no texture on backgrounds.

### 3.2 Type

The UI uses the **system font**, so controls feel native; break suggestions use a **bundled serif**, so they are identical across platforms.

| Role | Font stack |
|---|---|
| macOS UI | System UI (SF Pro Text / Display); Chinese `PingFang SC` |
| Windows UI | System UI (Segoe UI Variable Text); Chinese `Microsoft YaHei UI` |
| Linux UI | System UI (Inter / Noto Sans / Cantarell); Chinese `Noto Sans CJK SC` |
| Break suggestion | `Spectral` (bundled 300/400), `Noto Serif SC` (bundled subset 300/400), `Iowan Old Style`, `Songti SC`, `SimSun`, `serif` |
| Numerals / countdown | `SF Mono`, `Cascadia Mono`, `JetBrains Mono`, `monospace`; or the system UI with `tnum` |

Type rules:

- UI fonts are not embedded; the break serif ships as an OFL subset: Spectral (Latin) and Noto Serif SC (covering every Han character in `locales/*.json` and `ideas/*.json`) under `internal/fonts/assets/`. **No font is downloaded at runtime.**
- The subset is rebuilt by `tools/fonts/subset.py`; a test in `internal/fonts` checks coverage character by character, so a new suggestion with a missing glyph fails the test rather than silently falling back to a system serif.
- Numerals are always tabular (`tnum` or monospace); time does not jitter.
- Body text is at least 11 px; `micro` (10/14) is only for version lines, footers and placeholder hints; the 8–10 px type inside **scaled previews** is a miniature, not UI text.
- Chinese suggestions are set +2 with a line height ×1.25; Chinese takes no negative tracking; Latin display type may use −0.01em.
- System text scaling (TextScale on Windows/GNOME) reaches all UI text through `theme.Px`, range 1.0–1.5; scaled previews are excluded.
- No fluid/clamp type, and no shrinking type to fit a window; when content does not fit, the window scrolls or the text wraps.

#### Type scale

| Token | Size / line height | Weight | Use |
|---|---|---|---|
| `display` | 34/40 | 600 | Welcome / cover-level titles |
| `title.lg` | 22/28 | 600, −0.01em | Settings page title |
| `brand` | 20/26 | 600, −0.01em | Welcome wordmark |
| `title.md` | 18/24 | 600 | Group titles |
| `heading` | 14/20 | 600 | Control groups, sidebar |
| `body` | 13/19 | 400 | Controls, body |
| `secondary` | 12/17 | 400 | Descriptions |
| `note` | 11/16 | 400 | Inline notes, hints |
| `caption` | 11/15 | 500, +0.14em | All-caps micro labels |
| `micro` | 10/14 | 500 | Versions, footers, placeholders |
| `action` | 13/20 | 500 | Buttons, exits |
| `countdown` | 13/16 | 400, +0.04em | Countdown |
| `break.mini` | 42/51 (38/46 on screens ≤1280; Chinese 44/55 or 40/50) | 300 | Mini break suggestion |
| `break.long.title` | 50/57 (46/53 on screens ≤1280; Chinese 52/65 or 48/60) | 300 | Long break title |
| `break.long.body` | 26/36 (Chinese 28/35), 72% foreground | 300 | Long break body |

### 3.3 Spacing and layout

- Base unit 4; layout prefers multiples of 8.
- Desktop density: controls 32 high; rows 44 (50–56 with a description); compact rows 38; key action hit areas ≥ 44×28.
- Settings window: one size, **800×720** (all four pages share the height); minimum 760×560; sidebar **208**; content padding 32; content maximum width 528; content scrolls instead of shrinking type.
- Welcome window: **800×540**; left preview 320, right content 480.
- Break panel: **85%×85%** of the screen bounds, centered; radius 20; full-bleed with radius 0 when `fillScreen`, the menu bar and the Dock included; padding 48 (56 at a screen edge); the exit sits 64 above the panel's bottom edge.
- Break panel text: a mini suggestion caps at 780, a long break title at 820 and its body at 680; the progress line is 280 (360 for a long break), 40 below the suggestion, and the countdown 16 below the line.
- Scaling: 100–150% text scaling changes no layout structure (rows grow, pages scroll); type is never shrunk to force a fit.

### 3.4 Shape

| Radius | Use |
|---|---|
| 0 | A break panel that fills the screen |
| 4 | Small labels, inline blocks |
| 8 | Controls, buttons, inputs, color swatches |
| 12 | Windows, menu containers, overlays |
| 16 | Preview containers |
| 20 | Break panel |
| full | Switch tracks, rings |

### 3.5 Elevation

| Level | Shadow | Use |
|---|---|---|
| 0 | none | Settings rows, lists |
| 1 | `0 1px 2px rgba(0,0,0,.06)` | Previews, raised swatches |
| 2 | `0 8px 24px rgba(0,0,0,.12)` | Menus, dialogs |
| 3 | `0 24px 80px rgba(0,0,0,.38)` | Break panel |

- Shadows are colorless; no ghost cards made of a 1 px border plus a large shadow.
- System menus and dialogs draw their own shadows.

### 3.6 State layers and focus

| State | Rule |
|---|---|
| Hover | 8% primary (or 8% of the current foreground), 120 ms; pointer devices only |
| Focus | 2 px focus ring + offset 2; keyboard focus only (`:focus-visible`) |
| Pressed | 12%, 80–120 ms; no translation, no scale |
| Drag | 16%, sliders and drag controls only |
| Disabled | Text 38%, container 12%; ignores events |
| Selected | `surface` fill + weight 600; or a 2 px accent ring (swatches) |

### 3.7 Motion

- Durations: `120` short / `180` short2 / `220` medium / `300` long.
- Curves: standard `cubic-bezier(.2,0,0,1)`; emphasized `cubic-bezier(.05,.7,.1,1)`; exit `cubic-bezier(.3,0,1,1)`.
- Break panel in: window opacity 0→1 over 180 ms; content translateY 6→0 over 220 ms. Exit is immediate; a display change rebuilds the windows without replaying the entrance.
- Progress line: 900 ms linear, advancing every second; the countdown steps once per second.
- Settings: only control-state transitions at 120 ms; no page transitions, no staged entrance.
- Reduce Motion: drop translation and fades; the progress line may still step.
- Forbidden: looping pulses, breathing scale, particles, flowing gradients, parallax.

## 4. Components

Every control is drawn by the app and consistent across platforms; system menus, title bars and dialogs are the exceptions. Every interactive control has default / hover / pressed / focus / disabled, and **never a loading state** (all operations are local).

### 4.1 Button

| Variant | Fill | Label | Use |
|---|---|---|---|
| Filled | `primary` | `on-primary` | The one primary action per screen (Start) |
| Tonal | `secondary-container` | `#1B2A21` | Secondary action (Open Settings) |
| Outlined | transparent + `outline` border | `on-surface` | Cancel, minor actions |
| Text | transparent | `primary` | Previews, low-frequency actions |

- 32 high; padding 0 14; radius 8; type 13/20 at weight 500.
- Filled hover `#A63A28`, pressed `#8F3223`; focus ring per §3.6.
- **The break window's exit is not a filled button:** it is a text button, 15/20 at 82% foreground, underlined on hover; hit area 44×28.

### 4.2 Switch

- 36×20 track with a 16 px dot; on = `primary`, off = `outline-variant`.
- The whole row is clickable; Space toggles; the focus ring sits on the track.
- Turning it off grays out or hides dependent controls (notification lead time, postpone count).

### 4.3 Segmented

- 32 high; track `#EAE8DF`, padding 2; the selected item is `surface` at weight 600 with a 1 px shadow.
- Used for the menu bar style (Mark / Time / Ring); arrow keys move between items.

### 4.4 Stepper

- 32 high; border `outline-variant`; a 30-wide button on each side; value area 46–52, 13 px tabular.
- v1 steps once per click; hold-to-repeat is deferred. The button at a bound is disabled.
- The value renders itself (`5 min` / `25 min`) and does not depend on input focus.
- The buttons draw a **12 px vector glyph** (max(1.4, 10% of the glyph box) line, round caps), never a `−`/`+` character, and carry the accessible names `Decrease` / `Increase`.

### 4.5 Select

- 32 high; 150–180 wide; border `outline`; radius 8; an 11 px chevron on the right.
- Opens a system or self-drawn list; keyboard ↑↓ / Enter / Esc.
- Never the platform default `<select>` appearance (the three platforms differ too much).

### 4.6 Slider

- 180–220 wide; 2 px track; thumb 14; fill `primary`.
- The value is 12 px tabular in a fixed-width slot on the right, so it does not jump while dragging.

### 4.7 Swatch

- 34×34 (28 in the canvas art); radius 8; selected: a 2 px `primary` ring with a 2 px gap.
- No text inside a swatch; the name sits 10 px below.
- `Custom…` opens the system color picker; choosing a color adds a seventh swatch.

### 4.8 Banner

- Radius 8; padding 11–12; a whole-block tint, **never a single side rule**.
- Warning: fill `#F7E8C8`, border `#E6CF9F`, text `#4A3600`; Info: fill `surface-container`, border `outline-variant`.
- One title line + one description line + one text action; no modal.
- Focus order: the banner action follows the body copy, and a screen reader reaches it.

### 4.9 Menu

- **Always native** (NSMenu on macOS, the native menu on Windows, DBusMenu on Linux). Not drawn by the app, not unified in appearance.
- The design only fixes items, grouping, the state matrix, accelerators and copy; see §5.2.

### 4.10 Icons

The product has one small-icon language, all drawn by the app (`internal/icons`, a 16-unit grid):

| Name | Use | Size |
|---|---|---|
| `chevron` | Select, expandable groups | 12–14, max(1.4, 10%) line |
| `check` | Selection, menus | 12–14 |
| `plus` / `minus` | Stepper | 12, 1.4 line, round caps |
| `close` | Popover close | 12 |
| `more` | Popover opening the native menu | 12 (three dots) |
| `pause` | Tray pause | 18 |
| `ring` | Tray progress | 18 |

- Stroke width follows the glyph box: `max(1.4, 10% of the box)`, so a 12 px glyph draws 1.4 pt and larger glyphs scale.
- Beyond the mark and this table, no icon language is introduced; the sidebar has no icons.
- No emoji and no Unicode characters (`×`, `⋯`, `−`, `+`) as substitutes for a glyph.
- Icon color inherits the text color (`t.Text` / `t.TextMuted`); the hit area is still provided by a 20–28 px button.
- A meaningful icon button must have a localized accessible name: `Close` / `Show menu` / `Decrease` / `Increase`.
- Decorative icons are hidden from assistive technology.

## 5. Surfaces

### 5.1 Break window

#### Geometry

- One window per target display; no title bar, no zoom or close button, and no entry in the taskbar, Dock or app switcher.
- 85%×85% centered by default; radius 20; on top of full-screen apps and every Space; showing it neither activates the app nor takes keyboard focus.
- The window swallows clicks that land on it (so the desktop is not clicked by accident); only the action responds.
- The window cannot be dragged or resized.
- `fillScreen`: the full screen bounds, radius 0, over the menu bar and the Dock; never kiosk mode.
- Multiple displays: one field per screen; `contentScreen` decides which screen carries content; the others show only the plain field — no text, no line, no button.

#### Content order

```
MINI BREAK                                             14:32
                         ⌒
                          •
          Look out a window until your eyes relax.
              ──────────────────────────────
                          0:14

                    Postpone  ⌘X
```

0. Mark: 48×48, 28 above the suggestion; optically heavy arc (2.6) and dot (2.9).
1. Top-left label: `MINI BREAK` / `LONG BREAK`; `STRICT MINI BREAK` / `STRICT LONG BREAK` when strict. 11 pt, all caps, +0.14em, 55% foreground.
2. Top-right current time (optional): follows the locale's 12/24-hour clock; 11 pt, 55% foreground.
3. Center suggestion: one sentence for a mini break, at most 4 lines, 42/51 (38/46 on screens ≤1280; Chinese +2, line height ×1.25), serif 300, centered; a long break sets a 50/57 title (46/53 on ≤1280; Chinese 52/65) plus a 26/36 body (Chinese 28/35, 72% foreground), 24 apart.
4. Progress line: 280×1 (360×1 for a long break); track at 25% foreground, used at 85%; 40 below the suggestion.
5. Countdown: 13/16 tabular numerals, 65% foreground, +0.04em; 16 below the progress line.
6. Exit: bottom center, 64 above the panel's bottom; 15/20 at 82% foreground; underline on hover; the shortcut hint is 12 mono at 55%, 8 apart.
7. Break health (P1): when `danger > 2`, a vermilion inset edge all round, changing no layout.

#### Four moments

| Moment | Exit | Countdown | Progress |
|---|---|---|---|
| First 30% | `Postpone` (when postponing is allowed) | Counts down | Used 0–30% |
| After 30% | `Skip` (not strict) | Counts down | Used 30–100% |
| Manual finish | `Finish` | Counts up as `+m:ss`, in vermilion | Track cleared |
| Strict | **none** | Counts down | Normal |

- At most one exit at any moment; a strict countdown shows no exit; strict plus manual finish shows only `Finish`. Strict shows no lock icon and is not a danger state.
- Exit shortcut: `⌘X` on macOS, `Ctrl+X` on Windows/Linux; during a strict countdown there is no action, no hint and no error.
- Closing a break window is triggered only by: countdown end, postpone, skip, manual finish, pause, reset, quit, or the system session ending.
- Multi-display changes rebuild the window group while keeping `startAt`, `duration`, elapsed time and the action state; the timer is not restarted.

#### Copy and type

- A suggestion is a complete sentence: capitalized, with a full stop at the end (a Chinese full stop in Chinese).
- Each is ≤ 60 English characters / ≤ 24 Han characters; it gives one action.
- Overlong text: type is not stepped down per item; below 1280 window width the whole block drops one step to 38/46 and is then ellipsized at 4 lines (2+3 for a long break). No scrolling, no window scaling.
- An empty suggestion keeps the vertical rhythm and shows no blank line.
- Light field: ink foreground; track at 20% ink, used at 85%; the exit underlines in ink.
- The break window never shows the app name, the wordmark, or marketing copy.

### 5.2 Menu bar / tray and menus

#### Status copy

Follows functional spec §8.1; in short:

- Waiting: `Next mini break in about {n} minutes`; with a long break ahead, a second line `Long break after {n} more mini breaks`.
- Notice: `Mini break in {n} seconds` / `Long break in {n} seconds`.
- Breaking: `Mini break — {m:ss} left` / `Long break — {m:ss} left`.
- Manual finish: `Break over — waiting for you`.
- Paused: `Paused — resuming in about {n} minutes` / `Paused — indefinitely` / `Paused — Focus is on` / `Paused — you were away` / `Paused — the screen is locked` / `Paused — app exclusion rule`.
- Held: `Break held — fullscreen app` (the moment arrived but its condition has not cleared; the planner is still waiting, so this is not shown as a pause). A typing hold is silent: it lasts at most 30 seconds and keeps the ordinary countdown.
- Strict: `Strict mini break — {m:ss} left` / `Strict long break — {m:ss} left`.
- Tooltip: first line `Fermata — A held note in the workday`, a blank line, then the status.

#### Menu matrix

| State | Status line | Skip to | Pause | Resume | Reset | Settings | Quit |
|---|---|---|---|---|---|---|---|
| Waiting / notice | yes | yes | yes | — | yes | yes | yes |
| Breaking (not strict) | yes | yes | yes | — | yes | yes | yes |
| Manual finish (not strict) | yes | yes | yes | — | yes | yes | yes |
| Manually paused | yes | — | — | yes | — | yes | yes |
| DND / idle / locked / excluded | yes | — | — | — | — | yes | yes |
| Strict break | yes | — | — | — | — | — | — |

- Menu items are **hidden** with the state, not grayed; the status line is disabled.
- `Skip to ›`: `Mini break` / `Long break`, listing only enabled kinds.
- `Pause breaks ›`: 30 minutes, 1 hour, 2 hours, 5 hours, until morning, indefinitely.
- English labels: `Settings…` (with the ellipsis on macOS; `Settings` on Windows/Linux), `Quit Fermata` (macOS/Linux) / `Exit Fermata` (Windows).
- Copy is sentence case; Chinese uses the informal second person only.
- During a strict break the menu keeps only the status line by default; when `general.showTrayMenuInStrictMode` is true, Settings stays and Quit is still hidden and refused by `OnBeforeQuit`. Ending the system session still quits.
- The system draws the menu (no second, self-drawn menu per platform); the status bar menu offers no About, since About lives in the settings sidebar, and the macOS application menu keeps only Hide, Edit, Window and Quit.

#### Status popover

- A left click on the tray icon opens `mygo.NewFlyout`: `Width: 320` (`MinWidth`/`MaxWidth` both 320), `FitContent: true` (`MaxHeight: 560`), `Gap: 4`, `Focusable: false`, `Content: ui.View(a.popoverView)`.
- **The shell belongs to the system on macOS:** with `Popover: true`, NSPopover draws the material, corners, shadow, the arrow pointing at the tray and the open/close animation, and the content paints no fill and no border (`c.Vibrancy()` is true). The window is created with `Shadow: true` and `CornerRadius: 14` on every platform; on Windows and Linux v1 paints its own paper-and-ink fill. The Linux tray reports no clicks, so v1 offers no popover entry there and keeps the menu only.
- **Height is not hard-coded:** waiting / notice / breaking / paused / strict are sized by their content (about 296 / 336 / 207 / 236 / 147); when action rows come and go, FitContent catches the change on the next frame, and no height is written in code. The panel hugs its content; no `Spacer` stretches it.
- Skeleton (`ui.Column`, padding `t.Space(4)`, blocks `t.Space(3)` apart): the mark at 20 (`theme.PaintMark`, drawn) + the state word (10/600, letter spacing .16em, uppercase) on one row with two 22 px icon buttons at the top right (`ui.ButtonBase` + a 12 px drawn glyph + `Label`/`Tooltip`, `Show menu` then `Close`); the remaining time (`theme.Px(c,34)` mono, weight 600, `ContentTransition{Numeric, Down}`; `21m` while waiting, `8s` during the notice, `12:04` while breaking or paused); the **ruler** (12 high, 60 hairlines, every fifth one long, the passed span in vermilion, a 2×10 cursor at the leading edge; drawn in `Draw`, `Role(RoleProgress)` + `Label("34%")` for screen readers); an 11/15 detail line with `MaxLines(2)`. The ruler and the tray icon's Ring share the same `progressShare`.
- Actions: `ui.PrimaryButton` + `ui.Button` (secondary: surface + 1 px border), then, after a `ui.Divider`, a `Settings…` text row (20 high, underlined on hover). **The primary button needs `t.InactiveAccent = t.Accent`** (`theme.ApplyFlyout`): `PrimaryButton` only fills with the accent in a focused window, and a flyout usually is not one, so by default it would collapse into a plain button identical to the secondary one. A strict break has no actions and no settings row.
- State words: `Next mini break` / `Next long break` / `Mini break · starting` / `Mini break · in progress` / `Paused` / `Strict · Mini break`, composed from `popoverNextMini`, `popoverNextLong`, `popoverMini`, `popoverLong`, `popoverStarting`, `popoverInProgress`, `popoverPaused` and `popoverStrict`. The detail line reuses the menu's second line (`Long break after 4 more mini breaks`) or the break's suggestion (`lastIdea.Text`); a pause does not repeat its reason — the menu and the tooltip carry it.
- No "end now" button during a break: the exit belongs to the break window, and the popover only reports the remaining time and the suggestion.
- Motion belongs to MyGo: the digits roll through `ContentTransition` (350 ms spring); the action block resizes through `Transition` (200 ms ease-out, the default); the frame, the opening and the final height belong to NSPopover and FitContent; values change once per second with no tweening; `ReduceMotion` degrades to a fade.
- Accessibility: the digits are not a live region; the ruler and both icon buttons each carry a `Label`; on macOS NSPopover takes the keyboard (Tab, Return), on Windows/Linux the mouse only and the keyboard goes through the menu. Accepted in `internal/popover` (`ui.NewTester`: the state text, countdown and actions, the primary action running alone, header button names, strict without actions or footer, the ruler at 34%, `tt.SetDark`) and in `internal/app` (`remainingLabel`, `popoverState`, `popoverActions`).
- It closes on a second left click, on the top-right close button, on running an action, or when a break starts; `more` calls `tray.ShowMenu()` (the fallback where a right click cannot reach the icon); both buttons draw a 12 px glyph, never `×`/`⋯`.

### 5.3 Settings

#### Window and shell

- macOS: hidden title bar with the traffic lights (left); the sidebar runs to the top; content starts 54 below.
- Windows 11: the system title bar (32 px) with radius 11; content starts 24 below it; closing the window is not quitting.
- Linux: GTK/KDE decoration (about a 40 px header bar); content shifts down by the decoration height.
- Content is identical on all three; sidebar 208, content padding 32, content maximum width 528.
- The title bar is hidden; the top 32 pt is a drag region (`DragWindow`) that also maximizes on double click. The window can still be dragged by that strip while the content is scrolled.

#### Sidebar

- Top: a 22 px mark + the wordmark at 13 semibold, 44 from the top and 20 from the left; the mark uses an ink arc and a vermilion dot with no plate behind it.
- Items: General / Schedule / Appearance / About; row height 32; the selected row is `surface` + weight 600; no icons.
- Keyboard: Tab enters the content, Shift+Tab returns to the sidebar; arrow keys move between items.

#### General

| Row | Control | Platform difference |
|---|---|---|
| Start at login | Switch | — |
| Language | Select: System / English / Chinese | Applies immediately |
| Menu bar / Tray icon | Segmented: Mark / Time / Ring | `Time` needs a host that supports a title: available on macOS; depends on the tray host on Linux (AppIndicator label); on Windows only Mark/Ring, with the time in the tooltip |
| Pause when away | Switch + threshold Stepper (5/10/15/30 minutes) | — |
| Pause during Focus | Switch | macOS only in v1, behind a permission-free read; Windows Focus Assist and Linux (hidden) are planned with their platform work |
| Pause when the screen is locked | Switch | — |
| Wait while typing | Switch | Off by default; when enabled, if the moment arrives while typing, wait up to 30 seconds |
| Wait for fullscreen apps | Switch | macOS only in v1: wait up to 1 hour while a full-screen app is in front; the notice is held too; detection reads window bounds only and needs no screen-recording permission |
| Show on all displays | Switch | — |
| Content on | Select: All displays / Primary / Cursor / N | Shown only while all displays is on; with it off, the same row is labeled `Screen` and offers Primary / Cursor / N |
| Fill the screen | Switch | — |
| App exclusions | Group: one description line + a right-aligned `Add rule` | Per rule: a name, processes (comma separated), `Pause` / `Break only`, a switch; empty state `No rules yet.`; a rule without a name or processes is not written |

- No cards; a full-width row plus a 1 px separator; 32 between groups. Theme moved to Appearance, so the General page has no Theme row.
- Controls are right-aligned, labels left-aligned; a description sits 11/16 under the label and is never truncated.

#### Schedule

The page uses a **two-column** layout (Mini breaks / Long breaks) so the two rhythms can be compared side by side; content width 528, two columns of 252 with 24 between.

| Column | Rows |
|---|---|
| Mini breaks | Break for `5 min`; Every `25 min`; Notify before `10 s`; Allow postpone `2 min · once`; Wait for me to finish |
| Long breaks | Break for `15 min`; Every `4 mini breaks`; Notify before `30 s`; Allow postpone `5 min · once`; Wait for me to finish |

- A Switch sits at the right of each column header; turning a column off grays its rows out (it does not hide them, so it can be turned back on).
- The notify lead and the postpone length are shown as descriptions under their labels; v1 makes only the switches editable.
- Strict mode: a full-width group; two switches, `Mini` / `Long`; description `Strict breaks cannot be skipped, postponed, or closed.`
- Summary line: `5 min every 25 min · 15 min every 5th break`, with the rhythm chips on the right: `MINI 5m · MINI 5m · MINI 5m · MINI 5m · LONG 15m`; computed live, with compact units inside the chips.
- When notifications are off the lead time grays out; when strict is on the postpone switch grays out and shows `Strict mode cannot postpone.` When both kinds are turned off, the last one is refused and a danger line appears under the row: `Keep at least one break type on.` No dialog.
- When only long breaks are on, `Every` becomes a minute Stepper (10–120 minutes, default 100), and the summary reads `15 min every 100 min`.
- The `Break health` group sits below strict mode: one switch row plus `A vermilion edge when breaks are often postponed or skipped.`

#### Appearance

- Theme: Segmented System / Light / Dark; it affects app windows only, while the menu bar and Finder icons follow the system.
- Break window opacity: Slider 30–100% with the percentage on the right; the preview checkerboard shows the desktop coming through.
- Preview: 528×216, a miniature of the break panel (not a re-layout; type and mark are enlarged slightly for legibility, at about 0.43×); it updates live with the field, the time and the opacity; type inside the preview is fixed and does not follow system text scaling.
- Break field: six 34×34 swatches + names; a 2 px ring when selected; `Custom…` opens the color picker.
- Show the current time: Switch, for the break panel's top-right clock.
- Sounds: two Selects, Start / End, plus the text action `Preview`; a Volume Slider with a percentage; Mute all sounds. The start sound defaults to Silence and the end sound to Crystal glass; volume 0–100%, grayed when muted.

#### About

- 72×72 app icon; the wordmark at 22/28 600; `A held note in the workday.` at 13/20 serif in the secondary color.
- Version `Version 0.1.0 (build 1)` at 10 mono; `Inspired by Stretchly. Fermata is MIT licensed.`
- Footer `© 2026 gythialy · No account, no telemetry, no sync.` at 10/15.
- Links: Website · Source code · Report an issue (accent, underlined on hover).
- `Restore Defaults…` is a text button (danger) below a separator; it opens a system confirmation dialog.

#### Validation and feedback

- Save immediately; there is no Save button; success is silent.
- Failure: a banner (write failure / corrupted configuration); in-memory settings keep applying; the app never blocks.
- Permissions: a missing Focus permission shows a warning banner + `Open System Settings`, once, never as a repeat dialog.
- Errors: 12 px danger text under the control; no shake, no toast.
- Focus order: sidebar → content top to bottom; Tab enters the content, Shift+Tab returns to the sidebar; arrow keys switch Segmented; Space toggles a Switch.

### 5.4 Welcome window

- 800×540; a 320 real break-panel thumbnail on the left (cropped tall), 480 of content on the right.
- Content: the mark (30 px) + the wordmark; `A held note in the workday.`; the default rhythm in two lines (`5 min · every 25 min`; `15 min · every 5th break`); Language; Start at login; the `Start` primary button; the `Open Settings` text button; the closing hint `Closing this window starts Fermata too.`
- Closing the window is the same as Start; while it is open, nothing is scheduled, no notification is sent and no break window is created.
- It does not turn on the login item by itself and does not request notification permission.
- Platform copy: macOS `Lives in the menu bar. Quit from its menu. No account, no telemetry.`; Windows/Linux `Lives in the system tray. …` (two catalog keys chosen by `runtime.GOOS`); with no tray, the copy explains that the status window opens from the application menu.
- No pages, no carousel, no Next; one decision, with 34 of padding at the bottom so the footer line does not touch the window edge.

### 5.5 Notifications and system feedback

- Notification title `Fermata`; body `Mini break in {n} seconds.` / `Long break in {n} seconds.`; no action buttons; a click opens the status popover without changing the schedule; a notice still on screen is closed when the break starts or the app pauses.
- Permission denied: never blocking, never a repeat prompt; the menu keeps working.
- Resume notification: `Fermata is resuming.` only for a manual or timed pause ending; none for DND / idle / locked resumes.
- Tray overflow (Windows, not yet implemented): the one-time first-run hint is deferred with platform adaptation, and no dead keys are kept in the v1 catalog. A corrupted configuration is backed up to `.corrupt` with defaults restored, and the settings page shows a banner once.
- All system feedback is text and one action; no modals.

## 6. Cross-platform adaptation

| Facet | macOS 13+ | Windows 11 | Linux (GNOME/KDE) |
|---|---|---|---|
| Entry point | Menu bar extra; no Dock icon | System tray + overflow; no persistent main window | StatusNotifierItem; GNOME needs the AppIndicator extension |
| Menu | Native; `⌘` accelerators | Native; `Ctrl` accelerators | Native DBusMenu; `Ctrl` accelerators |
| Icon | 18 pt template; may carry a `9m` title | 16–24 px monochrome; no title (`SetTitle` no-op) | 22 px symbolic; AppIndicator may carry a label (`MonospacedTitle` no-op) |
| Window | Hidden title bar + traffic lights; the break panel is borderless | System title bar + radius 11; break panel on top, no taskbar entry | GTK/KDE decoration; break panel on top / layer-shell |
| Title bar height | 28 pt | 32 px | 40 px |
| Notifications | Notification Center; only for a packaged app | Toast + Action Center | libnotify / desktop notifications |
| Fonts | SF Pro + PingFang SC | Segoe UI Variable + Microsoft YaHei | Inter/Noto Sans/Cantarell + Noto Sans CJK; on all three the break suggestion is the bundled Spectral + Noto Serif SC |
| Shortcuts | `⌘X` · `⌘,` · `⌘Q` | `Ctrl+X` · `Ctrl+,` (quit from the tray) | `Ctrl+X` · `Ctrl+,` · `Ctrl+Q` |
| Scaling | 2× Retina; 1× external display | 100/125/150%, text scale up to 150% | 100–200%, fractional scaling; text scale up to 150% |
| No tray | — | The overflow hint appears once (deferred with platform adaptation) | With no extension, the status window opens with the same menu content |
| Virtual desktop / focus | All Spaces; the panel does not activate the app and keeps the keyboard | The current virtual desktop; the panel is a front tool window and needs a global shortcut for the keyboard | X11 sticky; Wayland layer-shell output; layer shell takes exclusive keyboard |
| DND / Focus | Focus; reading the assertion needs full disk access | Focus Assist / DND; MyGo has no unified API, so it needs a platform implementation | No unified API; per-desktop implementations, hidden when missing |
| Launch at login | SMAppService / login items | Startup tasks / the Run registry key | `~/.config/autostart/*.desktop` |
| Multiple displays | One NSPanel per screen; the content screen is selectable | One top-most window per display; no taskbar entry; does not cross virtual desktops | One window per output; the content screen is selectable |
| Quit semantics | `⌘Q`; hidden while strict | Tray Exit; closing Settings is not quitting | Tray Quit; closing Settings is not quitting |

### 6.1 Rendering assumptions

- **MyGo's native UI is drawn by MyGo itself on macOS / Windows / Linux**; Fermata uses no webview frontend.
- Content is rendered by **the same set of drawn controls** (32 px, radius 8, state layers), so the three platforms look the same.
- Window decoration, menus, notifications, login items and file pickers belong to the system and are not drawn by the app.
- Design sizes are expressed in **dp / CSS px @1×**; high DPI is scaled by the renderer, never hard-coded in pixels.

### 6.2 Implementation boundary

- MyGo 0.10.0 supports macOS, Windows and Linux (plus iOS/Android); one codebase builds all three, with no second renderer.
- Platform differences live in **API semantics**, not in rendering: no tray title on Windows, a panel only on the current virtual desktop, Linux titles depending on the host, Wayland needing layer-shell, macOS notifications only for a packaged app.
- Design tokens and platform service interfaces (tray / notification / idle / DND / login / display) stay decoupled for testing and replacement; the default implementation is MyGo.
- MyGo's font metrics may differ by ±1 px of line height across the three platforms; that error is accepted, and it changes neither hierarchy, spacing rhythm nor hit areas.

### 6.3 Deferred

- Windows/Linux build artifacts, installers and on-device acceptance (same MyGo codebase).
- Real API verification for Windows Focus Assist / Linux DND.
- Wayland layer-shell multi-display and full-screen behavior, and the status-window details for GNOME without AppIndicator.
- Installers, signing and automatic updates.

## 7. Accessibility

### 7.1 Checklist

- Body and description text ≥ 4.5:1; large text and graphics ≥ 3:1; re-checked on light break fields.
- Focus ring 2 px with offset 2, keyboard focus only; never `outline: none`.
- Fully keyboard reachable: Tab/Shift+Tab, arrow keys for Segmented, Space for Switch, Esc closes overlays.
- Screen readers get role/name/state; a break starting is announced by the system notification, not by the window itself.
- Reduce Motion: drop translation and fades; the progress line may step.
- Increase Contrast: hairlines darken and the focus ring thickens to 3 px. Reduce Transparency: no blur; the break panel stays opaque.
- 150% text is not truncated; rows grow with the content (`theme.Px` caps at 1.5; previews are the exception).
- Controls ≥ 32×32; the break exit is 44×28.
- State never relies on color: strict, paused and errors all carry words.
- EN / zh-Hans reserve 1.3× width; button labels do not wrap.
- The break window does not take focus; `⌘X` / `Ctrl+X` and the mouse both work.

### 7.2 Platform acceptance

- macOS: VoiceOver walks Settings and the break window; Reduce Motion/Transparency; 1×/2×; the icon reads on both light and dark menu bars.
- Windows: Narrator / UIA names; High Contrast; 100/125/150/200%; the tray overflow and the 16 px icon are legible.
- Linux: Orca reads menus and Settings; GTK font scaling; fractional scaling; the status window when GNOME has no extension.

### 7.3 Known limitations

- Notification authorization cannot be queried; Focus is enabled only where it is readable without extra access (not readable by default on macOS 13+, and the app does not request full disk access); strict mode has no in-app emergency exit (ending the system session still ends it); Wayland multi-display behavior depends on the compositor.

## 8. Copy and voice

### 8.1 Tone

- Quiet, specific, actionable; not motivational, not diagnostic, not commanding.
- English is sentence case; Chinese uses the informal second person only.
- A suggestion is an action, not a slogan: `Stand up and let your shoulders drop.`, not `Take a mindful pause.`
- Numbers and units: `10 minutes` in English. No empty filler ("take a break"), no citations of research, no exclamation marks.

### 8.2 Key strings

The source of truth is `locales/en.json`; the keys below are the catalog's own. `{n}`, `{time}` and friends are placeholders, plural forms are in the catalog (only the plural-other form is shown here), and two further keys, `settingsPreview` and `settingsRuleProcesses`, are catalogued without a view reading them in v1.

**App, tray and menu**

| Key | String | Where |
|---|---|---|
| `appName` | Fermata | Window titles, notification title, About |
| `appTagline` | A held note in the workday. | Welcome, About |
| `trayTooltip` | Fermata — A held note in the workday | Tooltip, first line |
| `menuNextMini` / `menuNextLong` | Next mini break in about {n} minutes / Next long break in about {n} minutes | Menu status, waiting |
| `menuAfterMini` | Long break after {n} more mini breaks | Menu status, second line |
| `menuSoonMini` / `menuSoonLong` | Mini break any moment now / Long break any moment now | Status inside the last second |
| `menuNotifiedMini` / `menuNotifiedLong` | Mini break in {n} seconds / Long break in {n} seconds | Status during the notice |
| `menuBreakingMini` / `menuBreakingLong` | Mini break — {time} left / Long break — {time} left | Status during a break |
| `menuStrictMini` / `menuStrictLong` | Strict mini break — {time} left / Strict long break — {time} left | Status during a strict break |
| `menuBreakOver` | Break over — waiting for you | Manual finish |
| `menuPausedTimed` | Paused — resuming in about {n} minutes | Timed pause |
| `menuPausedForever` | Paused — indefinitely | Indefinite pause |
| `menuPausedDnd` / `menuPausedIdle` / `menuPausedLocked` / `menuPausedExclusion` | Paused — Focus is on / Paused — you were away / Paused — the screen is locked / Paused — app exclusion rule | Automatic pauses |
| `menuPreparing` | Getting ready | First moments after launch |
| `menuHeldFullscreen` | Break held — fullscreen app | Fullscreen hold at a due break (a typing hold stays silent) |
| `menuSkipTo` | Skip to | Submenu parent |
| `menuMiniBreak` / `menuLongBreak` | Mini break / Long break | `Skip to` submenu |
| `menuPause` / `menuResume` / `menuReset` | Pause breaks / Resume breaks / Reset breaks | Menu commands |
| `menuPause30` … `menuPause5h` | 30 minutes / 1 hour / 2 hours / 5 hours | Pause submenu |
| `menuPauseMorning` / `menuPauseIndefinitely` | Until morning / Indefinitely | Pause submenu |
| `menuSettingsEllipsis` / `menuSettings` | Settings… / Settings | Menu; the ellipsis form is macOS |
| `menuQuit` / `menuExit` | Quit Fermata / Exit Fermata | Menu; Exit is Windows |

**Break window**

| Key | String | Where |
|---|---|---|
| `breakLabelMini` / `breakLabelLong` / `breakLabelStrictMini` / `breakLabelStrictLong` | MINI BREAK / LONG BREAK / STRICT MINI BREAK / STRICT LONG BREAK | Top-left label, strict variants when strict |
| `breakPostpone` / `breakSkip` / `breakFinish` | Postpone / Skip / Finish | The exits: first 30%, after 30%, manual finish |
| `actionDecrease` / `actionIncrease` / `actionClose` / `actionShowMenu` | Decrease / Increase / Close / Show menu | Accessible names of icon buttons |

**Status popover**

| Key | String | Where |
|---|---|---|
| `popoverNextMini` / `popoverNextLong` | Next mini break / Next long break | State word |
| `popoverMini` / `popoverLong` | Mini break / Long break | State word composite |
| `popoverStarting` / `popoverInProgress` | starting / in progress | `Mini break · starting`, `Mini break · in progress` |
| `popoverStrict` / `popoverPaused` | Strict / Paused | State word |
| `popoverTakeBreak` / `popoverStartNow` | Take a break now / Start now | Primary action |
| `popoverPush5` | Push 5 minutes | Notice action |
| `popoverPause` | Pause 30 minutes | Secondary action |
| `popoverSettings` | Settings… | Footer text row |

**Settings**

| Key | String | Where |
|---|---|---|
| `settingsTitle` | Settings | Window title |
| `settingsGeneral` / `settingsSchedule` / `settingsAppearance` / `settingsAbout` | General / Schedule / Appearance / About | Sidebar, page titles |
| `settingsGeneralSubtitle` / `settingsScheduleSubtitle` / `settingsAppearanceSubtitle` | When Fermata goes quiet. / Set the rhythm. Keep at least one break type on. / Choose the light you rest in. | The three page subtitles |
| `settingsOpenAtLogin` | Start at login | General row |
| `settingsLanguage` | Language | General row |
| `languageSystem` / `languageEnglish` / `languageChineseSimplified` | System / English / 简体中文 | Language select; the endonym ships verbatim in the English catalog |
| `settingsMenuBar` / `settingsTrayIcon` | Menu bar / Tray icon | Row label; Tray icon is the Windows label |
| `styleMark` / `styleTime` / `styleRing` | Mark / Time / Ring | Segmented options |
| `settingsPauseAway` + `settingsAfterMinutes` | Pause when away — after {n} minutes | General row and its indented threshold |
| `settingsPauseFocus` | Pause during Focus | General row |
| `settingsPauseLock` | Pause when the screen is locked | General row |
| `settingsWaitTyping` + `settingsWaitTypingHelp` | Wait while typing — Waits up to 30 seconds while you type. | General row and description |
| `settingsWaitFullscreen` + `settingsWaitFullscreenHelp` | Wait for fullscreen apps — Waits up to 1 hour while an app is fullscreen. | General row and description |
| `settingsAllDisplays` | Show on all displays | General row |
| `settingsContentOn` / `settingsScreen` | Content on / Screen | Display select label |
| `displayAll` / `displayPrimary` / `displayCursor` / `displayNumbered` | All displays / Primary display / Display with the pointer / Display {n} | Display select options |
| `settingsFillScreen` | Fill the screen | General row |
| `settingsMiniBreaks` / `settingsLongBreaks` | Mini breaks / Long breaks | Schedule column headers |
| `settingsBreakFor` / `settingsEvery` / `settingsEveryMiniBreaks` | Break for / Every / mini breaks | Schedule rows; the last is the description under `Every` |
| `settingsNotifyBefore` + `durationSeconds` / `settingsAllowPostpone` + `settingsPostponeOnce` / `settingsWaitFinish` + `settingsManualFinish` | Notify before — {n} seconds / Allow postpone — {n} min · once / Wait for me to finish — until you press Finish | The three schedule rows and their descriptions |
| `settingsStrictMode` + `settingsStrictHelp` | Strict mode — Strict breaks cannot be skipped, postponed, or closed. | Schedule group and description |
| `settingsKeepOne` | Keep at least one break type on. | Danger line, both kinds off |
| `settingsStrictCannotPostpone` | Strict mode cannot postpone. | Postpone row while strict is on |
| `settingsSummary` | {mini} every {interval} · {long} every {nth} break | Summary line |
| `settingsSummaryMiniOnly` / `settingsSummaryLongOnly` | {duration} every {interval} | Summary with one kind |
| `rhythmMini` / `rhythmLong` | MINI / LONG | Rhythm chips |
| `unitMinutes` / `unitSeconds` / `unitCount` | {n} min / {n} s / {n} | Steppers |
| `unitMinutesShort` / `unitSecondsShort` | {n}m / {n}s | Rhythm chips |
| `ordinal` | {n}st / nd / rd / th | `every 5th break` |
| `settingsHealth` + `settingsHealthHelp` | Break health — A vermilion edge when breaks are often postponed or skipped. | Schedule group and description |
| `settingsExclusions` + `settingsExclusionsHelp` | App exclusions — Pause breaks while a listed app runs. | General group and description |
| `settingsExclusionsEmpty` | No rules yet. | Empty state |
| `settingsAddRule` / `settingsRuleRemove` | Add rule / Remove | Text actions |
| `settingsRulePause` / `settingsRuleResume` | Pause / Break only | Rule Segmented |
| `settingsRuleName` / `settingsRuleProcesses` / `settingsRuleProcessesHint` | Name / Processes / Comma-separated names, e.g. Xcode, zoom.us | Rule field labels; the hint is the process placeholder |
| `settingsRuleActive` | Enabled | Rule switch label |
| `settingsTheme` + `settingsThemeHelp` | Theme — Follow the system, or force light or dark. | Appearance row and description |
| `themeSystem` / `themeLight` / `themeDark` | System / Light / Dark | Theme Segmented |
| `settingsOpacity` + `settingsOpacityHelp` | Break window opacity — 30–100% of the field's color over the desktop. | Appearance row and description |
| `settingsPreviewScale` | Preview · ~0.43× | Caption under the preview |
| `settingsBreakField` | Break field | Appearance group |
| `fieldPine` … `fieldPaper` | Pine / Ink / Oxblood / Slate / Sand / Paper | Swatch names |
| `settingsCustom` | Custom… | The seventh swatch |
| `settingsShowTime` | Show the current time | Appearance row |
| `settingsSounds` | Sounds | Appearance group |
| `settingsStartSound` / `settingsEndSound` / `settingsPreviewSound` | Start / End / Preview | Sound selects and the preview action |
| `settingsVolume` / `settingsMute` | Volume / Mute all sounds | Appearance rows |
| `soundSilence` … `soundReverie` | Silence / Crystal glass / Wind chime / Tic-toc / Reverie | Sound select options |

**Welcome, About, dialogs and banners**

| Key | String | Where |
|---|---|---|
| `welcomeStart` / `welcomeOpenSettings` / `welcomeClosingHint` | Start / Open Settings / Closing this window starts Fermata too. | Welcome actions and footer |
| `welcomeLivesInMenuBar` / `welcomeLivesInTray` | Lives in the menu bar. Quit from its menu. No account, no telemetry. / Lives in the system tray. Quit from its menu. No account, no telemetry. | Welcome footer, chosen by `runtime.GOOS` |
| `welcomeMiniBreak` / `welcomeLongBreak` | Mini break / Long break | Welcome rhythm rows |
| `welcomeEvery` / `welcomeEveryNth` | every {interval} / every {n} break | Welcome rhythm values |
| `aboutVersion` | Version {version} (build {build}) | About |
| `aboutInspired` | Inspired by Stretchly. Fermata is MIT licensed. | About |
| `aboutWebsite` / `aboutSource` / `aboutReport` | Website / Source code / Report an issue | About links |
| `aboutFooter` | © 2026 gythialy · No account, no telemetry, no sync. | About footer |
| `aboutUpdateAvailable` / `aboutReleaseNotes` | Version {version} is available. / Release notes | About update line |
| `settingsRestore` | Restore Defaults… | About text button |
| `dialogRestoreTitle` / `dialogRestoreBody` | Restore defaults? / All settings return to their initial values. This cannot be undone. | Confirmation dialog |
| `dialogCancel` / `dialogRestore` | Cancel / Restore | Confirmation buttons |
| `bannerWriteFailed` / `bannerConfigCorrupt` | Settings could not be saved. / Settings were reset after a read error. | Banners |
| `notifyMini` / `notifyLong` | Mini break in {n} seconds. / Long break in {n} seconds. | Notification body |
| `notifyResumed` | Fermata is resuming. | Notification after a pause |

### 8.3 Break suggestions

The shipped decks hold 60 mini and 36 long suggestions, in `internal/ideas/assets/en.json` and `zh-Hans.json`. Examples:

Mini breaks:

- Look out a window until your eyes relax.
- Stand up and let your shoulders drop.
- Rest your eyes on something far away.
- Drink a glass of water.
- Unclench your jaw and take one slow breath.
- Roll your wrists and let your hands hang loose.

Long breaks (title + body):

- Step away — Walk to a window and look at something far away.
- Move a little — Take a short walk, even if it is just down the hall.
- Reset your posture — Stand up, roll your shoulders back, and breathe out slowly.

Writing rules: each ≤ 60 English characters / ≤ 24 Han characters (the longest shipped mini sentence is 52 characters and the longest long-break body is 60); one action; no "should", "must" or "health"; no research citations.

## 9. Implementation mapping (MyGo 0.10.0 / desktop)

MyGo 0.10.0 provides the same window, tray, menu, notification, power, login-item, multi-display and native-UI APIs on macOS, Windows and Linux; the table focuses on the v1 macOS acceptance, and §6 covers the platform differences.

| Design object | MyGo |
|---|---|
| Theme | `c.SetTheme(&customTheme)` on top of `ui.LightTheme()` / `ui.DarkTheme()`, overriding Accent, Background, Surface, Border, Text, TextMuted, Focus; `Radius = 8`, `Spacing = 4`, `FontSize = 13` |
| Break panel window | `mygo.NewWindow(WindowOptions{Panel: true, AlwaysOnTop: true, Width, Height, X, Y, Content, BackgroundColor})`; `ShowInactive()`; `DisableClose` for strict |
| Multiple displays | `mygo.Screen.Displays()` for `Bounds`; one window per screen; `OnDisplaysChanged` rebuilds it and keeps the timer |
| Break suggestion | `ui.Text(...).Font(theme.Serif)`; size through `theme.Px(c, 42)`; Chinese is drawn from the bundled Noto Serif SC subset |
| Mark | The one geometry in `internal/mark`; `ui.Box(...).Draw(theme.PaintMark)`; the app icon and the tray take their shape from it too |
| Small icons | Paths from `internal/icons` (chevron / check / plus / minus / close / more); no Unicode glyphs |
| Tray | `mygo.NewTray(TrayOptions{Icon, IconIsTemplate, Title, MonospacedTitle, Menu, ToolTip})`; `SetTitle` is a no-op on Windows, the Linux label depends on the host, `MonospacedTitle` is macOS-only |
| Menu | `mygo.NewMenu` + `MenuItem`; the status line is disabled; `SetLabel` updates in place |
| Settings window | macOS `WindowOptions{TitleBarStyle: TitleBarHidden}`; Windows/Linux use the system title bar with a content offset; a hand-drawn 208 sidebar (`ButtonBase` rows) beside a scrolling content column; controls `ui.Switch`, `ui.Select`, `ui.Segmented`, `ui.Slider`, `ui.ColorWell`, `ui.TextInputBase`, plus the schedule page's own stepper and text-button helpers |
| Confirmation | `mygo.Dialog.Message(MessageOptions{Type: Question, Buttons: [...]})` |
| Notifications | `mygo.NewNotification(NotificationOptions{Title, Body, Silent}).Show()`; `NotificationsSupported()`; a denial returns `ErrNotificationsDenied`; macOS needs a packaged app |
| Global shortcut | `mygo.GlobalShortcut.Register("CmdOrCtrl+X", fn)` |
| Open a URL | `mygo.Shell.OpenExternal` |
| Fonts | `ui.RegisterFont(embeddedTTF, "Spectral")` twice (300/400) and `ui.RegisterFont(embeddedOTF, "Noto Serif SC")` twice; `fonts.Load()` at startup, logging a failure and falling back to the system serif |
| Text scaling | `theme.Px(c, v)` = v × `clamp(TextScale, 1, 1.5)`; previews are the exception |
| Tests | `ui.NewTester(view, 800, 720)` with typed assertions on text, roles and state; `Advance` drives the countdown; `internal/fonts` checks subset coverage character by character |

Implementation constraints:

- The break panel does not use `Vibrancy` (no glass); its corners come from the window options or the content. `Panel` is an NSPanel that does not activate the app on macOS, a front tool window on the current virtual desktop on Windows, and a utility window / Wayland layer-shell on Linux; focus and virtual-desktop behavior must be verified on the device.
- The settings window does not use top `ui.Tabs`; a sidebar suits four pages with descriptions and holds longer Chinese labels more easily.
- UI elements cannot be kept across frames; keep `ui.Handle`/view state only and put state in the `App` or the view struct. Menu items update in place (`SetLabel` and friends), and while a menu is open only the status line is updated, so the whole menu is not rebuilt and closed.

### 9.1 Platform capability interfaces

| Capability | MyGo API | Platform difference |
|---|---|---|
| Tray / menu | `NewTray` + `NewMenu` | macOS titles work; `SetTitle` is a no-op on Windows; the Linux label depends on the host |
| Window / panel | `NewWindow{Panel, ShowInactive}` | macOS does not activate the app; Windows uses the current virtual desktop; Linux layer-shell / X11 utility |
| Notifications | `NewNotification`, `NotificationsSupported` | macOS packaged app; Windows balloon; Linux notification service |
| Idle / lock | `Power.IdleTime`, `OnLockScreen`/`OnSuspend` | `IdleTime` is 0 on some Linux desktops; the lock event is the screensaver starting |
| DND | no unified API | macOS only when readable without extra permission; platform implementations on Windows/Linux, hidden when missing |
| Login item | `SetOpenAtLogin` | macOS 13+ needs packaging; Linux XDG autostart; Windows Run key |
| Multiple displays | `Screen.Displays` | Available on all three; virtual desktops and output membership per §6 |
| URLs / dialogs | `Shell.OpenExternal`, `Dialog.Message` | Available on all three |

Interfaces are split by capability and views depend only on the abstraction; MyGo types do not leak into the planner or the settings layer.

## 10. Quality floor

Every item is checked before a release:

- Body text contrast ≥ 4.5:1; large text and graphics ≥ 3:1.
- Every interactive control has hover / focus / pressed / disabled.
- No gradient text, glass, neon, glow, or cards inside cards; no kicker/eyebrow — the break panel's `MINI BREAK` is a state annotation, not a small label above a title.
- Icons are drawn or vector; no emoji or Unicode characters standing in for a glyph.
- Motion serves state only: appear, hold, release.
- Error, empty, loading, permission, strict and paused states all exist.
- Real content (a suggestion is a sentence, not Lorem ipsum).
- Light or dark is decided by the scenario, never by category default.
- The same content on all three platforms; only the shell, menus, shortcuts and fonts follow the platform.
- Strict mode has no hidden exit; every pause reason has words.
- No visual asset or copy is taken from Stretchly.

## 11. Open design questions

1. Windows/Linux acceptance: tray titles (no-op on Windows, host-dependent on Linux), panel focus and virtual desktops, Wayland layer-shell, notification packaging and permission, DND detection.
2. Break panel `Panel` focus: macOS does not activate the app; Windows/Linux may become the front window; the M1 prototype verified `ShowInactive` and the global shortcut.
3. Detectability of Windows Focus Assist / Linux DND; when undetectable, hide the row or show it disabled.
4. The status-window entry point and close behavior on GNOME without AppIndicator.
5. `Time` style: never offered on Windows; on Linux only where the tray host supports a label. Hidden or disabled is still undecided.
6. Measured body contrast on the Sand light field; deepen the foreground if needed.
7. Wayland multi-display and layer-shell behavior; the limits of content-screen selection under a compositor.
8. Windows virtual desktops: a panel shows only on the current one; accept "appears on that desktop only", or cover with an extra window — a product decision.

## 12. Implementation record (shipped on macOS)

What follows is shipped detail that is not a separate visual system; the numbers and strings above remain the authority, and this section no longer overrides them.

- **Fonts:** `internal/fonts` bundles Spectral (300/400) and a Noto Serif SC subset (300/400, OFL); `tools/fonts/subset.py` rebuilds it and the coverage test guards missing glyphs. The UI stays on the system sans; suggestions use the bundled serif.
- **Mark and small icons:** `internal/mark` is the single geometry; `internal/icons` provides plus / minus / close / more / check / chevron; the tray template is rasterized by `trayicon` at the 18 pt optical size; the app icon is redrawn from the same geometry by `tools/genicons` (the mark is centered on its ink with a 2.6-unit lift inside the 700 box).
- **General page, app exclusions:** group title + one description line + a right-aligned `Add rule`; each rule has two rows — a 140-wide name input, a process input taking the remaining width (comma separated, placeholder `Comma-separated names, e.g. Xcode, zoom.us`), and a `Remove` text action; the second row holds the `Pause / Break only` Segmented, the `Enabled` switch and a hint at the right. The empty list reads `No rules yet.`; a rule with no name or no process is not written.
- **Schedule page, break health:** one switch row with its description below strict mode; when on, the break panel draws a vermilion inset edge once the counter is above 2 — inset radius 24–60 and alpha 0.14–0.38 scaled by `(count − 2) / 8`; turning the switch off zeroes the counter. The counter runs 0–10: a completed break subtracts its cost, a postponement adds 1, a skip or an interruption adds its cost (mini 1, long 2).
- **About, update hint:** a silent daily check of GitHub Releases; a newer version adds `Version x.y.z is available.` and a `Release notes` link under the version line — no dialog, no interruption.
- **Break panel background image:** when `breakWindow.backgroundImage` is true, an image is drawn from the user data directory's `backgrounds/` (random, not repeating the previous one) with a 60% black scrim over it; it is not exposed in preferences.
- **Strict-mode tray menu:** with `general.showTrayMenuInStrictMode` true, the menu keeps only Settings during a strict break; Quit stays hidden and is refused by `OnBeforeQuit`.
- **Popover and resumed timing:** the tray popover, clicking the notice and the context menu; a restart within 10 minutes resumes the countdown (`schedule.json`); the popover is anchored to the tray icon with `mygo.NewFlyout`, and on Linux there is only the menu.
- **Hold at a due break:** a full-screen app (reading only window bounds/layer/PID, no screen-recording permission; up to 1 hour) and typing (idle < 0.5 s; up to 30 seconds) hold the break back; once the condition clears the whole break starts immediately.
- **Two app-icon appearances:** the source is `icons/AppIcon.icon`, the products are `icons/AppIcon.car` and `resources/icon.png`; regenerating needs Xcode 26+. The marks in About, the welcome window and the sidebar follow the appearance.
- **Focus pause:** one permission-free probe at launch; if Focus cannot be read, the row is hidden, nothing is monitored and no permission prompt appears; once the user grants access, a restart or a settings change takes effect.
- **Settings window and sidebar:** one 800×720 size, minimum 760×560, a scrollable content column; the sidebar draws its own rows (32 high, selected = surface + weight 600) under a 22 px mark and the 13 semibold wordmark.

### 12.1 The Fermata name

- Bundle id `dev.gythialy.fermata`, module `github.com/gythialy/fermata`.
- UI names are localized (`Fermata` in English, the product's Chinese name in Simplified Chinese; menus, About, welcome, notifications and the CLI all follow). The app icon remains the same fermata: Fermata is the mark itself and the Chinese name is its reading, with no strings drawn, because at small sizes they turn into a face.

### 12.2 Implementation status

The v1 macOS build follows this system. What remains is platform work, not
visual drift: packaging and tray hosts for Windows and Linux, a Linux status
entry, and the per-platform icon sizes and popover shells noted above. Two
interactions are deliberately smaller than the drawing: the typing hold is
opt-in, and the stepper steps once per click instead of repeating on hold.

## Appendix: visual canvas

The canvas, its six pages and twenty-six boards, how to open it and the rules
for maintaining it are in [product.md §11](product.md#11-visual-canvas). It is a
visual reference; this document is the specification, and when the two disagree
the design system wins.
