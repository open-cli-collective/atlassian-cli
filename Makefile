.PHONY: build test lint all build-cfl build-jtk

all: build test lint

build:
	go build -v ./tools/cfl/cmd/cfl
	go build -v ./tools/jtk/cmd/jtk

test:
	go test -v ./tools/cfl/...
	go test -v ./tools/jtk/...

lint:
	cd tools/cfl && golangci-lint run
	cd tools/jtk && golangci-lint run

build-cfl:
	go build -v -o bin/cfl ./tools/cfl/cmd/cfl

build-jtk:
	go build -v -o bin/jtk ./tools/jtk/cmd/jtk
