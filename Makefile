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

# Where `make install` puts the binary. Empty means GOBIN, then GOPATH/bin.
# Override with: make install INSTALL_DIR=/usr/local/bin
INSTALL_DIR ?=

## install: Install the binary to GOBIN (or GOPATH/bin)
install:
	@# mise-managed toolchains only: earlier `go install` runs left copies in mise's
	@# version-scoped Go dir, and their shims would win on $$PATH. Skipped without mise.
	@go_installs="$${MISE_DATA_DIR:-$$HOME/.local/share/mise}/installs/go"; \
	if command -v mise >/dev/null 2>&1 && ls $$go_installs/*/bin/$(BINARY_NAME) >/dev/null 2>&1; then \
		rm -f $$go_installs/*/bin/$(BINARY_NAME); \
		mise reshim || true; \
		echo "Removed stale $(BINARY_NAME) binaries from mise's Go install dirs"; \
	fi
	@command -v go >/dev/null 2>&1 || { echo "go not found on PATH" >&2; exit 1; }
	@dir="$(INSTALL_DIR)"; \
	if [ -z "$$dir" ]; then dir="$$(go env GOBIN)"; fi; \
	case "$$dir" in */mise/installs/go/*) dir="" ;; esac; \
	if [ -z "$$dir" ]; then dir="$$(go env GOPATH)/bin"; fi; \
	mkdir -p "$$dir"; \
	echo "Installing $(BINARY_NAME) to $$dir"; \
	go build $(LDFLAGS) -o "$$dir/$(BINARY_NAME)" $(CMD_PATH)

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
