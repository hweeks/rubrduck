.PHONY: build run test clean install dev lint fmt
# Use Homebrew Go if installed at standard locations
ifneq ("$(wildcard /opt/homebrew/opt/go/libexec/bin)","")
    export PATH := /opt/homebrew/opt/go/libexec/bin:$(PATH)
endif
ifneq ("$(wildcard /usr/local/opt/go/libexec/bin)","")
    export PATH := /usr/local/opt/go/libexec/bin:$(PATH)
endif

# Variables
BINARY_NAME=rubrduck
MAIN_PATH=cmd/rubrduck/main.go
BUILD_DIR=bin
VERSION ?= dev
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build flags
LDFLAGS=-ldflags "-X github.com/hammie/rubrduck/cmd/rubrduck/commands.Version=${VERSION} \
	-X github.com/hammie/rubrduck/cmd/rubrduck/commands.GitCommit=${GIT_COMMIT} \
	-X github.com/hammie/rubrduck/cmd/rubrduck/commands.BuildDate=${BUILD_DATE}"

# Default target
all: build

# Build the binary
build:
	@echo "Building ${BINARY_NAME}..."
	@mkdir -p ${BUILD_DIR}
	go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME} ${MAIN_PATH}
	@echo "Build complete: ${BUILD_DIR}/${BINARY_NAME}"

# Run the application
run: build
	@${BUILD_DIR}/${BINARY_NAME}

# Run with hot reload (requires air)
dev:
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "Please install air first: go install github.com/air-verse/air@latest"; \
		exit 1; \
	fi

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf ${BUILD_DIR}
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

# Install the binary to GOPATH/bin
install: build
	@echo "Installing ${BINARY_NAME}..."
	go install ${LDFLAGS} ${MAIN_PATH}
	@echo "Installation complete"

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	@echo "Formatting complete"

# Run linters
lint:
	@echo "Running linters..."
	@if command -v golangci-lint > /dev/null; then \
	golangci-lint run --no-config --issues-exit-code=0; \
	else \
	echo "Please install golangci-lint: https://golangci-lint.run/usage/install/"; \
	exit 1; \
	fi

# Run CLI directly (no build step)
cli-run:
	@echo "Running CLI with debug logging..."
	DEBUG=true go run cmd/rubrduck/main.go

# Run CLI with debug logging and capture output
cli-debug:
	@echo "Running CLI with debug logging and capturing to debug.log..."
	DEBUG=true go run cmd/rubrduck/main.go 2>&1 | tee debug.log

# Test tool calls with logging
test-tool-calls:
	@echo "Testing tool calls with debug logging..."
	@echo "what is in the next_steps.md file" | DEBUG=true go run cmd/rubrduck/main.go 2>&1 | tee tool-test.log

# Test streaming with a complex query
test-streaming:
	@echo "Testing streaming with complex query..."
	@echo "list all go files in this project and tell me about the main function" | DEBUG=true go run cmd/rubrduck/main.go 2>&1 | tee stream-test.log

# Watch log file in real time (run in separate terminal)
watch-logs:
	@echo "Watching rubrduck log file..."
	@if [ -f ~/.rubrduck/rubrduck.log ]; then \
		tail -f ~/.rubrduck/rubrduck.log; \
	else \
		echo "Log file not found. Run the app first to create it."; \
	fi

# Clear logs
clear-logs:
	@echo "Clearing log files..."
	@rm -f ~/.rubrduck/rubrduck.log debug.log tool-test.log stream-test.log
	@echo "Logs cleared"

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy
	@echo "Dependencies downloaded"

# Build for multiple platforms
build-all:
	@echo "Building for multiple platforms..."
	@mkdir -p ${BUILD_DIR}
	GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-darwin-amd64 ${MAIN_PATH}
	GOOS=darwin GOARCH=arm64 go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-darwin-arm64 ${MAIN_PATH}
	GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-linux-amd64 ${MAIN_PATH}
	GOOS=linux GOARCH=arm64 go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-linux-arm64 ${MAIN_PATH}
	GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-windows-amd64.exe ${MAIN_PATH}
	@echo "Multi-platform build complete"

# TUI testing targets
test-tui:
	@echo "Running TUI tests..."
	go test -v ./internal/tui2/...

test-tui-coverage:
	@echo "Running TUI tests with coverage..."
	go test -v -coverprofile=tui-coverage.out ./internal/tui2/...
	go tool cover -html=tui-coverage.out -o tui-coverage.html
	@echo "TUI coverage report generated: tui-coverage.html"

test-update:
	@echo "Updating golden files..."
	go test -update ./internal/tui2/...

test-ci:
	@echo "Running tests in CI mode..."
	go test -v -race ./...

test-tui-ci:
	@echo "Running TUI tests in CI mode..."
	go test -v -race ./internal/tui2/...

test-performance:
	@echo "Running TUI performance tests..."
	go test -v -run TestPerformance ./internal/tui2/...

test-scenarios:
	@echo "Running predefined test scenarios..."
	go test -v -run "TestPredefinedScenarios" ./internal/tui2/...

test-workflows:
	@echo "Running workflow tests..."
	go test -v -run "TestWorkflow" ./internal/tui2/...

tdd:
	@echo "Starting TDD workflow..."
	@if [ -z "$(TEST)" ]; then \
		echo "Error: TEST variable not set. Use: make tdd TEST=TestName"; \
		exit 1; \
	fi
	@echo "Running test: $(TEST)"
	@go test -v -run $(TEST) ./internal/tui2/...

test-failing:
	@echo "Running tests that are expected to fail..."
	@go test -v -run "TestTUIFeatureNotImplemented" ./internal/tui2/... || echo "Tests failed as expected"

deps-test:
	@echo "Installing testing dependencies..."
	go get github.com/charmbracelet/x/exp/teatest@latest
	go get github.com/stretchr/testify@latest

test-clean:
	@echo "Cleaning test artifacts..."
	@rm -f tui-coverage.out tui-coverage.html
	@find . -name "*.test" -delete 2>/dev/null || true

# Show help
help:
	@echo "Available targets:"
	@echo ""
	@echo "Build targets:"
	@echo "  make build         - Build the binary"
	@echo "  make run          - Build and run the application"
	@echo "  make dev          - Run with hot reload (requires air)"
	@echo "  make build-all    - Build for multiple platforms"
	@echo "  make install      - Install the binary"
	@echo ""
	@echo "Test targets:"
	@echo "  make test         - Run all tests"
	@echo "  make test-tui     - Run TUI tests specifically"
	@echo "  make test-coverage - Run tests with coverage report"
	@echo "  make test-tui-coverage - Run TUI tests with coverage"
	@echo "  make test-update  - Update golden files"
	@echo "  make test-ci      - Run tests in CI mode"
	@echo "  make test-tui-ci  - Run TUI tests in CI mode"
	@echo "  make test-performance - Run performance tests"
	@echo "  make test-scenarios - Run predefined test scenarios"
	@echo "  make test-workflows - Run workflow tests"
	@echo ""
	@echo "TDD targets:"
	@echo "  make tdd TEST=TestName - Run specific test for TDD"
	@echo "  make test-failing - Run tests expected to fail"
	@echo ""
	@echo "Maintenance targets:"
	@echo "  make clean        - Clean build artifacts"
	@echo "  make test-clean   - Clean test artifacts"
	@echo "  make fmt          - Format code"
	@echo "  make lint         - Run linters"
	@echo "  make deps         - Download dependencies"
	@echo "  make deps-test    - Install testing dependencies"
	@echo ""
	@echo "  make help         - Show this help message"
