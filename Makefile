# ProofPix Mono-repo Makefile

# Build the API service binary
build:
	@echo "Building API service..."
	go build -o bin/api cmd/api/main.go

# Run the API service
run:
	@echo "Running API service..."
	./bin/api

# Run with environment variables
run-local:
	@echo "Running API service locally with environment variables..."
	@set PROJECT_ID=make-connection-464709 && set FIREBASE_PROJECT_ID=make-connection-464709 && .\bin\api

# Test authentication endpoints
test-auth:
	@echo "Testing authentication endpoints..."
	powershell -ExecutionPolicy Bypass -File test-auth.ps1

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/

# Create bin directory if it doesn't exist
bin:
	mkdir -p bin

# Build with bin directory creation
build-with-bin: bin build

# Help target to show available commands
help:
	@echo "Available targets:"
	@echo "  build     - Build the API service binary"
	@echo "  run       - Run the compiled API binary"
	@echo "  clean     - Clean build artifacts"
	@echo "  help      - Show this help message"

.PHONY: build run clean help bin build-with-bin 