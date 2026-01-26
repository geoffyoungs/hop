.PHONY: build dist test debug clean install uninstall lint fmt vet docs help

# Build variables
BINARY_NAME := hop
BUILD_DIR := ./build
CMD_DIR := ./cmd/hop
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

# Go commands
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOVET := $(GOCMD) vet
GOFMT := gofmt
GOMOD := $(GOCMD) mod

# Default target
all: build

## build: Build the binary
build:
	$(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)

## dist: Build release binaries using goreleaser
dist:
	goreleaser build --snapshot --clean

## test: Run all tests
test:
	$(GOTEST) -v ./...

## debug: Build with debug symbols (no stripping)
debug:
	$(GOBUILD) -gcflags="all=-N -l" -o $(BUILD_DIR)/$(BINARY_NAME)-debug $(CMD_DIR)

## clean: Remove build artifacts
clean:
	rm -rf $(BUILD_DIR)
	rm -rf dist/

## install: Install the binary to GOPATH/bin
install: build
	cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/$(BINARY_NAME)

## uninstall: Remove the binary from GOPATH/bin
uninstall:
	rm -f $(GOPATH)/bin/$(BINARY_NAME)

## lint: Run linter (requires golangci-lint)
lint:
	golangci-lint run ./...

## fmt: Format code
fmt:
	$(GOFMT) -s -w .

## vet: Run go vet
vet:
	$(GOVET) ./...

## docs: Generate man pages
docs:
	$(GOCMD) run ./cmd/gendocs

## tidy: Tidy and verify module dependencies
tidy:
	$(GOMOD) tidy
	$(GOMOD) verify

## coverage: Run tests with coverage report
coverage:
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | column -t -s ':'
