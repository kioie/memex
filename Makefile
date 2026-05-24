.PHONY: all build release test coverage clean install

BINARY_NAME=memex
CMD=./cmd/memex
LDFLAGS=-s -w

all: test build

build:
	go build -o bin/$(BINARY_NAME) $(CMD)

release:
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/$(BINARY_NAME) $(CMD)

install:
	go install $(CMD)

test:
	go test -race -short ./...

coverage:
	go test -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -func=coverage.out

clean:
	rm -rf bin coverage.out
	go clean
