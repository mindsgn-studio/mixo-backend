package playback

import (
	"fmt"
	"io"
	"os/exec"
	"sync"
)

type FFmpegStreamer struct {
	cmd      *exec.Cmd
	stdout   io.ReadCloser
	filePath string
	mu       sync.Mutex
}

func ffmpegArgs(filePath string) []string {
	return []string{
		"-re",
		"-i", filePath,
		"-map", "0:a:0",
		"-vn",
		"-f", "mp3",
		"-acodec", "libmp3lame",
		"-ar", "44100",
		"-ac", "2",
		"-b:a", "128k",
		"-",
	}
}

func NewFFmpegStreamer(filePath string) (*FFmpegStreamer, error) {
	// -re: output at native frame rate (real-time pacing)
	// -i: input file
	// -map 0:a:0 / -vn: stream audio only and ignore attached cover-art streams
	// -f mp3: output format
	// -acodec libmp3lame: MP3 encoder
	// -ar 44100: sample rate
	// -ac 2: stereo
	// -b:a 128k: bitrate
	// -: output to stdout
	cmd := exec.Command("ffmpeg", ffmpegArgs(filePath)...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	return &FFmpegStreamer{
		cmd:      cmd,
		stdout:   stdout,
		filePath: filePath,
	}, nil
}

func (f *FFmpegStreamer) Read(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.stdout.Read(p)
}

func (f *FFmpegStreamer) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.cmd != nil {
		if err := f.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill ffmpeg: %w", err)
		}
		_ = f.cmd.Wait()
	}
	if f.stdout != nil {
		_ = f.stdout.Close()
	}
	return nil
}

func (f *FFmpegStreamer) IsRunning() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.cmd != nil && f.cmd.Process != nil
}
