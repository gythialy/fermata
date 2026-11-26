# Build and check Fermata.
#
#   make build    build the app (no DMG)
#   make dmg      build the app and its DMG
#   make build-linux    build the Linux app and its packages
#   make build-windows  build the Windows app, its installer and its portable archive (installer needs NSIS)
#   make build-all  build every platform into build/
#   make e2e      run the e2e suite against an e2e build of this machine
#   make test     run the tests
#   make check    fmt-check, vet and test
#   make run      run from source
#   make icons    redraw the app and menu bar icons (needs Xcode)
#   make screenshots  render the README screenshots into docs/images
#   make animation  render the README's loop and the landing page's video (needs ffmpeg)
#   make assets   redraw the GitHub banner and social preview (needs librsvg)

GO      ?= go
MYGO    ?= $(GO) tool mygo
APP     := build/darwin-universal/Fermata.app
GOFLAGS ?=
PLATFORM ?= darwin/universal
SIGN    ?= -
VERSION := $(shell python3 -c 'import json;print(json.load(open("mygo.json"))["version"])' 2>/dev/null || echo 0.0.0)
WINDOWS_ARCHES := amd64 arm64

.PHONY: all build app dmg build-linux build-windows build-all e2e run test test-race vet fmt fmt-check check icons screenshots animation assets clean help

all: check build

help:
	@sed -n '2,16p' Makefile

build:
	$(MYGO) build -skip-dmg -platform $(PLATFORM) -sign $(SIGN)

app: build

dmg:
	$(MYGO) build -platform $(PLATFORM) -sign $(SIGN)

build-linux:
	$(MYGO) build -platform linux/amd64,linux/arm64

build-windows:
	$(MYGO) build -platform windows/amd64,windows/arm64
	@for arch in $(WINDOWS_ARCHES); do \
		$(GO) run ./tools/portablezip -dir build/windows-$$arch \
			-out build/Fermata-$(VERSION)-windows-$$arch-portable.zip || exit 1; \
	done

build-all:
	$(MYGO) build -platform darwin/universal,windows/amd64,windows/arm64,linux/amd64,linux/arm64 -sign $(SIGN)

e2e:
	./tools/e2e/run.sh

run:
	$(GO) $(GOFLAGS) run .

test:
	$(GO) $(GOFLAGS) test ./...

test-race:
	$(GO) $(GOFLAGS) test -race ./...

vet:
	$(GO) $(GOFLAGS) vet ./...

fmt:
	gofmt -w .

fmt-check:
	@files="$$(gofmt -l .)"; if [ -n "$$files" ]; then echo "gofmt needed:"; echo "$$files"; exit 1; fi

check: fmt-check vet test

icons:
	$(GO) run ./tools/genicons

screenshots:
	$(GO) test -tags screenshots -run TestScreenshots ./internal/breakwin ./internal/prefs ./internal/welcome ./internal/popover

animation:
	$(GO) test -tags screenshots -run TestAnimation ./internal/breakwin
	./tools/screens/animate.sh

assets:
	./tools/banner/render.sh

clean:
	rm -rf build
