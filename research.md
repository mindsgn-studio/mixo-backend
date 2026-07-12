# Real-Time Audio Streaming Approaches — Research

## Overview

This document compares different approaches for real-time audio streaming in a Go internet radio server. Our current approach uses FFmpeg with the `-re` flag for real-time pacing.

---

## Approach 1: FFmpeg Subprocess with `-re` (Our Current Fix)

**How it works:**
- Spawn FFmpeg as a subprocess per song
- Use `-re` flag to output at native frame rate (real-time pacing)
- Pipe decoded MP3 data to stdout → read in Go → broadcast to listeners via fan-out broadcaster

**Implementation:**
```go
cmd := exec.Command("ffmpeg", "-re", "-i", filePath, "-f", "mp3", "-acodec", "libmp3lame", "-ar", "44100", "-ac", "2", "-b:a", "128k", "-")
```

**Pros:**
- Handles virtually every audio format (MP3, AAC, FLAC, OGG, WAV, etc.)
- Built-in resampling, normalization, format conversion
- Battle-tested, production-proven (used by Icecast, Shoutcast, major streaming platforms)
- Hardware acceleration support (NVENC, QuickSync, etc.)
- `-re` flag handles pacing natively — no custom throttle logic needed
- Handles VBR (variable bitrate) files correctly

**Cons:**
- External dependency — FFmpeg must be installed on the system
- One FFmpeg process per song — ~20-50MB RAM per process
- Subprocess overhead — fork/exec for each track transition
- Not pure Go — can't cross-compile easily, harder to test in CI
- If FFmpeg crashes, playback stops silently

**When to use:**
- You need to support many audio formats
- You want production-proven reliability
- You're OK with the external dependency
- You need format conversion or audio processing
- **Best for: Most internet radio projects (including ours)**

**Verdict: BEST for our use case.** Our fix (adding `-re`) is the correct approach.

---

## Approach 2: Pure Go HTTP Streaming (Fan-Out Broadcaster)

**How it works:**
- Pre-read entire audio file into memory (or stream from disk)
- Use a fan-out broadcaster pattern to send chunks to all connected clients
- Pace the output using `time.Ticker` based on expected bitrate and chunk size
- No external dependencies — pure Go

**Implementation (reference: icelain/radio):**
```go
type ConnectionPool struct {
    bufferChannelMap map[chan []byte]struct{}
    mu               sync.Mutex
}

func stream(connPool *ConnectionPool, content []byte) {
    for {
        tempfile := bytes.NewReader(content)
        ticker := time.NewTicker(time.Millisecond * 250) // pace by bitrate
        for range ticker.C {
            buf := make([]byte, 4096)
            n, _ := tempfile.Read(buf)
            if n == 0 {
                break // end of file, restart
            }
            connPool.Broadcast(buf[:n])
        }
    }
}
```

**Pros:**
- Zero external dependencies — pure Go, single binary
- Easy to cross-compile and deploy
- Full control over pacing logic
- Low memory footprint (no FFmpeg processes)
- Simple architecture — easy to understand and debug
- Fast startup — no subprocess overhead
- Existing Go libraries: `gopxl/beep` (MP3/WAV/FLAC/OGG decoder), `minimp3`, `malgo`

**Cons:**
- Must handle format detection and decoding yourself
- Limited format support compared to FFmpeg (MP3/WAV/FLAC/OGG, but not AAC/WMA/etc.)
- Custom pacing logic — easy to get wrong (as we discovered with the 10ms sleep bug)
- VBR files require bitrate calculation or header parsing
- No built-in audio processing (normalization, resampling, effects)
- More code to maintain and test
- Decoder bugs can cause audio glitches

**When to use:**
- You only need MP3/WAV/FLAC/OGG support
- You want a single binary with no external dependencies
- You're building a small/personal project
- You want full control over the streaming pipeline
- **Best for: Small personal radio stations, embedded systems, single-binary deployments**

**Verdict: GOOD alternative if you want zero dependencies.** More work to get right, but eliminates FFmpeg.

---

## Approach 3: HLS (HTTP Live Streaming)

**How it works:**
- Split audio into small segments (2-10 seconds each, typically `.ts` or `.m4s` files)
- Generate an M3U8 playlist file listing all segments
- Serve playlist and segments over HTTP
- Client (browser/player) requests playlist, then fetches segments sequentially
- Playlist updates periodically for live content

**Implementation:**
```go
// Generate M3U8 playlist
#EXTM3U
#EXT-X-TARGETDURATION:10
#EXT-X-MEDIA-SEQUENCE:0
#EXTINF:10.0,
segment_000.ts
#EXTINF:10.0,
segment_001.ts
#EXTINF:8.5,
segment_002.ts
#EXT-X-ENDLIST
```

