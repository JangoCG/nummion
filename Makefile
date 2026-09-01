.PHONY: build install test fmt fmt-check vet check clean

VERSION ?= dev
PREFIX ?= $(HOME)/.local

build:
	@mkdir -p bin
	go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o bin/lexware ./cmd/lexware

install: build
	install -d "$(PREFIX)/bin"
	install -m 0755 bin/lexware "$(PREFIX)/bin/lexware"

test:
	go test ./...

fmt:
	gofmt -w cmd internal skills

fmt-check:
	@test -z "$$(gofmt -l cmd internal skills)" || (gofmt -l cmd internal skills && exit 1)

vet:
	go vet ./...

check: fmt-check vet test build

clean:
	rm -rf bin dist
