.PHONY: build test lint all build-cfl build-jtk test-shared lint-shared install-hooks

all: build test lint

build:
	go build -v ./shared/...
	go build -v ./tools/cfl/cmd/cfl
	go build -v ./tools/jtk/cmd/jtk

test:
	go test -v ./shared/...
	go test -v ./tools/cfl/...
	go test -v ./tools/jtk/...

lint:
	cd shared && golangci-lint run
	cd tools/cfl && golangci-lint run
	cd tools/jtk && golangci-lint run

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
