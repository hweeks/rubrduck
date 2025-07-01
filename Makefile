.PHONY: build run test clean install dev lint fmt

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
	go test -v -race -cover ./...

# Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -race -coverprofile=coverage.out ./...
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
		golangci-lint run; \
	else \
		echo "Please install golangci-lint: https://golangci-lint.run/usage/install/"; \
		exit 1; \
	fi

cli-run:
	@echo "Running CLI..."
	go run cmd/rubrduck/main.go

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

# Show help
help:
	@echo "Available targets:"
	@echo "  make build         - Build the binary"
	@echo "  make run          - Build and run the application"
	@echo "  make dev          - Run with hot reload (requires air)"
	@echo "  make test         - Run tests"
	@echo "  make test-coverage - Run tests with coverage report"
	@echo "  make clean        - Clean build artifacts"
	@echo "  make install      - Install the binary"
	@echo "  make fmt          - Format code"
	@echo "  make lint         - Run linters"
	@echo "  make deps         - Download dependencies"
	@echo "  make build-all    - Build for multiple platforms"
	@echo "  make help         - Show this help message" 