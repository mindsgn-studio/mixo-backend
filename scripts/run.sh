#!/bin/bash
set -e

echo "=========================================="
echo "  Running Radio Server (Development)"
echo "=========================================="
echo ""

# Change to project root
cd "$(dirname "$0")/.."

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed"
    exit 1
fi

# Check if .env exists
if [ ! -f .env ]; then
    echo "Warning: .env file not found"
    echo "Creating .env from .env.example..."
    if [ -f .env.example ]; then
        cp .env.example .env
    else
        cat > .env << 'EOF'
PORT=8080
SONG_DIR=../songs
DB_PATH=./radio.db
STREAM_TIMEOUT=30
CRAWL_DIRS=../songs
QUEUE_HOURS=24
CRAWL_INTERVAL=60
EOF
    fi
fi

# Create songs directory if it doesn't exist
if [ ! -d "../songs" ]; then
    echo "Creating songs directory..."
    mkdir -p ../songs
fi

# Check if FFmpeg is installed
if ! command -v ffmpeg &> /dev/null; then
    echo "Warning: FFmpeg is not installed"
    echo "Audio playback may not work correctly"
    echo ""
fi

echo "Starting server..."
echo "------------------------------------------"
echo "Stream: http://localhost:8080/stream"
echo "Admin:  http://localhost:8080/admin"
echo "API:    http://localhost:8080/api"
echo "Health: http://localhost:8080/health"
echo "------------------------------------------"
echo ""
echo "Press Ctrl+C to stop"
echo ""

# Run the server
go run cmd/server/main.go
