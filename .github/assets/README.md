# GitHub assets

Artwork for the repository page. Everything here is generated from the app's
own design tokens and bundled fonts, so it stays in step with the product.

| File | Size | Used for |
|---|---|---|
| `banner.png` | 2560×1280 (2× of 1280×640) | The README hero image |
| `social-preview.png` | 1280×640 | Repository **Settings → Social preview** |
| `mark.svg` | 32-unit mark, scalable | Avatars, slides, anywhere a vector mark is needed |
| `icon.png` | 1024×1024 | Repository avatar or organization logo |

## Regenerating

```sh
make assets      # banner.png and social-preview.png, needs rsvg-convert
make screenshots # docs/images/*.png, rendered from the app's own views
```

`make assets` runs [`tools/banner/render.sh`](../../tools/banner/render.sh),
which points fontconfig at `internal/fonts/assets` so the wordmark is set in
Spectral and 停琴 in Noto Serif SC — the same faces the break window uses. On
macOS, `brew install librsvg` provides `rsvg-convert`.

The banner source is [`tools/banner/banner.svg`](../../tools/banner/banner.svg).
Edit it there and re-render; do not trace the PNG.

The screenshots are real renders of the app's views through `ui.NewTester`
(no mockups): see the build-tagged `screenshot_test.go` files beside
`internal/breakwin`, `internal/prefs`, `internal/welcome` and
`internal/popover`.

## Copy

- English: **Fermata — a menu bar break reminder.**
- Chinese: **停琴 — 菜单栏里的休息提醒。**
- Tagline: **A break, held.** / **一次被延长的休息。**
