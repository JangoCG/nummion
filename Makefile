.PHONY: build install test fmt fmt-check vet check completions installer-test release-check snapshot security-check hooks clean

VERSION ?= dev
PREFIX ?= $(HOME)/.local

build:
	@mkdir -p bin
	go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o bin/num ./cmd/nummion

install: build
	install -d "$(PREFIX)/bin"
	install -m 0755 bin/num "$(PREFIX)/bin/num"
	ln -sf num "$(PREFIX)/bin/lexware"

test:
	go test ./...

fmt:
	gofmt -w cmd internal skills

fmt-check:
	@test -z "$$(gofmt -l cmd internal skills)" || (gofmt -l cmd internal skills && exit 1)

vet:
	go vet ./...

check: fmt-check vet test build

completions:
	bash scripts/completions.sh

installer-test:
	python3 -m unittest discover -s scripts -p 'test_*.py'

release-check:
	goreleaser check
	bash -n scripts/install.sh scripts/completions.sh
	$(MAKE) installer-test

snapshot: release-check
	goreleaser release --snapshot --clean --skip=publish,sign

security-check:
	gitleaks git . --log-opts="--all --full-history" --redact=100 --no-banner --ignore-gitleaks-allow
	python3 scripts/verify-secret-scanner.py
	go run golang.org/x/vuln/cmd/govulncheck@v1.7.0 ./...

hooks:
	pre-commit install --install-hooks

clean:
	rm -rf bin dist completions
