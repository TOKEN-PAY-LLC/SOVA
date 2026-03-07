.PHONY: build build-all build-linux build-windows build-macos build-android build-ios clean test install-deps release

# Version
VERSION := v2.0.0
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')

# Build flags
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildTime=$(BUILD_TIME)"

# Directories
OUTPUT_DIR := ./bin
DIST_DIR := ./dist

# Default target
build: build-server build-client

# Build server
build-server:
	@echo "Building SOVA Server..."
	@mkdir -p $(OUTPUT_DIR)
	go build $(LDFLAGS) -o $(OUTPUT_DIR)/sova-server ./server

# Build client
build-client:
	@echo "Building SOVA Client..."
	@mkdir -p $(OUTPUT_DIR)
	go build $(LDFLAGS) -o $(OUTPUT_DIR)/sova ./client

# Build ALL platforms
build-all: build-linux-amd64 build-linux-arm64 build-windows-amd64 build-windows-arm64 build-macos-amd64 build-macos-arm64

# Linux builds
build-linux-amd64:
	@echo "Building for Linux AMD64..."
	@mkdir -p $(DIST_DIR)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(DIST_DIR)/sova-server-linux-amd64 ./server
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(DIST_DIR)/sova-linux-amd64 ./client

build-linux-arm64:
	@echo "Building for Linux ARM64..."
	@mkdir -p $(DIST_DIR)
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(DIST_DIR)/sova-server-linux-arm64 ./server
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(DIST_DIR)/sova-linux-arm64 ./client

# Windows builds
build-windows-amd64:
	@echo "Building for Windows AMD64..."
	@mkdir -p $(DIST_DIR)
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(DIST_DIR)/sova-server-windows-amd64.exe ./server
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(DIST_DIR)/sova-windows-amd64.exe ./client

build-windows-arm64:
	@echo "Building for Windows ARM64..."
	@mkdir -p $(DIST_DIR)
	GOOS=windows GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(DIST_DIR)/sova-server-windows-arm64.exe ./server
	GOOS=windows GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(DIST_DIR)/sova-windows-arm64.exe ./client

# macOS builds
build-macos-amd64:
	@echo "Building for macOS AMD64..."
	@mkdir -p $(DIST_DIR)
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(DIST_DIR)/sova-server-macos-amd64 ./server
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(DIST_DIR)/sova-macos-amd64 ./client

build-macos-arm64:
	@echo "Building for macOS ARM64..."
	@mkdir -p $(DIST_DIR)
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(DIST_DIR)/sova-server-macos-arm64 ./server
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(DIST_DIR)/sova-macos-arm64 ./client

# Android (requires gomobile)
build-android:
	@echo "Building for Android..."
	@mkdir -p $(DIST_DIR)
	gomobile build -o $(DIST_DIR)/sova-android.aar ./client

# iOS (requires gomobile)
build-ios:
	@echo "Building for iOS..."
	@mkdir -p $(DIST_DIR)
	gomobile build -o $(DIST_DIR)/sova-ios.xcframework ./client

# Test
test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Benchmark
bench:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

# Install dependencies
install-deps:
	@echo "Installing Go dependencies..."
	go mod download
	go mod tidy
	@echo "Dependencies installed."

# Crypto tests
test-crypto:
	@echo "Testing cryptography module..."
	go test -v -run TestCrypto ./common

# AI adapter tests
test-ai:
	@echo "Testing AI adapter..."
	go test -v -run TestAdaptive ./common

# Integration tests
test-integration:
	@echo "Running integration tests..."
	go test -v -tags integration ./...

# Release - package all binaries
release: build-all
	@echo "Creating release artifacts..."
	@mkdir -p $(DIST_DIR)/release
	cd $(DIST_DIR) && \
	tar -czf release/sova-linux-amd64-$(VERSION).tar.gz sova-server-linux-amd64 sova-linux-amd64 && \
	tar -czf release/sova-linux-arm64-$(VERSION).tar.gz sova-server-linux-arm64 sova-linux-arm64 && \
	tar -czf release/sova-macos-amd64-$(VERSION).tar.gz sova-server-macos-amd64 sova-macos-amd64 && \
	tar -czf release/sova-macos-arm64-$(VERSION).tar.gz sova-server-macos-arm64 sova-macos-arm64 && \
	zip -q release/sova-windows-amd64-$(VERSION).zip sova-server-windows-amd64.exe sova-windows-amd64.exe && \
	zip -q release/sova-windows-arm64-$(VERSION).zip sova-server-windows-arm64.exe sova-windows-arm64.exe
	@echo "Release artifacts created in $(DIST_DIR)/release"

# Clean
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(OUTPUT_DIR)
	rm -rf $(DIST_DIR)
	rm -f coverage.out coverage.html
	@echo "Clean complete."

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Lint
lint:
	@echo "Linting code..."
	golangci-lint run ./...

# Help
help:
	@echo "SOVA Protocol Build System"
	@echo ""
	@echo "Available targets:"
	@echo "  make build              - Build server and client for current OS"
	@echo "  make build-all          - Build for all supported platforms"
	@echo "  make build-server       - Build server only"
	@echo "  make build-client       - Build client only"
	@echo "  make build-linux-*      - Build for Linux (amd64, arm64)"
	@echo "  make build-windows-*    - Build for Windows (amd64, arm64)"
	@echo "  make build-macos-*      - Build for macOS (amd64, arm64)"
	@echo "  make test               - Run all tests"
	@echo "  make test-crypto        - Test crypto module"
	@echo "  make test-ai            - Test AI adapter"
	@echo "  make test-integration   - Run integration tests"
	@echo "  make bench              - Run benchmarks"
	@echo "  make release            - Create release artifacts for all platforms"
	@echo "  make install-deps       - Install Go dependencies"
	@echo "  make fmt                - Format code"
	@echo "  make lint               - Lint code"
	@echo "  make clean              - Remove build artifacts"
	@echo "  make help               - Show this help message"
