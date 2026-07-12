#!/bin/bash
set -e

echo "=========================================="
echo "  Building Radio Server"
echo "=========================================="
echo ""

# Change to project root
cd "$(dirname "$0")/.."

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed"
    exit 1
fi

# Create bin directory if it doesn't exist
mkdir -p bin

# Build the binary
echo "Building binary..."
echo "------------------------------------------"
go build -o bin/radio-server ./cmd/server/

echo ""
echo "Build successful!"
echo "Binary: $(pwd)/bin/radio-server"
echo ""

# Make binary executable
chmod +x bin/radio-server

echo "=========================================="
echo "  Build completed successfully!"
echo "=========================================="
