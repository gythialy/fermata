#!/usr/bin/env python3
"""Rebuild the bundled serif subsets.

The app ships Spectral for Latin and a Noto Serif SC subset for the Han
characters the suggestions and the tagline use. Both are OFL; the license
texts sit beside the fonts. Run this only when the copy changes:

    python3 tools/fonts/subset.py \
        --spectral-light /path/to/Spectral-Light.ttf \
        --spectral-regular /path/to/Spectral-Regular.ttf \
        --noto-serif /path/to/NotoSerifCJK.ttc

The Noto collection must hold the SC Light and SC Regular faces at indexes 6
and 10 (Google's NotoSerifCJK.ttc does). The subset is text-driven: every
string of locales/*.json and internal/ideas/assets/*.json is included, plus
the design canvas' specimen sentences. A Go test checks the coverage, so a
new suggestion that the subset misses fails the build rather than falling
back silently.
"""

import argparse
import json
import os
import sys
from pathlib import Path

from fontTools import subset
from fontTools.ttLib import TTFont

ROOT = Path(__file__).resolve().parents[2]
ASSETS = ROOT / "internal" / "fonts" / "assets"

# Strings the design canvas sets in the serif; they are not part of the app's
# catalogs, but the bundled subset should draw them too.
EXTRA = [
    "望向窗外，直到眼睛放松下来。",
    "休息，可以多停一会儿。",
    "休息，可以多停一会儿。界面只做三件事：标出时长、给出一条建议、留一个按钮；其余时间，它退到菜单栏或系统托盘里。",
    "System voice, serif for the break.",
    "Fermata", "停琴",
]

PUNCTUATION = "，。、；：！？「」『』（）《》…—·“”‘’"


def catalog_text(path: Path) -> list[str]:
    """Every string value of a catalog, skipping the @-metadata."""
    data = json.loads(path.read_text(encoding="utf-8"))
    out: list[str] = []

    def walk(v):
        if isinstance(v, str):
            out.append(v)
        elif isinstance(v, list):
            for item in v:
                walk(item)
        elif isinstance(v, dict):
            for key, item in v.items():
                if not key.startswith("@"):
                    walk(item)

    walk(data)
    return out


def charset() -> str:
    text: list[str] = []
    for path in sorted((ROOT / "locales").glob("*.json")):
        text += catalog_text(path)
    for path in sorted((ROOT / "internal" / "ideas" / "assets").glob("*.json")):
        text += catalog_text(path)
    text += EXTRA
    chars = set("".join(text)) | set(PUNCTUATION)
    chars |= {chr(c) for c in range(0x20, 0x7F)}
    return "".join(sorted(chars))


def rename(font: TTFont, family: str, subfamily: str) -> None:
    name = font["name"]
    for rec in list(name.names):
        if rec.nameID == 1:
            rec.string = family
        elif rec.nameID == 2:
            rec.string = subfamily
        elif rec.nameID == 4:
            rec.string = family if subfamily == "Regular" else f"{family} {subfamily}"
        elif rec.nameID == 6:
            rec.string = f"{family}-{subfamily}"
        elif rec.nameID == 16:
            rec.string = family
        elif rec.nameID == 17:
            rec.string = subfamily


def write(font: TTFont, out: Path) -> None:
    font.save(out)
    print(f"wrote {out.relative_to(ROOT)} ({out.stat().st_size // 1024} KiB)")


def subset_font(source: Path, out: Path, text: str, index: int | None = None,
                family: str | None = None, subfamily: str | None = None,
                weight: int | None = None) -> None:
    font = TTFont(source, fontNumber=index) if index is not None else TTFont(source)
    options = subset.Options()
    options.layout_features = ["*"]
    options.name_IDs = ["*"]
    options.name_legacy = True
    options.name_languages = ["*"]
    options.glyph_names = True
    options.symbol_cmap = True
    options.legacy_cmap = True
    options.notdef_glyph = True
    options.notdef_outline = True
    options.recommended_glyphs = True
    subsetter = subset.Subsetter(options)
    subsetter.populate(text=text)
    subsetter.subset(font)
    if family and subfamily:
        rename(font, family, subfamily)
    if weight:
        font["OS/2"].usWeightClass = weight
    write(font, out)


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--spectral-light", required=True, type=Path)
    parser.add_argument("--spectral-regular", required=True, type=Path)
    parser.add_argument("--noto-serif", required=True, type=Path,
                        help="NotoSerifCJK.ttc; indexes 6 and 10 are SC Light and SC Regular")
    parser.add_argument("--noto-light-index", type=int, default=6)
    parser.add_argument("--noto-regular-index", type=int, default=10)
    args = parser.parse_args()

    text = charset()
    han = "".join(c for c in text if "\u3400" <= c <= "\u9fff")
    print(f"{len(text)} characters, {len(han)} Han")

    # Latin: keep the whole lowercase/uppercase range and punctuation; the
    # file is small and a missing accent would be a silent fallback.
    latin = "".join(chr(c) for c in range(0x20, 0x100))
    latin += "".join(chr(c) for c in range(0x2000, 0x2070)) + PUNCTUATION
    subset_font(args.spectral_light, ASSETS / "spectral-light.ttf", latin, family="Spectral", subfamily="Light", weight=300)
    subset_font(args.spectral_regular, ASSETS / "spectral-regular.ttf", latin, family="Spectral", subfamily="Regular", weight=400)

    subset_font(args.noto_serif, ASSETS / "noto-serif-sc-light.otf", text,
                index=args.noto_light_index, family="Noto Serif SC", subfamily="Light", weight=300)
    subset_font(args.noto_serif, ASSETS / "noto-serif-sc-regular.otf", text,
                index=args.noto_regular_index, family="Noto Serif SC", subfamily="Regular", weight=400)
    return 0


if __name__ == "__main__":
    sys.exit(main())
