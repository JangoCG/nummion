.PHONY: build test fmt fmt-check vet check clean

VERSION ?= dev

build:
	@mkdir -p bin
	go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o bin/lexware ./cmd/lexware

test:
	go test ./...

fmt:
	gofmt -w cmd internal

fmt-check:
	@test -z "$$(gofmt -l cmd internal)" || (gofmt -l cmd internal && exit 1)

vet:
	go vet ./...

check: fmt-check vet test build

clean:
	rm -rf bin dist
