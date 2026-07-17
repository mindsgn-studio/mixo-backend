package stream

import (
	"testing"
	"time"
)

func TestBroadcaster_DropsClientWhenBufferIsFull(t *testing.T) {
	chunks := make(chan []byte)
	broadcaster := New(chunks, time.Second)
	client := &Client{
		ID:     "slow",
		Done:   make(chan struct{}),
		Chunks: make(chan []byte, 1),
	}
	client.Chunks <- []byte("already buffered")

	broadcaster.mu.Lock()
	broadcaster.clients[client.ID] = client
	broadcaster.mu.Unlock()

	go broadcaster.broadcastLoop()
	chunks <- []byte("new chunk")

	waitForClientCount(t, broadcaster, 0)
	close(chunks)
}

func TestBroadcaster_DeliversChunkToReadyClient(t *testing.T) {
	chunks := make(chan []byte)
	broadcaster := New(chunks, time.Second)
	client := &Client{
		ID:     "ready",
		Done:   make(chan struct{}),
		Chunks: make(chan []byte, 1),
	}

	broadcaster.mu.Lock()
	broadcaster.clients[client.ID] = client
	broadcaster.mu.Unlock()

	go broadcaster.broadcastLoop()
	chunks <- []byte("new chunk")

	select {
	case got := <-client.Chunks:
		if string(got) != "new chunk" {
			t.Fatalf("expected delivered chunk, got %q", got)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for delivered chunk")
	}
	close(client.Done)
	close(chunks)
}

func TestBroadcaster_RecordStartupChunkKeepsRecentCopies(t *testing.T) {
	broadcaster := New(make(chan []byte), time.Second)

	for i := 0; i < startupBufferChunks+2; i++ {
		chunk := []byte{byte(i)}
		broadcaster.recordStartupChunk(chunk)
		chunk[0] = 255
	}

	if len(broadcaster.startupBuffer) != startupBufferChunks {
		t.Fatalf("expected %d startup chunks, got %d", startupBufferChunks, len(broadcaster.startupBuffer))
	}
	if broadcaster.startupBuffer[0][0] != 2 {
		t.Fatalf("expected oldest retained chunk to be 2, got %d", broadcaster.startupBuffer[0][0])
	}
	if broadcaster.startupBuffer[len(broadcaster.startupBuffer)-1][0] != byte(startupBufferChunks+1) {
		t.Fatalf("expected newest retained chunk to be %d, got %d", startupBufferChunks+1, broadcaster.startupBuffer[len(broadcaster.startupBuffer)-1][0])
	}
}

func waitForClientCount(t *testing.T, broadcaster *Broadcaster, want int) {
	t.Helper()

	deadline := time.After(time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			t.Fatalf("expected client count %d, got %d", want, broadcaster.ClientCount())
		case <-ticker.C:
			if broadcaster.ClientCount() == want {
				return
			}
		}
	}
}
