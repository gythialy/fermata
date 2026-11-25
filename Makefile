# Build and check Fermata.
#
#   make build    build the app (no DMG)
#   make dmg      build the app and its DMG
#   make test     run the tests
#   make check    fmt-check, vet and test
#   make run      run from source
#   make icons    redraw the app and menu bar icons (needs Xcode)
#   make screenshots  render the README screenshots into docs/images
#   make assets   redraw the GitHub banner and social preview (needs librsvg)

GO      ?= go
MYGO    ?= $(GO) tool mygo
APP     := build/darwin-universal/Fermata.app
GOFLAGS ?=
PLATFORM ?= darwin/universal
SIGN    ?= -

.PHONY: all build app dmg run test test-race vet fmt fmt-check check icons screenshots assets clean help

all: check build

help:
	@sed -n '2,11p' Makefile

build:
	$(MYGO) build -skip-dmg -platform $(PLATFORM) -sign $(SIGN)

app: build

dmg:
	$(MYGO) build -platform $(PLATFORM) -sign $(SIGN)

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

assets:
	./tools/banner/render.sh

clean:
	rm -rf build
