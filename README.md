# Hackday Radio - Internet Radio Streaming Server

A real-time internet radio streaming server built with Go that broadcasts a single continuous audio stream to all connected listeners.

## Features

- **Single continuous audio stream** - All users hear the same playback in real-time
- **No seeking or per-user control** - Traditional radio-style broadcast
- **MP3 streaming over HTTP** - Chunked transfer encoding for real-time delivery
- **Fan-out broadcaster pattern** - Efficiently serves multiple clients
- **FIFO queue management** - Songs play in order they were added
- **Music crawler** - Auto-scans directories for MP3s, extracts metadata (title, artist, album, cover art)
- **Queue worker** - Automatically maintains 24h queue target by adding random songs
- **HTMX admin interface** - Web-based UI for managing songs and queue
- **Cover art support** - Extracts and displays album cover art
- **Admin API** - RESTful endpoints for managing songs and queue
- **SQLite persistence** - Stores songs, queue, history, and state
- **FFmpeg integration** - Ensures consistent audio format
- **HLS output** - The radio stream is also encoded to HLS so it can be watched
  alongside live video (see [HLS output](#hls-output))
- **Password-protected admin** - `/admin` and its API can be locked with HTTP
  Basic Auth via `ADMIN_PASSWORD` (open by default)
- **Slow client handling** - Automatically drops clients that can't keep up

## Requirements

- Go 1.26 or higher
- FFmpeg installed and available in PATH
- SQLite (included via Go driver)

## Quick Start

### Using Scripts (Recommended)

```bash
# Clone the repository
git clone https://github.com/mindsgn/hackday-radio.git
cd mixo-backend

# Run tests
./scripts/test.sh

# Build the binary
./scripts/build.sh

# Run locally
./scripts/run.sh

# Run deploy
./scripts/deploy.sh
```

### Manual Setup

```bash
# Clone the repository
git clone https://github.com/mindsgn/hackday-radio.git
cd mixo-backend

# Install dependencies
go mod download

# Ensure FFmpeg is installed
ffmpeg -version

# Create .env file (or copy from example)
cp .env.example .env

# Create songs directory and add MP3 files
mkdir -p ../songs
# Add your MP3 files to ../songs/

# Run the server
go run cmd/server/main.go
```

## Configuration

Create a `.env` file in the project root:

```env
PORT=8080
SONG_DIR=../songs
DB_PATH=./radio.db
STREAM_TIMEOUT=30
CRAWL_DIRS=../songs
QUEUE_HOURS=24
CRAWL_INTERVAL=60

# Password-protect the admin interface (HTTP Basic Auth on /admin).
# Leave ADMIN_PASSWORD empty to keep it open.
ADMIN_USERNAME=admin
ADMIN_PASSWORD=

# HLS output for the radio stream (see "HLS output" below).
FFMPEG_BIN=ffmpeg
HLS_DIR=/var/www/html/hls
HLS_STREAM_ID=radio
HLS_SEGMENT_TIME=2
HLS_PLAYLIST_SIZE=6
```

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | HTTP server port | `8080` |
| `SONG_DIR` | Directory for uploaded songs | `./songs` |
| `DB_PATH` | SQLite database file path | `./radio.db` |
| `STREAM_TIMEOUT` | Timeout for slow clients (seconds) | `5` |
| `CRAWL_DIRS` | Comma-separated directories to scan for MP3s | `../songs` |
| `QUEUE_HOURS` | Target queue duration in hours | `24` |
| `CRAWL_INTERVAL` | How often to refill queue (minutes) | `60` |
| `ADMIN_USERNAME` | Admin username (Basic Auth) | `admin` |
| `ADMIN_PASSWORD` | Admin password; empty = admin open | `""` |
| `FFMPEG_BIN` | FFmpeg binary used for HLS encoding | `ffmpeg` |
| `HLS_DIR` | HLS output directory | `/var/www/html/hls` |
| `HLS_STREAM_ID` | Subdirectory for the radio playlist | `radio` |
| `HLS_SEGMENT_TIME` | HLS segment length in seconds | `2` |
| `HLS_PLAYLIST_SIZE` | Segments kept in the playlist | `6` |

## HLS output

The radio stream is continuously encoded to HLS by FFmpeg. The playlist is
written to:

```
$HLS_DIR/$HLS_STREAM_ID/index.m3u8   # default: /var/www/html/hls/radio/index.m3u8
```

On Ubuntu the default `HLS_DIR` (`/var/www/html/hls`) is the web root that
nginx serves directly at `/hls/`. This directory is **shared** with the video
streaming server (**a11-video-stream**), which writes its own playlists into
`/hls/<stream-id>/` there. Serving both from the same root means the radio page
can show either the live video stream (from a11-video-stream) or the radio
music (from mixo-backend) at `/hls/radio/index.m3u8`.

Ensure the directory exists and is writable by the user running the server:

```bash
sudo mkdir -p /var/www/html/hls
sudo chown -R www-data:www-data /var/www/html/hls   # or the app user
```

The radio HLS playlist is also served by this server at `/hls/radio/index.m3u8`
for direct access.

## Usage

### Start the Server

```bash
# Development mode (with live reload)
./scripts/run.sh

# Or manually
go run cmd/server/main.go
```

The server will:
1. Scan `CRAWL_DIRS` for MP3 files and add them to the database
2. Fill the queue to `QUEUE_HOURS` target
3. Start the playback engine and stream broadcaster
4. Begin accepting connections

### Endpoints

| Endpoint | Description |
|----------|-------------|
| `http://localhost:8080/stream` | Audio stream (use with any media player) |
| `http://localhost:8080/hls/` | Radio HLS playlist + segments (`/hls/radio/index.m3u8`) |
| `http://localhost:8080/admin` | HTMX admin interface (Basic Auth when `ADMIN_PASSWORD` is set) |
| `http://localhost:8080/api/*` | REST API endpoints |
| `http://localhost:8080/health` | Health check |

### Listen to the Stream

```bash
# Using ffplay
ffplay http://localhost:8080/stream

# Using VLC
vlc http://localhost:8080/stream

# Using mpv
mpv http://localhost:8080/stream

# Or open in browser
open http://localhost:8080/stream
```

### API Examples

```bash
# List songs
curl http://localhost:8080/api/songs

# Add song to queue
curl -X POST http://localhost:8080/api/queue/1

# Get current queue
curl http://localhost:8080/api/queue

# Get now playing
curl http://localhost:8080/api/now-playing

# Get playback history
curl http://localhost:8080/api/history?limit=50

# Upload a song
curl -X POST http://localhost:8080/api/upload \
  -F "file=@/path/to/song.mp3" \
  -F "title=Song Title" \
  -F "artist=Artist Name"
```

## Project Structure

```
mixo-backend/
├── cmd/server/
│   └── main.go                    # Application entry point
├── internal/
│   ├── admin/
│   │   ├── auth.go                # HTTP Basic Auth for /admin
│   │   ├── handler.go             # Admin API handlers
│   │   ├── routes.go              # Route registration
│   │   └── templates.go           # HTML templates for HTMX
│   ├── config/
│   │   ├── config.go              # Configuration loading
│   │   └── config_test.go         # Config tests
│   ├── crawler/
│   │   ├── crawler.go             # Music directory scanner
│   │   └── crawler_test.go        # Crawler tests
│   ├── database/
│   │   ├── migrations.go          # Database schema
│   │   ├── migrations_test.go     # Migration tests
│   │   └── sqlite.go              # SQLite connection
│   ├── hls/
│   │   ├── hls.go                 # HLS encoder (radio stream → FFmpeg → HLS)
│   │   └── hls_test.go            # HLS encoder tests
│   ├── playback/
│   │   ├── engine.go              # Playback engine
│   │   └── ffmpeg.go              # FFmpeg integration
│   ├── queue/
│   │   ├── manager.go             # Queue management
│   │   └── manager_test.go        # Queue tests
│   ├── stream/
│   │   ├── broadcaster.go         # Fan-out broadcaster (+ HLS sink)
│   │   └── handler.go             # HTTP stream handler
│   └── worker/
│       ├── worker.go              # Queue maintenance worker
│       └── worker_test.go         # Worker tests
├── scripts/
│   ├── test.sh                    # Run tests
│   ├── build.sh                   # Build binary
│   └── run.sh                     # Run locally
├── .env                           # Environment variables (not committed)
├── .env.example                   # Example environment file
├── AGENTS.md                      # Development guidelines
├── go.mod                         # Go module definition
└── go.sum                         # Go dependencies
```

## Workers

### Music Crawler

Scans directories for MP3 files and:
- Extracts metadata (title, artist, album, cover art)
- Adds new songs to the database
- Removes songs with missing files from the database

Runs on server startup.

### Queue Worker

Maintains the queue by:
- Checking queue duration every `CRAWL_INTERVAL` minutes
- Adding random songs when queue is below `QUEUE_HOURS` target
- Ensuring continuous playback without gaps

Runs continuously in the background.

## Scripts

| Script | Description |
|--------|-------------|
| `./scripts/test.sh` | Run all tests with race detection |
| `./scripts/build.sh` | Build the binary to `./bin/radio-server` |
| `./scripts/run.sh` | Run the server locally |

## CI/CD Setup

This project uses GitHub Actions for continuous integration and deployment.

### Workflows

| Workflow | Trigger | Description |
|----------|---------|-------------|
| **Staging** | Push to `staging` | Lints, builds, tests, creates PR to `main` |
| **Deploy** | Push to `main` | Tests, builds, tags, deploys to server, restarts PM2 |

### Required GitHub Secrets

| Secret | Description | Example |
|--------|-------------|---------|
| `DEPLOY_HOST` | Server IP or hostname | `192.168.0.141` |
| `DEPLOY_USERNAME` | SSH username on server | `deploy` |
| `DEPLOY_SSH_KEY` | Private SSH key for authentication | `-----BEGIN OPENSSH PRIVATE KEY-----...` |
| `DEPLOY_PORT` | SSH port (optional, defaults to 22) | `22` |
| `DEPLOY_PATH` | Absolute path to app directory on server | `/home/deploy/mixo-backend` |

### Server Prerequisites (Ubuntu)

```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install Go
wget https://go.dev/dl/go1.26.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.26.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Install FFmpeg
sudo apt install ffmpeg -y

# Install Node.js (for PM2)
curl -fsSL https://deb.nodesource.com/setup_24.x | sudo -E bash -
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
# Install Homebrew (if not not installed)
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

### PM2 Commands

```bash
# Start the app
pm2 start ./bin/radio-server --name mixo-backend

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

Current version: v0.2.0

## License

MIT License
