#!/bin/bash
set -e

APP_NAME="mixo-backend"
BINARY="bin/radio-server"

echo "=========================================="
echo "  Deploying $APP_NAME"
echo "=========================================="
echo ""

# Change to project root
cd "$(dirname "$0")/.."

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed"
    exit 1
fi

# Step 1: Install dependencies
echo "------------------------------------------"
echo "Step 1/5: Installing dependencies..."
echo "------------------------------------------"
go mod download
echo ""

# Step 2: Run tests
echo "------------------------------------------"
echo "Step 2/5: Running tests..."
echo "------------------------------------------"
go test -v -race ./...
echo ""

# Step 3: Build binary
echo "------------------------------------------"
echo "Step 3/5: Building binary..."
echo "------------------------------------------"
mkdir -p bin
go build -o "$BINARY" ./cmd/server/
chmod +x "$BINARY"
echo "Build successful: $(pwd)/$BINARY"
echo ""

# Step 4: Deploy and restart via pm2
echo "------------------------------------------"
echo "Step 4/5: Managing pm2 process..."
echo "------------------------------------------"
if pm2 describe "$APP_NAME" > /dev/null 2>&1; then
    echo "Process '$APP_NAME' exists, restarting..."
    pm2 restart "$APP_NAME"
else
    echo "Process '$APP_NAME' not found, starting new..."
    pm2 start "./$BINARY" --name "$APP_NAME"
fi
echo ""

# Step 5: Save pm2 process list
echo "------------------------------------------"
echo "Step 5/5: Saving pm2 process list..."
echo "------------------------------------------"
pm2 save
echo ""

echo "=========================================="
echo "  Deployment complete!"
echo "=========================================="
pm2 list
