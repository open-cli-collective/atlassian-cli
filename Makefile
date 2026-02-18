.PHONY: build test lint tidy check all build-cfl build-jtk test-shared lint-shared install-hooks

# CI gate: everything that must pass before merge
check: tidy lint test build

all: check

# Build all binaries into bin/
build:
	go build -v -o bin/cfl ./tools/cfl/cmd/cfl
	go build -v -o bin/jtk ./tools/jtk/cmd/jtk

# Run tests with race detector
test:
	go test -race ./shared/...
	go test -race ./tools/cfl/...
	go test -race ./tools/jtk/...

# Lint with golangci-lint (config in each module's .golangci.yml)
lint:
	cd shared && golangci-lint run
	cd tools/cfl && golangci-lint run
	cd tools/jtk && golangci-lint run

# Tidy and verify modules are clean
tidy:
	cd shared && go mod tidy
	cd tools/cfl && go mod tidy
	cd tools/jtk && go mod tidy
	git diff --exit-code shared/go.mod shared/go.sum tools/cfl/go.mod tools/cfl/go.sum tools/jtk/go.mod tools/jtk/go.sum

# Build individual tools to bin/
build-cfl:
	go build -v -o bin/cfl ./tools/cfl/cmd/cfl

build-jtk:
	go build -v -o bin/jtk ./tools/jtk/cmd/jtk

test-shared:
	go test -v -race -coverprofile=coverage-shared.out ./shared/...

lint-shared:
	cd shared && golangci-lint run

install-hooks:
	cp hooks/pre-commit .git/hooks/pre-commit
	chmod +x .git/hooks/pre-commit
	@echo "Pre-commit hook installed."
