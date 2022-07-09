.PHONY: all build build-example run-example test fmt

all: build

build:
	go build

build-example:
	go build -o bin/example example/main.go

run-example: build-example
	./bin/example

test:
	go test --count=1 ./...

fmt:
	go fmt ./...

