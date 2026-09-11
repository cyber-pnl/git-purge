.PHONY: build run test lint clean

build:
	go build -o bin/git-purge ./cmd/git-purge

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
