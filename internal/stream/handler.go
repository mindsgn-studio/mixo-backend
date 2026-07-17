package stream

import (
	"net/http"
	"strconv"
	"sync/atomic"
	"time"
)

var clientIDCounter uint64

type Handler struct {
	broadcaster *Broadcaster
}

func NewHandler(broadcaster *Broadcaster) *Handler {
	return &Handler{broadcaster: broadcaster}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Set headers for streaming
	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("icy-name", "Hackday Radio")
	w.Header().Set("icy-pub", "1")

	// Create client
	clientID := generateClientID()
	client := &Client{
		ID:     clientID,
		Writer: w,
		Done:   make(chan struct{}),
	}

	// Register client
	h.broadcaster.Register(client)
	defer h.broadcaster.Unregister(clientID)

	select {
	case <-client.Done:
	case <-r.Context().Done():
	}
}

func generateClientID() string {
	id := atomic.AddUint64(&clientIDCounter, 1)
	return time.Now().Format("20060102150405.000000") + "-" + strconv.FormatUint(id, 10)
}
