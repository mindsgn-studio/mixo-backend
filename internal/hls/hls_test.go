package hls

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfig_Args(t *testing.T) {
	cfg := Config{
		Bin:          "ffmpeg",
		HLSDir:       "/var/www/html/hls",
		StreamID:     "radio",
		SegmentTime:  2,
		PlaylistSize: 6,
	}

	args := cfg.Args()
	joined := strings.Join(args, " ")

	if !strings.Contains(joined, "-i pipe:0") {
		t.Errorf("expected stdin input, got %q", joined)
	}
	if !strings.Contains(joined, "-f hls") {
		t.Errorf("expected hls muxer, got %q", joined)
	}
	if !strings.Contains(joined, "-hls_time 2") {
		t.Errorf("expected segment time 2s, got %q", joined)
	}
	if !strings.Contains(joined, "-hls_list_size 6") {
		t.Errorf("expected playlist size 6, got %q", joined)
	}
	if !strings.Contains(joined, filepath.Join("/var/www/html/hls", "radio", "index.m3u8")) {
		t.Errorf("expected output path in args, got %q", joined)
	}
}

func TestNew_Defaults(t *testing.T) {
	e := New(Config{})

	if e.cfg.Bin != "ffmpeg" {
		t.Errorf("expected default bin ffmpeg, got %q", e.cfg.Bin)
	}
	if e.cfg.HLSDir != "/var/www/html/hls" {
		t.Errorf("expected default hls dir, got %q", e.cfg.HLSDir)
	}
	if e.cfg.StreamID != "radio" {
		t.Errorf("expected default stream id 'radio', got %q", e.cfg.StreamID)
	}
	if e.cfg.SegmentTime != 2 {
		t.Errorf("expected default segment time 2, got %d", e.cfg.SegmentTime)
	}
	if e.cfg.PlaylistSize != 6 {
		t.Errorf("expected default playlist size 6, got %d", e.cfg.PlaylistSize)
	}
}

func TestEncoder_StartStop(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "fakeffmpeg")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nsleep 30\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	e := New(Config{
		Bin:         bin,
		HLSDir:      dir,
		StreamID:    "radio",
		SegmentTime: 2,
	})
	if err := e.Start(); err != nil {
		t.Fatalf("expected start to succeed: %v", err)
	}
	t.Cleanup(e.Stop)

	if !e.IsRunning() {
		t.Fatal("expected encoder to be running after Start")
	}

	if _, err := e.Write([]byte("fake audio chunk")); err != nil {
		t.Fatalf("expected write to succeed: %v", err)
	}

	e.Stop()
	if e.IsRunning() {
		t.Fatal("expected encoder to be stopped after Stop")
	}
}

func TestEncoder_StartTwiceIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "fakeffmpeg")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nsleep 30\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	e := New(Config{Bin: bin, HLSDir: dir, StreamID: "radio"})
	if err := e.Start(); err != nil {
		t.Fatalf("expected first start to succeed: %v", err)
	}
	t.Cleanup(e.Stop)

	if err := e.Start(); err != nil {
		t.Fatalf("expected second start to be a no-op: %v", err)
	}
}
