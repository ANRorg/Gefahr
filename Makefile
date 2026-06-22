.PHONY: build test test-race test-integration load-check check

build:
	go build -trimpath -o bin/goproxy ./cmd/goproxy

test:
	go test ./...

test-race:
	go test -race ./...

test-integration:
	go test -tags=integration ./test/integration

load-check:
	go run ./cmd/loadcheck

check:
	test -z "$$(gofmt -l $$(git ls-files '*.go'))"
	go vet ./...
	go test -race ./...
