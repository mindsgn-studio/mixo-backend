package stream

import (
	"log"
	"net/http"
	"sync"
	"time"
)

const (
	clientBufferSize    = 64
	startupBufferChunks = 12
)

type Client struct {
	ID     string
	Writer http.ResponseWriter
	Done   chan struct{}
	Chunks chan []byte
}

type Broadcaster struct {
	clients       map[string]*Client
	sinks         map[chan []byte]struct{}
	mu            sync.RWMutex
	chunkChan     <-chan []byte
	streamTimeout time.Duration
	startupBuffer [][]byte
}

func New(chunkChan <-chan []byte, timeout time.Duration) *Broadcaster {
	return &Broadcaster{
		clients:       make(map[string]*Client),
		sinks:         make(map[chan []byte]struct{}),
		chunkChan:     chunkChan,
		streamTimeout: timeout,
		startupBuffer: make([][]byte, 0, startupBufferChunks),
	}
}

// AddSink registers an additional consumer of every broadcast chunk (for
// example the HLS encoder). Sends are non-blocking: a slow sink drops chunks
// rather than stalling the live broadcast. The returned function removes the
// sink again.
func (b *Broadcaster) AddSink(ch chan []byte) func() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.sinks[ch] = struct{}{}
	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		delete(b.sinks, ch)
	}
}

func (b *Broadcaster) Register(client *Client) {
	if client.Done == nil {
		client.Done = make(chan struct{})
	}
	if client.Chunks == nil {
		client.Chunks = make(chan []byte, clientBufferSize)
	}

	b.mu.Lock()
	for _, chunk := range b.startupBuffer {
		select {
		case client.Chunks <- chunk:
		default:
			log.Printf("Startup stream buffer overflow for client %s", client.ID)
		}
	}
	b.clients[client.ID] = client
	total := len(b.clients)
	b.mu.Unlock()

	go b.writeClient(client)
	log.Printf("Client registered: %s (total: %d)", client.ID, total)
}

func (b *Broadcaster) Unregister(clientID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if client, ok := b.clients[clientID]; ok {
		close(client.Done)
		delete(b.clients, clientID)
		log.Printf("Client unregistered: %s (total: %d)", clientID, len(b.clients))
	}
}

func (b *Broadcaster) Start() {
	go b.broadcastLoop()
}

func (b *Broadcaster) broadcastLoop() {
	for chunk := range b.chunkChan {
		b.mu.Lock()
		b.recordStartupChunk(chunk)

		slowClients := make([]string, 0)
		for _, client := range b.clients {
			select {
			case <-client.Done:
				continue
			case client.Chunks <- chunk:
			default:
				slowClients = append(slowClients, client.ID)
			}
		}

		for sink := range b.sinks {
			select {
			case sink <- chunk:
			default:
				log.Printf("Broadcast sink fell behind - dropping chunk")
			}
		}
		b.mu.Unlock()

		for _, clientID := range slowClients {
			log.Printf("Client %s fell behind stream buffer", clientID)
			b.Unregister(clientID)
		}
	}
}

func (b *Broadcaster) recordStartupChunk(chunk []byte) {
	copied := make([]byte, len(chunk))
	copy(copied, chunk)
	b.startupBuffer = append(b.startupBuffer, copied)
	if len(b.startupBuffer) > startupBufferChunks {
		copy(b.startupBuffer, b.startupBuffer[len(b.startupBuffer)-startupBufferChunks:])
		b.startupBuffer = b.startupBuffer[:startupBufferChunks]
	}
}

func (b *Broadcaster) writeClient(client *Client) {
	for {
		select {
		case <-client.Done:
			return
		case chunk := <-client.Chunks:
			if !b.writeToClient(client, chunk) {
				b.Unregister(client.ID)
				return
			}
		}
	}
}

func (b *Broadcaster) writeToClient(client *Client, chunk []byte) bool {
	controller := http.NewResponseController(client.Writer)
	if b.streamTimeout > 0 {
		_ = controller.SetWriteDeadline(time.Now().Add(b.streamTimeout))
		defer func() {
			_ = controller.SetWriteDeadline(time.Time{})
		}()
	}

	if _, err := client.Writer.Write(chunk); err != nil {
		log.Printf("Error writing to client %s: %v", client.ID, err)
		return false
	}
	if err := controller.Flush(); err != nil {
		log.Printf("Error flushing client %s: %v", client.ID, err)
		return false
	}
	return true
}

func (b *Broadcaster) ClientCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.clients)
}
