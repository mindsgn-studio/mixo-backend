package playback

import (
	"testing"
	"time"
)

func TestBitrateDelay_WhenAheadOfSchedule_ReturnsDelay(t *testing.T) {
	start := time.Unix(0, 0)
	bytesSent := int64(4096)

	delay := bitrateDelay(start, bytesSent, start)

	expected := time.Duration((bytesSent * 8 * int64(time.Second)) / stationBitrateBitsPerSec)
	if delay != expected {
		t.Fatalf("expected delay %s, got %s", expected, delay)
	}
}

func TestBitrateDelay_WhenOnSchedule_ReturnsZero(t *testing.T) {
	start := time.Unix(0, 0)
	bytesSent := int64(4096)
	elapsed := time.Duration((bytesSent * 8 * int64(time.Second)) / stationBitrateBitsPerSec)

	delay := bitrateDelay(start, bytesSent, start.Add(elapsed))

	if delay != 0 {
		t.Fatalf("expected no delay, got %s", delay)
	}
}

func TestBitrateDelay_WhenBehindSchedule_ReturnsZero(t *testing.T) {
	start := time.Unix(0, 0)
	bytesSent := int64(4096)
	elapsed := time.Duration((bytesSent * 8 * int64(time.Second)) / stationBitrateBitsPerSec)

	delay := bitrateDelay(start, bytesSent, start.Add(elapsed+time.Second))

	if delay != 0 {
		t.Fatalf("expected no delay, got %s", delay)
	}
}
