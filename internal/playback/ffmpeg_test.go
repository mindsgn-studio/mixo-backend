package playback

import (
	"slices"
	"testing"
)

func TestFFmpegArgs_UsesAudioOnlyStream(t *testing.T) {
	t.Helper()

	args := ffmpegArgs("/tmp/song.mp3")

	if !slices.Contains(args, "-vn") {
		t.Fatalf("expected ffmpeg args to disable video streams, args=%v", args)
	}

	mapIndex := slices.Index(args, "-map")
	if mapIndex == -1 || mapIndex+1 >= len(args) {
		t.Fatalf("expected ffmpeg args to include stream mapping, args=%v", args)
	}

	if args[mapIndex+1] != "0:a:0" {
		t.Fatalf("expected ffmpeg to map the first audio stream, got %q", args[mapIndex+1])
	}
}
