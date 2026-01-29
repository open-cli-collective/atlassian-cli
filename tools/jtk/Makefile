.PHONY: build test clean install lint

BINARY_NAME=jtk
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE?=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS=-ldflags "-X github.com/open-cli-collective/jira-ticket-cli/internal/version.Version=$(VERSION) \
	-X github.com/open-cli-collective/jira-ticket-cli/internal/version.Commit=$(COMMIT) \
	-X github.com/open-cli-collective/jira-ticket-cli/internal/version.BuildDate=$(BUILD_DATE)"

build:
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/jtk

test:
	go test -race -v ./...

test-cover:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

install: build
	cp bin/$(BINARY_NAME) /usr/local/bin/

lint:
	golangci-lint run

tidy:
	go mod tidy

deps:
	go mod download

all: tidy lint test build
