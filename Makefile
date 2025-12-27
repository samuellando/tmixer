# filepath: Makefile
.PHONY: all build install fmt test clean run

all: build

build:
	go build ./cmd/tmixer

install:
	go install ./...

fmt:
	go fmt ./...

test:
	go test ./...

clean:
	go clean
	rm -f tmixer

run:
	go run ./cmd/tmixer
