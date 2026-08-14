// Package hls converts the continuous radio audio stream into an HLS live
// playlist. Files are written to a shared web root so nginx can serve them
// alongside the video streaming server's HLS output.
package hls

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
)

// Config describes how to build the HLS encoder for the radio stream.
type Config struct {
	Bin          string
	HLSDir       string
	StreamID     string
	SegmentTime  int
	PlaylistSize int
	Stderr       io.Writer
}

// Args returns the exact command-line arguments used to convert the radio MP3
// stream (read from stdin) into HLS. It is a pure function so it can be unit
// tested without launching FFmpeg.
func (c Config) Args() []string {
	output := filepath.Join(c.HLSDir, c.StreamID, "index.m3u8")
	return []string{
		"-nostdin",
		"-hide_banner",
		"-loglevel", "error",
		"-i", "pipe:0",
		"-vn",
		"-acodec", "aac",
		"-ar", "44100",
		"-ac", "2",
		"-b:a", "128k",
		"-f", "hls",
		"-hls_time", strconv.Itoa(c.SegmentTime),
		"-hls_list_size", strconv.Itoa(c.PlaylistSize),
		"-hls_flags", "delete_segments",
		output,
	}
}

// Encoder runs a single long-lived FFmpeg process that converts the radio
// stream into HLS playlists and segments written to HLSDir/StreamID. The
// encoder must be fed the same MP3 chunks the HTTP broadcaster sends out.
type Encoder struct {
	cfg Config
	mu  sync.Mutex
	cmd *exec.Cmd
	in  io.WriteCloser
}

// New builds an Encoder, applying safe defaults for unset fields.
func New(cfg Config) *Encoder {
	if cfg.Bin == "" {
		cfg.Bin = "ffmpeg"
	}
	if cfg.HLSDir == "" {
		cfg.HLSDir = "/var/www/html/hls"
	}
	if cfg.StreamID == "" {
		cfg.StreamID = "radio"
	}
	if cfg.SegmentTime <= 0 {
		cfg.SegmentTime = 2
	}
	if cfg.PlaylistSize <= 0 {
		cfg.PlaylistSize = 6
	}
	return &Encoder{cfg: cfg}
}

// Start launches the FFmpeg process, creating the output directory first.
// It is a no-op if the encoder is already running.
func (e *Encoder) Start() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.cmd != nil {
		return nil
	}

	dir := filepath.Join(e.cfg.HLSDir, e.cfg.StreamID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create hls directory: %w", err)
	}

	cmd := exec.Command(e.cfg.Bin, e.cfg.Args()...)
	if e.cfg.Stderr != nil {
		cmd.Stderr = e.cfg.Stderr
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create ffmpeg stdin pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ffmpeg hls encoder: %w", err)
	}
	e.cmd = cmd
	e.in = stdin
	go e.watch(cmd)
	return nil
}

// Write feeds one chunk of radio audio into FFmpeg's stdin.
func (e *Encoder) Write(p []byte) (int, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.in == nil {
		return 0, io.ErrClosedPipe
	}
	return e.in.Write(p)
}

// Stop terminates FFmpeg and waits for the process to exit.
func (e *Encoder) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.cmd == nil {
		return
	}
	if e.in != nil {
		_ = e.in.Close()
		e.in = nil
	}
	if e.cmd.Process != nil {
		_ = e.cmd.Process.Kill()
	}
	_ = e.cmd.Wait()
	e.cmd = nil
}

// IsRunning reports whether the encoder process is currently alive.
func (e *Encoder) IsRunning() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.cmd != nil && e.cmd.Process != nil
}

// watch clears the encoder state when FFmpeg exits on its own.
func (e *Encoder) watch(cmd *exec.Cmd) {
	if err := cmd.Wait(); err != nil {
		log.Printf("HLS encoder stopped: %v", err)
	} else {
		log.Printf("HLS encoder stopped")
	}
	e.mu.Lock()
	e.in = nil
	if e.cmd == cmd {
		e.cmd = nil
	}
	e.mu.Unlock()
}
