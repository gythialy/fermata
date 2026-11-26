#!/bin/sh
# Stamp the version a release is built from: the tag's version into mygo.json
# and main.go, with the run and the commit for the about page.
#
#   GITHUB_REF_NAME=v1.1.0 GITHUB_RUN_NUMBER=42 ./tools/version.sh
set -eu

tag=${GITHUB_REF_NAME:?a release runs on a tag}
version=${tag#v}
test -n "$version"
build=${GITHUB_RUN_NUMBER:-0}
commit=$(git rev-parse --short HEAD)

python3 -c 'import json,sys; p="mygo.json"; d=json.load(open(p)); d["version"]=sys.argv[1]; json.dump(d, open(p,"w"), indent=2, ensure_ascii=False); open(p,"a").write("\n")' "$version"

perl -0pi -e "s/version = \"[^\"]*\"/version = \"$version\"/; s/build   = \"[^\"]*\"/build   = \"$build\"/; s/commit  = \"[^\"]*\"/commit  = \"$commit\"/" main.go

echo "stamped $version (build $build, commit $commit)"
