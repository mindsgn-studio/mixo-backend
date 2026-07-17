package playback

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

const (
	StationBitrateKbps       = 128
	stationBitrateBitsPerSec = StationBitrateKbps * 1000
)

type FFmpegStreamer struct {
	cmd    *exec.Cmd
	stdout io.ReadCloser
	mu     sync.Mutex
}

func NewFFmpegStreamer(filePath string) (*FFmpegStreamer, error) {
	cmd := exec.Command(
		"ffmpeg",
		"-nostdin",
		"-hide_banner",
		"-loglevel", "error",
		"-re",
		"-i", filePath,
		"-vn",
		"-f", "mp3",
		"-acodec", "libmp3lame",
		"-ar", "44100",
		"-ac", "2",
		"-b:a", fmt.Sprintf("%dk", StationBitrateKbps),
		"-",
	)

	return startFFmpegStreamer(cmd)
}

func NewFFmpegSilenceStreamer() (*FFmpegStreamer, error) {
	cmd := exec.Command(
		"ffmpeg",
		"-nostdin",
		"-hide_banner",
		"-loglevel", "error",
		"-re",
		"-f", "lavfi",
		"-i", "anullsrc=r=44100:cl=stereo",
		"-vn",
		"-f", "mp3",
		"-acodec", "libmp3lame",
		"-ar", "44100",
		"-ac", "2",
		"-b:a", fmt.Sprintf("%dk", StationBitrateKbps),
		"-",
	)

	return startFFmpegStreamer(cmd)
}

func startFFmpegStreamer(cmd *exec.Cmd) (*FFmpegStreamer, error) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	return &FFmpegStreamer{
		cmd:    cmd,
		stdout: stdout,
	}, nil
}

type FileStreamer struct {
	file *os.File
	mu   sync.Mutex
}

func NewFileStreamer(filePath string) (*FileStreamer, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open audio file: %w", err)
	}

	return &FileStreamer{file: file}, nil
}

func (f *FileStreamer) Read(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.file.Read(p)
}

func (f *FileStreamer) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.file == nil {
		return nil
	}
	if err := f.file.Close(); err != nil {
		return fmt.Errorf("failed to close audio file: %w", err)
	}
	f.file = nil
	return nil
}

func (f *FFmpegStreamer) Read(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.stdout.Read(p)
}

func (f *FFmpegStreamer) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.stdout != nil {
		_ = f.stdout.Close()
		f.stdout = nil
	}
	if f.cmd != nil {
		if f.cmd.Process != nil && f.cmd.ProcessState == nil {
			_ = f.cmd.Process.Kill()
		}
		_ = f.cmd.Wait()
		f.cmd = nil
	}
	return nil
}

func (f *FFmpegStreamer) IsRunning() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.cmd != nil && f.cmd.Process != nil
}

func StationCacheDir(songDir string) string {
	return filepath.Join(songDir, ".normalized")
}

func NormalizeToStationMP3(sourcePath, targetDir string) (string, error) {
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create normalized audio directory: %w", err)
	}

	targetPath := filepath.Join(targetDir, normalizedFilename(sourcePath))
	if info, err := os.Stat(targetPath); err == nil && info.Size() > 0 {
		return targetPath, nil
	}

	tmpPath := fmt.Sprintf("%s.tmp-%d", targetPath, os.Getpid())
	cmd := exec.Command(
		"ffmpeg",
		"-nostdin",
		"-hide_banner",
		"-loglevel", "error",
		"-y",
		"-i", sourcePath,
		"-vn",
		"-map_metadata", "-1",
		"-f", "mp3",
		"-acodec", "libmp3lame",
		"-ar", "44100",
		"-ac", "2",
		"-b:a", fmt.Sprintf("%dk", StationBitrateKbps),
		"-write_id3v1", "0",
		"-id3v2_version", "0",
		"-write_xing", "0",
		tmpPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		_ = os.Remove(tmpPath)
		message := strings.TrimSpace(string(output))
		if message == "" {
			return "", fmt.Errorf("ffmpeg normalization failed: %w", err)
		}
		return "", fmt.Errorf("ffmpeg normalization failed: %w: %s", err, message)
	}

	if err := os.Rename(tmpPath, targetPath); err != nil {
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("failed to move normalized audio: %w", err)
	}

	return targetPath, nil
}

func normalizedFilename(sourcePath string) string {
	sum := sha256.Sum256([]byte(sourcePath))
	base := filepath.Base(sourcePath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	name = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= 'A' && r <= 'Z':
			return r
		case r >= '0' && r <= '9':
			return r
		case r == '-' || r == '_':
			return r
		default:
			return '-'
		}
	}, name)
	name = strings.Trim(name, "-")
	if name == "" {
		name = "audio"
	}

	return fmt.Sprintf("%s-%s.mp3", name, hex.EncodeToString(sum[:])[:12])
}
