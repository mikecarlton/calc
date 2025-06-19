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

# Run tests
test:
	cd $(TEST_DIR) && go test -v

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME)

# Show help
help:
	@echo "Available commands:"
	@echo "  build  - Build the calculator binary"
	@echo "  run    - Build and run the calculator (use ARGS='...' to pass arguments)"
	@echo "  test   - Run all tests"
	@echo "  clean  - Remove build artifacts"
	@echo "  help   - Show this help message"
	@echo ""
	@echo "Examples:"
	@echo "  make build"
	@echo "  make run ARGS='1 3 /'"
	@echo "  make run ARGS='-p 6 1 3 /'"
	@echo "  make test"

.PHONY: build run test clean help