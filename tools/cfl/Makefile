.PHONY: build test test-cover test-short lint clean install deps

BINARY := cfl
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -ldflags "-s -w \
	-X github.com/open-cli-collective/atlassian-go/version.Version=$(VERSION) \
	-X github.com/open-cli-collective/atlassian-go/version.Commit=$(COMMIT) \
	-X github.com/open-cli-collective/atlassian-go/version.BuildDate=$(DATE)"

# Build the binary
build:
	go build $(LDFLAGS) -o bin/$(BINARY) ./cmd/cfl

# Run all tests with race detection
test:
	go test -v -race ./...

# Run tests with coverage report
test-cover:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Run short tests only (skip slow/integration tests)
test-short:
	go test -v -short ./...

# Run linter
lint:
	golangci-lint run

# Clean build artifacts
clean:
	rm -rf bin/ dist/ coverage.out coverage.html

# Install binary to GOPATH/bin
install: build
	cp bin/$(BINARY) $(shell go env GOPATH)/bin/

# Download dependencies
deps:
	go mod download
	go mod tidy

# Run the CLI (for development)
run:
	go run ./cmd/cfl $(ARGS)

# Format code
fmt:
	go fmt ./...
	goimports -w .

# Verify module
verify:
	go mod verify
