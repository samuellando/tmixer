# filepath: Makefile
.PHONY: all build install fmt test clean run

all: build

check: check-fmt lint spell
build: build-protobuf build-app

build-app:
	go build ./cmd/tmixer

build-protobuf:
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative internal/protocol/service.proto

install:
	go install ./...

fmt:
	go fmt ./...

check-fmt:
	test -z "$$(gofmt -l .)"

test:
	go test ./...

lint:
	golangci-lint run

spell:
	cspell .

test-all:
	go test -count=1 ./...

clean:
	go clean
	rm -f tmixer

run:
	go run ./cmd/tmixer
