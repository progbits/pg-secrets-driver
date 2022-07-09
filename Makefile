.PHONY: all build fmt

all: build

build:
	go build

fmt:
	go fmt ./...

