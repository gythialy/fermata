# Bundled serif subsets

The break voice is set in two OFL families, committed under
`internal/fonts/assets/`:

| File | Family | Weight |
|---|---|---|
| `spectral-light.ttf` | Spectral | 300 |
| `spectral-regular.ttf` | Spectral | 400 |
| `noto-serif-sc-light.otf` | Noto Serif SC | 300 |
| `noto-serif-sc-regular.otf` | Noto Serif SC | 400 |

Spectral is the Latin original from Google Fonts. Noto Serif SC is a subset of
Noto Serif CJK SC, cut to the characters in `locales/*.json`,
`internal/ideas/assets/*.json` and the specimen sentences in `subset.py`. The
licenses are `OFL-Spectral.txt` and `OFL-NotoSerifSC.txt` beside the fonts.

## Rebuild

The subset needs Python with fontTools (3.11 in this checkout has it) and the
two source files:

```sh
python3 tools/fonts/subset.py \
  --spectral-light /path/to/Spectral-Light.ttf \
  --spectral-regular /path/to/Spectral-Regular.ttf \
  --noto-serif /path/to/NotoSerifCJK.ttc
```

`NotoSerifCJK.ttc` is the Google collection: index 6 is SC Light, index 10 is
SC Regular. On macOS the installed collection is
`~/Library/Fonts/NotoSerifCJK.ttc`.

The script rewrites the family names to `Spectral` / `Noto Serif SC` and keeps
the weights so MyGo's `ui.RegisterFont` picks the right face.

## Why a subset

The full Noto Serif CJK SC is tens of megabytes per weight. The app draws only
suggestion text, the tagline and the canvas specimen in this family, so the
subset is a few hundred kilobytes. `internal/fonts/fonts_test.go` parses the
shipped fonts and checks every character the catalogs can set in the serif; a
new suggestion that misses a glyph fails the test instead of falling back
silently.
