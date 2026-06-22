.PHONY: build test test-race test-integration test-integration-race load-check check acceptance

VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
LDFLAGS = -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT)

build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o bin/goproxy ./cmd/goproxy

test:
	go test ./...

test-race:
	go test -race ./...

test-integration:
	go test -tags=integration ./test/integration

test-integration-race:
	go test -race -tags=integration ./test/integration

load-check:
	go run ./cmd/loadcheck

check:
	test -z "$$(gofmt -l $$(git ls-files '*.go'))"
	go vet ./...
	go test -race ./...

acceptance: check test-integration-race