**Pros:**
- Works in all modern browsers natively (via `<audio>` or `<video>` tag)
- CDN-friendly — segments are cacheable static files
- Adaptive bitrate streaming (ABR) — client can switch quality
- Handles network interruptions gracefully (buffering)
- Widely supported by players (VLC, ffplay, iOS, Android, etc.)
- No persistent connections — stateless HTTP
- Can pre-generate segments for on-demand content

**Cons:**
- Higher latency — 2-30 seconds depending on segment duration
- Not truly "real-time" — there's always a buffer delay
- Requires segment generation pipeline (FFmpeg or similar)
- More complex infrastructure — segment storage, playlist management
- Clients must buffer before playback starts
- Not suitable for live DJ sets or synchronized playback
- Requires CORS configuration for cross-origin segment requests
- Playlist must be regenerated for live content

**When to use:**
- You're building a podcast/on-demand platform
- You need CDN caching and global distribution
- You want browser-native playback without plugins
- Latency of 2-10 seconds is acceptable
- You need adaptive bitrate streaming
- **Best for: Podcast platforms, on-demand music services, large-scale distribution**

**Verdict: WORST for our use case.** We need synchronized real-time playback (all listeners hear the same thing simultaneously). HLS introduces unacceptable latency for internet radio.

---

## Approach 4 (Bonus): libmpg123 / Native MP3 Decoding

**How it works:**
- Use `libmpg123` C library (via CGo) to decode MP3 files directly
- Or use pure Go decoders like `gopxl/beep`, `minimp3`, `tosone/minimp3`
- Output raw PCM data, re-encode to MP3 for streaming, or stream raw
- Pace output based on sample rate and bitrate

**Implementation (libmpg123 via CGo):**
```go
// #cgo LDFLAGS: -lmpg123
// #include <mpg123.h>
import "C"

handle := C.mpg123_new(nil, nil)
C.mpg123_open(handle, cPath)
// decode frames...
```

**Implementation (pure Go - gopxl/beep):**
```go
import "github.com/gopxl/beep/mp3"

f, _ := os.Open("song.mp3")
streamer, format, _ := mp3.Decode(f)
// streamer implements beep.Streamer interface
// format.SampleRate, format.NumChannels, format.Precision
```

**Pros:**
- `libmpg123` is extremely fast — 500x realtime on modern hardware
- `libmpg123` is battle-tested (used by mpg123 player since 1999)
- Pure Go decoders (`beep`) — no CGo, single binary
- Full control over decoding and output
- Lower resource usage than FFmpeg
- Can decode frame-by-frame for precise control

**Cons:**
- `libmpg123` requires CGo — harder to cross-compile
- `libmpg123` is MP3-only — need separate decoders for other formats
- Pure Go decoders (`beep`) are less mature than FFmpeg
- `beep` is archived — migrated to `gopxl/beep` (less active)
- Must handle format detection, header parsing, and bitrate calculation
- No built-in audio processing
- More complex than FFmpeg subprocess approach

**When to use:**
- You only need MP3 support and want maximum decoding speed
- You're building a high-performance server with thousands of concurrent streams
- You want to minimize CPU usage
- You can accept CGo dependency (`libmpg123`)
- **Best for: High-performance MP3-only servers, resource-constrained environments**

**Verdict: GOOD for MP3-only use cases.** But FFmpeg is more versatile and easier to use.

---

## Summary Comparison

| Approach | Latency | Format Support | Dependencies | Complexity | Best For |
|----------|---------|---------------|-------------|-----------|----------|
| **FFmpeg + `-re`** | Real-time | All formats | FFmpeg binary | Low | Internet radio (our use case) |
| **Pure Go HTTP** | Real-time | MP3/WAV/FLAC/OGG | None | Medium | Small projects, single binary |
| **HLS** | 2-30s | All formats | Segment storage | High | Podcasts, on-demand, CDN |
| **libmpg123** | Real-time | MP3 only | libmpg123 (CGo) | Medium | High-perf MP3 servers |

## Recommendation

**Keep FFmpeg with `-re` flag.** It's the right tool for internet radio:
- Handles all audio formats
- `-re` handles real-time pacing correctly
- Battle-tested at scale
- Simple to implement and debug

If you later need to eliminate the FFmpeg dependency, **Approach 2 (Pure Go)** with `gopxl/beep` is the next best option for MP3/WAV/FLAC/OGG support.
