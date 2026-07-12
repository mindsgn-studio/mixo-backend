package stream

import (
	"net/http"
	"time"
)

type Handler struct {
	broadcaster *Broadcaster
}

func NewHandler(broadcaster *Broadcaster) *Handler {
	return &Handler{broadcaster: broadcaster}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Set headers for streaming
	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Cache-Control", "no-cache, no-store")
	w.Header().Set("X-Accel-Buffering", "no")

	// Create client
	clientID := generateClientID()
	client := &Client{
		ID:     clientID,
		Writer: w,
		Done:   make(chan struct{}),
	}

	// Register client
	h.broadcaster.Register(client)
	defer func() {
		h.broadcaster.Unregister(clientID)
	}()

	// Flush headers immediately
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	// Keep connection open until client disconnects
	<-client.Done
}

func generateClientID() string {
	return time.Now().Format("20060102150405.000000")
}
