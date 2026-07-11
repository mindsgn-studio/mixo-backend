#!/bin/bash
set -e

echo "=========================================="
echo "  Running Tests"
echo "=========================================="
echo ""

# Change to project root
cd "$(dirname "$0")/.."

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed"
    exit 1
fi

echo "Go version: $(go version)"
echo ""

# Run tests with race detection
echo "Running tests with race detection..."
echo "------------------------------------------"
go test -v -race ./...

echo ""
echo "=========================================="
echo "  Tests completed successfully!"
echo "=========================================="
