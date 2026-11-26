#!/bin/sh
# Encode the break window's animation from the frames TestAnimation drew.
#
# make animation renders the frames into build/screens/break-loop; this script
# turns each language into a looping APNG for the README and a WebM/MP4 pair for
# the landing page in docs/. Run from anywhere:  ./tools/screens/animate.sh
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
cd "$root"

command -v ffmpeg >/dev/null 2>&1 || {
	echo "animate.sh: ffmpeg is missing (brew install ffmpeg)" >&2
	exit 1
}

frames=build/screens/break-loop
# 1680 is 2x the width the README shows, so the loop stays sharp on a retina
# display; the video scales the same frames for docs/index.html.
width=1680
height=$((width * 2 / 3))
fps=24

# The field is one flat colour, sampled once from each frame's left edge, and
# the video is composited onto it: H.264 has no alpha, and a straight
# RGBA-to-YUV conversion would leave the cut corners black.
matte="crop=1:1:5:800,scale=${width}:${height}:flags=neighbor"
video_filters="split=2[fg][px];[px]${matte},format=rgba[bg];[fg]scale=${width}:-1:flags=lanczos,format=rgba[fg];[bg][fg]overlay=shortest=1:format=rgb,format=yuv420p"

for locale in en zh-Hans; do
	dir="$frames/$locale"
	out=docs/images
	if [ "$locale" != "en" ]; then
		out="$out/$locale"
	fi
	if [ ! -d "$dir" ]; then
		echo "animate.sh: no frames in $dir (run make animation)" >&2
		exit 1
	fi

	# APNG, not GIF: it keeps the field's rounded corners transparent against
	# either GitHub theme, and its delta frames carry only what changed. The
	# file is named .png because that is the animated PNG a browser plays.
	ffmpeg -y -loglevel error -framerate "$fps" -pattern_type glob -i "$dir/f*.png" \
		-vf "scale=$width:-1:flags=lanczos" \
		-f apng -plays 0 -pred mixed \
		"$out/break-loop.png"

	# One thread and bit-exact output keep the encode reproducible: re-running
	# make animation on unchanged views leaves the files byte for byte the same.
	ffmpeg -y -loglevel error -framerate "$fps" -pattern_type glob -i "$dir/f*.png" \
		-filter_complex "$video_filters" \
		-c:v libvpx-vp9 -pix_fmt yuv420p -crf 32 -b:v 0 -an -threads 1 \
		-flags:v +bitexact -fflags +bitexact \
		"$out/break-loop.webm"

	ffmpeg -y -loglevel error -framerate "$fps" -pattern_type glob -i "$dir/f*.png" \
		-filter_complex "$video_filters" \
		-c:v libx264 -pix_fmt yuv420p -crf 24 -preset slow -threads 1 \
		-flags:v +bitexact -fflags +bitexact -movflags +faststart -an \
		"$out/break-loop.mp4"
done

for file in docs/images/break-loop.png docs/images/break-loop.webm docs/images/break-loop.mp4 \
	docs/images/zh-Hans/break-loop.png docs/images/zh-Hans/break-loop.webm docs/images/zh-Hans/break-loop.mp4; do
	printf '%8s  %s\n' "$(du -h "$file" | cut -f1)" "$file"
done
