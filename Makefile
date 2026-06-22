.PHONY: build test test-race test-integration check

build:
	go build -trimpath -o bin/goproxy ./cmd/goproxy

test:
	go test ./...

test-race:
	go test -race ./...

test-integration:
	go test -tags=integration ./test/integration

check:
	test -z "$$(gofmt -l .)"
	go vet ./...
	go test -race ./...

