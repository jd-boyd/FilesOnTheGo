# FilesOnTheGo Makefile
# A makefile for building, testing, and managing the FilesOnTheGo project

# Go parameters
BINARY_NAME=filesonthego
BINARY_UNIX=$(BINARY_NAME)_unix
GO_VERSION=1.24
BUILD_DIR=build
MAIN_FILE=main.go

# Build and development settings
LDFLAGS=-ldflags "-X main.version=$(shell git describe --tags --always --dirty 2>/dev/null || echo 'dev')"
CGO_ENABLED=0

# Default target
.PHONY: all
all: clean deps lint test build

# Help target - shows available commands
.PHONY: help
help:
	@echo "FilesOnTheGo - Self-hosted file storage and sharing service"
	@echo ""
	@echo "Available commands:"
	@echo "  help        Show this help message"
	@echo "  deps        Install dependencies"
	@echo "  build       Build the binary for current platform"
	@echo "  build-all   Build binaries for multiple platforms"
	@echo "  test        Run all tests"
	@echo "  test-unit   Run unit tests only"
	@echo "  test-integration Run integration tests only"
	@echo "  test-coverage Run tests with coverage report"
	@echo "  benchmark   Run benchmark tests"
	@echo "  race        Run tests with race detection"
	@echo "  lint        Run code linter"
	@echo "  fmt         Format Go code"
	@echo "  vet         Run go vet"
	@echo "  clean       Clean build artifacts and cache"
	@echo "  run         Run the application in development mode"
	@echo "  dev         Set up development environment"
	@echo "  security    Run security checks"
	@echo "  update-deps Update Go dependencies"
	@echo "  mod-tidy    Clean up go.mod"
	@echo ""

# Dependency management
.PHONY: deps
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod verify

.PHONY: update-deps
update-deps:
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy

.PHONY: mod-tidy
mod-tidy:
	@echo "Cleaning up dependencies..."
	go mod tidy

# Code quality
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	go fmt ./...

.PHONY: vet
vet:
	@echo "Running go vet..."
	go vet ./...

.PHONY: lint
lint:
	@echo "Running linter..."
	@if command -v revive >/dev/null 2>&1; then \
		revive ./...; \
	else \
		echo "revive not found. Install with: go install github.com/mgechev/revive@latest"; \
		exit 1; \
	fi

# Building
.PHONY: build
build: clean
	@echo "Building $(BINARY_NAME)..."
	CGO_ENABLED=$(CGO_ENABLED) go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_FILE)
	@echo "Binary built: $(BUILD_DIR)/$(BINARY_NAME)"

.PHONY: build-linux
build-linux: clean
	@echo "Building $(BINARY_UNIX) for Linux..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_UNIX) $(MAIN_FILE)

.PHONY: build-all
build-all: clean
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)

	# Linux AMD64
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_FILE)

	# Linux ARM64
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(MAIN_FILE)

	# Darwin (macOS) AMD64
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_FILE)

	# Darwin (macOS) ARM64
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_FILE)

	# Windows AMD64
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_FILE)

	@echo "All binaries built in $(BUILD_DIR)/"

# Testing
.PHONY: test
test:
	@echo "Running all tests..."
	go test -v ./...

.PHONY: test-unit
test-unit:
	@echo "Running unit tests..."
	go test -v -short ./...

.PHONY: test-integration
test-integration:
	@echo "Running integration tests..."
	@if [ -d "tests/integration" ]; then \
		go test -v -tags=integration ./tests/integration/...; \
	else \
		echo "No integration tests found in tests/integration/"; \
	fi

.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"
	@go test -cover ./... | grep "coverage:" | tail -1

.PHONY: benchmark
benchmark:
	@echo "Running benchmark tests..."
	go test -bench=. -benchmem ./...

.PHONY: race
race:
	@echo "Running tests with race detection..."
	go test -race -v ./...

# Development
.PHONY: run
run: build
	@echo "Starting FilesOnTheGo in development mode..."
	./$(BUILD_DIR)/$(BINARY_NAME) serve

.PHONY: dev
dev: deps fmt vet test
	@echo "Development environment setup complete!"
	@echo "Run 'make run' to start the application."

# Security
.PHONY: security
security:
	@echo "Running security checks..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "gosec not found. Install with: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"; \
	fi
	@echo "Scanning for known security vulnerabilities..."
	go list -json -m all | nancy sleuth

# Tools installation
.PHONY: install-tools
install-tools:
	@echo "Installing development tools..."
	go install github.com/mgechev/revive@latest
	go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	go install github.com/sonatypecommunity/nancy@latest

# Cleanup
.PHONY: clean
clean:
	@echo "Cleaning up..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	go clean -testcache
	go clean -cache

# Docker (if you add Docker support later)
.PHONY: docker-build
docker-build:
	@echo "Building Docker image..."
	docker build -t filesonthego:latest .

.PHONY: docker-run
docker-run:
	@echo "Running Docker container..."
	docker run -p 8090:8090 -v $(PWD)/pb_data:/pb_data filesonthego:latest

# Database operations (using PocketBase)
.PHONY: db-migrate
db-migrate: build
	@echo "Running database migrations..."
	./$(BUILD_DIR)/$(BINARY_NAME) migrate

.PHONY: db-backup
db-backup:
	@echo "Creating database backup..."
	@if [ -d "pb_data" ]; then \
		cp -r pb_data pb_data_backup_$$(date +%Y%m%d_%H%M%S); \
		echo "Backup created: pb_data_backup_$$(date +%Y%m%d_%H%M%S)"; \
	else \
		echo "No pb_data directory found"; \
	fi

# Utility targets
.PHONY: check-go-version
check-go-version:
	@go version | grep -q "go$(GO_VERSION)" || (echo "Go $(GO_VERSION) is required" && exit 1)

.PHONY: pre-commit
pre-commit: fmt vet test lint
	@echo "Pre-commit checks passed!"

.PHONY: ci
ci: deps fmt vet test race lint security
	@echo "CI pipeline completed successfully!"

# Version information
.PHONY: version
version:
	@if [ -f .git ]; then \
		git describe --tags --always --dirty 2>/dev/null || echo "dev"; \
	else \
		echo "dev"; \
	fi

.PHONY: info
info:
	@echo "FilesOnTheGo Build Information:"
	@echo "  Go Version:    $$(go version)"
	@echo "  Go Mod:        $$(go list -m)"
	@echo "  Binary Name:   $(BINARY_NAME)"
	@echo "  Build Dir:     $(BUILD_DIR)"
	@echo "  Git Version:   $$(make version)"
	@echo "  Git Branch:    $$(git branch --show-current 2>/dev/null || echo 'not a git repo')"

# Include custom makefile if it exists
-include Makefile.custom