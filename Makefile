.PHONY: all build release test test-full test-integration lint coverage coverage-check coverage-sonar coverage-sonar-check vulncheck clean install

BINARY_NAME=memex
CMD=./cmd/memex
LDFLAGS=-s -w
COVERAGE_MIN=75
COVERAGE_SONAR_MIN=80

all: test build

build:
	go build -o bin/$(BINARY_NAME) $(CMD)

release:
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o bin/$(BINARY_NAME) $(CMD)

install:
	go install $(CMD)

test:
	go test -race -short ./...

test-full:
	go test -race ./memex ./cmd/memex

test-integration:
	go test -tags=integration -race -timeout 5m ./integration/...

lint:
	golangci-lint run ./...

coverage:
	go test -race -coverprofile=coverage.out -covermode=atomic ./memex/...
	go tool cover -func=coverage.out

coverage-check: coverage
	@total=$$(go tool cover -func=coverage.out | awk '/^total:/ {gsub(/%/,"",$$3); print $$3}'); \
	awk -v t="$$total" -v min="$(COVERAGE_MIN)" 'BEGIN { if (t+0 < min+0) { printf "coverage %.1f%% below minimum %s%%\n", t, min; exit 1 } else { printf "coverage %.1f%% (min %s%%)\n", t, min } }'

coverage-sonar:
	go test -race -short -coverprofile=coverage.out -covermode=atomic ./memex/...

coverage-sonar-check: coverage-sonar
	@total=$$(go tool cover -func=coverage.out | awk '/^total:/ {gsub(/%/,"",$$3); print $$3}'); \
	awk -v t="$$total" -v min="$(COVERAGE_SONAR_MIN)" 'BEGIN { if (t+0 < min+0) { printf "Sonar coverage %.1f%% below minimum %s%%\n", t, min; exit 1 } else { printf "Sonar coverage %.1f%% (min %s%%)\n", t, min } }'

vulncheck:
	govulncheck ./...

clean:
	rm -rf bin coverage.out
	go clean
