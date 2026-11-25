#!/bin/sh
# Render Fermata's GitHub artwork from tools/banner/banner.svg.
#
# The wordmark is set in the fonts the app bundles, so fontconfig is pointed at
# internal/fonts/assets. macOS system fonts stay available for the interface
# text. Run from anywhere:  ./tools/banner/render.sh
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
cd "$root"

command -v rsvg-convert >/dev/null 2>&1 || {
	echo "render.sh: rsvg-convert is missing (brew install librsvg)" >&2
	exit 1
}

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

cat > "$tmp/fonts.conf" <<EOF
<?xml version="1.0"?>
<!DOCTYPE fontconfig SYSTEM "urn:fontconfig:fonts.dtd">
<fontconfig>
  <dir>$root/internal/fonts/assets</dir>
  <dir>$HOME/Library/Fonts</dir>
  <dir>/Library/Fonts</dir>
  <dir>/System/Library/Fonts</dir>
  <dir>/System/Library/Fonts/Supplemental</dir>
  <cachedir>$tmp/cache</cachedir>
</fontconfig>
EOF

export FONTCONFIG_FILE="$tmp/fonts.conf"

mkdir -p .github/assets
rsvg-convert --width 2560 --height 1280 tools/banner/banner.svg -o .github/assets/banner.png
rsvg-convert --width 1280 --height 640 tools/banner/banner.svg -o .github/assets/social-preview.png

echo "wrote .github/assets/banner.png and .github/assets/social-preview.png"
