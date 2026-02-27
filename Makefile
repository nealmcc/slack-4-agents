# Makefile for Slack MCP Server

# Binary name
BINARY_NAME=slack-mcp
# Main package path
MAIN_PATH=./cmd/slack-mcp
# Install location
INSTALL_PATH=$(HOME)/bin

# Go commands
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOVET=$(GOCMD) vet
GOFMT=gofmt
GOMOD=$(GOCMD) mod

# Build flags
LDFLAGS=-ldflags "-s -w"

.PHONY: all build install clean test vet fmt tidy generate help

# Default target
all: build

# Build the binary
build: generate
	@echo "Building $(BINARY_NAME)..."
	$(GOBUILD) -o $(BINARY_NAME) $(MAIN_PATH)

# Build with optimizations
build-release:
	@echo "Building $(BINARY_NAME) with optimizations..."
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) $(MAIN_PATH)

# Install the binary to $(INSTALL_PATH)
install: build-release
	@echo "Installing $(BINARY_NAME) to $(INSTALL_PATH)..."
	@mkdir -p $(INSTALL_PATH)
	@cp $(BINARY_NAME) $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "Installed to $(INSTALL_PATH)/$(BINARY_NAME)"

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -race -v ./...

# Run tests with coverage (excluding mock files)
cover:
	@echo "Running tests with coverage..."
	$(GOTEST) -race -v -coverprofile=coverage.raw.out ./...
	@grep -v '_mocks' coverage.raw.out > coverage.out
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"
	open coverage.html

# Run go vet
vet:
	@echo "Running go vet..."
	$(GOVET) ./...

# Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) -s -w .

# Check if code is formatted
fmt-check:
	@echo "Checking code formatting..."
	@test -z "$$($(GOFMT) -l .)" || (echo "Code is not formatted. Run 'make fmt'" && exit 1)

# Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	$(GOMOD) tidy

# Generate mocks
generate:
	@echo "Generating mocks..."
	$(GOCMD) generate ./...

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME)
	@rm -f coverage.out coverage.raw.out coverage.html
	@echo "Clean complete"

# Run all checks (fmt, vet, test)
check: fmt-check vet test

# Help target
help:
	@echo "Available targets:"
	@echo "  make build          - Build the binary"
	@echo "  make build-release  - Build optimized binary"
	@echo "  make install        - Install binary to ~/.local/bin"
	@echo "  make test           - Run tests"
	@echo "  make cover          - Run tests with coverage report"
	@echo "  make vet            - Run go vet"
	@echo "  make fmt            - Format code"
	@echo "  make fmt-check      - Check if code is formatted"
	@echo "  make tidy           - Tidy dependencies"
	@echo "  make generate       - Generate mocks"
	@echo "  make check          - Run fmt-check, vet, and test"
	@echo "  make clean          - Remove build artifacts"
	@echo "  make help           - Show this help message"
