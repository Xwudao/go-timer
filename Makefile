BINARY     := timerd
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT     ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE       ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS    := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE) -s -w"
BUILD_DIR  := dist

.PHONY: all build clean test lint install uninstall fmt vet tidy

all: build

## build: Compile the binary into ./dist/timerd
build:
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) .

## install: Install the binary to /usr/local/bin
install: build
	@echo "Installing $(BINARY) to /usr/local/bin ..."
	install -m 755 $(BUILD_DIR)/$(BINARY) /usr/local/bin/$(BINARY)
	@echo "Done. Run '$(BINARY) init' to get started."

## uninstall: Remove the binary from /usr/local/bin
uninstall:
	rm -f /usr/local/bin/$(BINARY)

## run: Build and run (pass ARGS="..." for arguments)
run: build
	$(BUILD_DIR)/$(BINARY) $(ARGS)

## test: Run all tests
test:
	go test ./... -v -race -count=1

## test-short: Run tests without -race
test-short:
	go test ./... -count=1

## coverage: Generate test coverage report
coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## lint: Run golangci-lint
lint:
	golangci-lint run ./...

## fmt: Format all Go files
fmt:
	gofmt -s -w .
	goimports -w . 2>/dev/null || true

## vet: Run go vet
vet:
	go vet ./...

## tidy: Tidy go.mod and go.sum
tidy:
	go mod tidy

## clean: Remove build artifacts
clean:
	rm -rf $(BUILD_DIR) coverage.out coverage.html

## release: Cross-compile for Linux (amd64 and arm64)
release:
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-linux-arm64 .
	@echo "Binaries in $(BUILD_DIR)/"

## help: Show this help
help:
	@echo "Usage: make [target]"
	@echo ""
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'
