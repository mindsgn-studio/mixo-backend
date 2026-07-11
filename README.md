# Go Internet Radio Streaming Server

A real-time internet radio streaming server built with Go that broadcasts a single continuous audio stream to all connected listeners.

## Features

- **Single continuous audio stream** - All users hear the same playback in real-time
- **No seeking or per-user control** - Traditional radio-style broadcast
- **MP3 streaming over HTTP** - Chunked transfer encoding for real-time delivery
- **Fan-out broadcaster pattern** - Efficiently serves multiple clients
- **FIFO queue management** - Songs play in order they were added
- **Admin API** - RESTful endpoints for managing songs and queue
- **SQLite persistence** - Stores songs, queue, history, and state
- **FFmpeg integration** - Ensures consistent audio format
- **Slow client handling** - Automatically drops clients that can't keep up

## Requirements

- Go 1.21 or higher
- FFmpeg installed and available in PATH
- SQLite (included via Go driver)

## Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd radio/backend
```

2. Install dependencies:
```bash
go mod download
```

3. Ensure FFmpeg is installed:
```bash
ffmpeg -version
```

## Configuration

Create a `.env` file in the `backend` directory:

```env
PORT=8080
SONG_DIR=../songs
DB_PATH=./radio.db
STREAM_TIMEOUT=5
```

- `PORT`: HTTP server port (default: 8080)
- `SONG_DIR`: Directory containing MP3 files
- `DB_PATH`: SQLite database file path
- `STREAM_TIMEOUT`: Timeout for slow clients in seconds

## Usage

### Start the server

```bash
cd backend
go run cmd/server/main.go
```

The server will start on the configured port with the following endpoints:
- Stream: `http://localhost:8080/stream`
- Admin API: `http://localhost:8080/api/*`
- Health check: `http://localhost:8080/health`

### Add songs

```bash
curl -X POST http://localhost:8080/api/songs \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Song Title",
    "artist": "Artist Name",
    "duration": 180,
    "location": "/path/to/song.mp3"
  }'
```

### List songs

```bash
curl http://localhost:8080/api/songs
```

### Add song to queue

```bash
curl -X POST http://localhost:8080/api/queue/{song_id}
```

### Get current queue

```bash
curl http://localhost:8080/api/queue
```

### Get now playing

```bash
curl http://localhost:8080/api/now-playing
```

### Get playback history

```bash
curl http://localhost:8080/api/history?limit=50
```

### Listen to the stream

Use any media player that supports HTTP streaming:
```bash
ffplay http://localhost:8080/stream
```

Or open in a browser:
```
http://localhost:8080/stream
```

## Project Structure

```
radio/
├── backend/
│   ├── cmd/server/
│   │   └── main.go              # Application entry point
│   ├── internal/
│   │   ├── admin/
│   │   │   ├── handler.go      # Admin API handlers
│   │   │   └── routes.go       # Route registration
│   │   ├── config/
│   │   │   └── config.go       # Configuration loading
│   │   ├── database/
│   │   │   ├── migrations.go   # Database schema
│   │   │   └── sqlite.go       # SQLite connection
│   │   ├── playback/
│   │   │   ├── engine.go       # Playback engine
│   │   │   └── ffmpeg.go       # FFmpeg integration
│   │   ├── queue/
│   │   │   └── manager.go      # Queue management
│   │   └── stream/
│   │       ├── broadcaster.go   # Fan-out broadcaster
│   │       └── handler.go       # HTTP stream handler
│   ├── .env                    # Environment variables
│   ├── go.mod                  # Go module definition
│   ├── go.sum                  # Go dependencies
│   └── article.md              # Core concepts documentation
└── songs/                      # MP3 files directory
```

## API Endpoints

### Songs
- `POST /api/songs` - Add a new song
- `GET /api/songs` - List all songs
- `DELETE /api/songs/:id` - Delete a song

### Queue
- `POST /api/queue/:songId` - Add song to queue
- `GET /api/queue` - Get current queue
- `DELETE /api/queue/:id` - Remove from queue

