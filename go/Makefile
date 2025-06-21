# Calculator Makefile

# Default target
.DEFAULT_GOAL := build

# Variables
BINARY_NAME := calc
TEST_DIR := test

# Build the calculator
build:
	go build -o $(BINARY_NAME) .

# Run the calculator with arguments
# Usage: make run ARGS="1 3 /"
run: build
	./$(BINARY_NAME) $(ARGS)

# Run all tests
test: test-unit test-integration

# Run only unit tests (fast)
test-unit:
	go test -v .

# Run only integration tests (slower)
test-integration:
	go test -tags=integration -v ./$(TEST_DIR)

# Run with coverage
test-cover:
	go test -cover .

# Run integration tests with coverage
test-integration-cover:
	go test -tags=integration -cover ./$(TEST_DIR)

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME)

# Show help
help:
	@echo "Available commands:"
	@echo "  build              - Build the calculator binary"
	@echo "  run                - Build and run the calculator (use ARGS='...' to pass arguments)"
	@echo "  test               - Run all tests (unit + integration)"
	@echo "  test-unit          - Run only unit tests (fast)"
	@echo "  test-integration   - Run only integration tests (slower)"
	@echo "  test-cover         - Run unit tests with coverage"
	@echo "  test-integration-cover - Run integration tests with coverage"
	@echo "  clean              - Remove build artifacts"
	@echo "  help               - Show this help message"
	@echo ""
	@echo "Examples:"
	@echo "  make build"
	@echo "  make run ARGS='1 3 /'"
	@echo "  make run ARGS='-p 6 1 3 /'"
	@echo "  make test          # Run all tests"
	@echo "  make test-unit     # Only unit tests"
	@echo "  make test-integration # Only integration tests"

.PHONY: build run test test-unit test-integration test-cover test-integration-cover clean help