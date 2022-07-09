.PHONY: all build test fmt

all: build

build:
	go build

test:
	go test --count=1 ./...

fmt:
	go fmt ./...

