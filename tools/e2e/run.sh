#!/bin/sh
# Runs the e2e suite against an e2e build of the app on this machine: builds
# it, starts it in a data directory of its own with the automation socket on,
# runs the tests, and stops it again.
#
#   tools/e2e/run.sh                 # the machine's own platform
#   tools/e2e/run.sh darwin/arm64    # one target
#
# A test that needs a break window skips here: start one with `Fermata mini`
# while the run holds the app.
set -eu
cd "$(dirname "$0")/../.."

platform=${1:-$(go env GOOS)/$(go env GOARCH)}
# mygo build names its output directories with a hyphen, as darwin-arm64.
out=build/${platform%/*}-${platform#*/}
case $platform in
darwin/*) app=$out/Fermata.app/Contents/MacOS/Fermata ;;
*) app=$out/fermata ;;
esac
addr=${FERMATA_E2E_ADDR:-127.0.0.1:9100}

GOFLAGS="-tags=e2e ${GOFLAGS:-}" MYGO_INSPECTOR=1 go tool mygo build -skip-dmg -platform "$platform" >/dev/null

data=$(mktemp -d "${TMPDIR:-/tmp}/fermata-e2e.XXXXXX")
cleanup() {
	kill "$pid" 2>/dev/null || true
	# The app may still be closing its log when the kill lands.
	sleep 0.5
	rm -rf "$data"
}
trap cleanup EXIT

FERMATA_E2E_DATA=$data MYGO_AUTOMATION=tcp:$addr "$app" preferences &
pid=$!

host=${addr%:*}
port=${addr##*:}
tries=0
until nc -z "$host" "$port" 2>/dev/null; do
	tries=$((tries + 1))
	if [ "$tries" -ge 100 ]; then
		echo "the app did not listen on $addr" >&2
		exit 1
	fi
	sleep 0.1
done

FERMATA_E2E_ADDR=$addr FERMATA_E2E_APP=${platform%%/*} go test -tags e2e ./internal/e2e -v
