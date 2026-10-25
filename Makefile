.PHONY: build run test test-integration lint clean

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X github.com/cyber-pnl/git-purge/internal/cli.Version=$(VERSION)

build:
	go build -ldflags "$(LDFLAGS)" -o bin/git-purge ./cmd/git-purge

run:
	go run ./cmd/git-purge

test:
	go test ./...

test-integration:
	go test ./... -run Integration -v

lint:
	$(shell go env GOPATH)/bin/golangci-lint run

clean:
	rm -rf bin/