### Status
- `GET /api/now-playing` - Get currently playing song
- `GET /api/history` - Get playback history

### Health
- `GET /health` - Health check

## Testing

Run unit tests:
```bash
cd backend
go test ./...
```

Run tests with coverage:
```bash
go test -cover ./...
```

## Core Concepts

For detailed explanations of the core concepts and patterns used in this project, see [article.md](backend/article.md).

Topics covered:
- Fan-out broadcaster pattern
- SQLite for persistence
- FFmpeg for audio streaming
- Chunked HTTP transfer encoding
- FIFO queue management
- Slow client detection
- Real-time throttling

## CI/CD Setup

This project uses GitHub Actions for continuous integration and deployment.

### Workflows

| Workflow | Trigger | Description |
|----------|---------|-------------|
| **Staging to Main** | Push to `staging` | Lints, builds, tests, and creates a PR to `main` |
| **Deploy to Production** | Push to `main` | Tests, builds, tags, deploys to server, restarts PM2 |

### Required GitHub Secrets

Go to your repository → **Settings** → **Secrets and variables** → **Actions** and add:

| Secret | Description | Example |
|--------|-------------|---------|
| `DEPLOY_HOST` | Server IP or hostname | `192.168.1.100` |
| `DEPLOY_USERNAME` | SSH username on server | `deploy` |
| `DEPLOY_SSH_KEY` | Private SSH key for authentication | `-----BEGIN OPENSSH PRIVATE KEY-----...` |
| `DEPLOY_PORT` | SSH port (optional, defaults to 22) | `22` |
| `DEPLOY_PATH` | Absolute path to app directory on server | `/home/deploy/mixo-backend` |

### Server Prerequisites (Ubuntu)

```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install Go
wget https://go.dev/dl/go1.23.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.23.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Install FFmpeg
sudo apt install ffmpeg -y

# Install Node.js (for PM2)
curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt install -y nodejs

# Install PM2 globally
sudo npm install pm2 -g

# Setup PM2 to start on boot
pm2 startup
sudo env PATH=$PATH:/usr/local/bin pm2 startup systemd -u $USER --hp $HOME

# Create app directory
mkdir -p /home/deploy/mixo-backend
```

### Server Prerequisites (macOS)

```bash
# Install Homebrew (if not installed)
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Install Go
brew install go

# Install FFmpeg
brew install ffmpeg

# Install Node.js
brew install node

# Install PM2
npm install pm2 -g

# Create app directory
mkdir -p ~/mixo-backend
```

### Generating SSH Key for Deployment

On your **local machine**:

```bash
# Generate a dedicated deploy key
ssh-keygen -t ed25519 -C "github-deploy" -f ~/.ssh/deploy_key -N ""

# Copy the public key to your server
ssh-copy-id -i ~/.ssh/deploy_key.pub user@your-server-ip
```

Then add the **private key** contents to the `DEPLOY_SSH_KEY` secret in GitHub:

```bash
# Print the private key to copy into GitHub Secrets
cat ~/.ssh/deploy_key
```

### Running Locally

**Ubuntu / macOS:**

```bash
# Clone the repo
git clone https://github.com/mindsgn-studio/mixo-backend.git
cd mixo-backend

# Install dependencies
go mod download

# Create .env file
cp .env.example .env  # or create manually

# Run the server
go run cmd/server/main.go

# Run tests
go test ./...

# Build the binary
go build -o radio-server ./cmd/server/

# Run with PM2
pm2 start ./radio-server --name mixo-backend
pm2 logs mixo-backend
```

### PM2 Commands

```bash
# Start the app
pm2 start ./radio-server --name mixo-backend

# Stop the app
pm2 stop mixo-backend

# Restart the app
pm2 restart mixo-backend

# View logs
pm2 logs mixo-backend

# Monitor processes
pm2 monit

# Save process list (for auto-restart)
pm2 save

# List all processes
pm2 list
```

## Version

Current version: v0.1.0

## License

MIT License
