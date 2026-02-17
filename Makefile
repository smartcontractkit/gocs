.PHONY: build test lint fmt vet clean install coverage help

# Build variables
BINARY_NAME := gocs
CMD_PATH := ./cmd/gocs
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

# Default target
all: lint test build

## build: Build the binary
build:
	go build $(LDFLAGS) -o $(BINARY_NAME) $(CMD_PATH)

## install: Install the binary to GOPATH/bin
install:
	go install $(LDFLAGS) $(CMD_PATH)

## test: Run tests
test:
	go test -race -v ./...

## coverage: Run tests with coverage
coverage:
	go test -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## lint: Run golangci-lint
lint:
	golangci-lint run ./...

## fmt: Format code
fmt:
	go fmt ./...
	gofumpt -l -w .

## vet: Run go vet
vet:
	go vet ./...

## clean: Remove build artifacts
clean:
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html

## tidy: Tidy and verify dependencies
tidy:
	go mod tidy
	go mod verify

## check: Run all checks (fmt, vet, lint, test)
check: fmt vet lint test

## help: Show this help
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | column -t -s ':' | sed 's/^/  /'
