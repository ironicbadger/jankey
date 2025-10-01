.PHONY: build install clean test help

BINARY_NAME=jankey
INSTALL_PATH=/usr/local/bin

# Default target
help:
	@echo "Jankey - Tailscale Auth Key Generator - Build Commands"
	@echo ""
	@echo "Available targets:"
	@echo "  make build     - Build the binary"
	@echo "  make install   - Build and install to $(INSTALL_PATH)"
	@echo "  make clean     - Remove built binary"
	@echo "  make test      - Run tests"
	@echo "  make run       - Build and run with --help"
	@echo ""

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@go build -o $(BINARY_NAME)
	@echo "✓ Build complete: ./$(BINARY_NAME)"

# Install the binary
install: build
	@echo "Installing $(BINARY_NAME) to $(INSTALL_PATH)..."
	@sudo mv $(BINARY_NAME) $(INSTALL_PATH)/
	@echo "✓ Installed: $(INSTALL_PATH)/$(BINARY_NAME)"

# Clean built artifacts
clean:
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME)
	@echo "✓ Clean complete"

# Run tests
test:
	@echo "Running tests..."
	@go test ./...
	@echo "✓ Tests complete"

# Build and run
run: build
	@./$(BINARY_NAME) --help

# Development dependencies
deps:
	@echo "Installing dependencies..."
	@go mod download
	@echo "✓ Dependencies installed"
