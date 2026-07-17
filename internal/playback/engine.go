package playback

import (
	"database/sql"
	"io"
	"log"
	"sync"
	"time"

	"github.com/mindsgn-studio/mixo-backend/internal/queue"
)

type audioStreamer interface {
	Read([]byte) (int, error)
	Close() error
}

type Engine struct {
	db          *sql.DB
	queue       *queue.Manager
	chunkChan   chan []byte
	currentSong *queue.Song
	mu          sync.RWMutex
	running     bool
	paused      bool
	stopChan    chan struct{}
	pauseChan   chan struct{}
	resumeChan  chan struct{}
}

func New(db *sql.DB, q *queue.Manager) *Engine {
	return &Engine{
		db:         db,
		queue:      q,
		chunkChan:  make(chan []byte, 100),
		stopChan:   make(chan struct{}),
		pauseChan:  make(chan struct{}),
		resumeChan: make(chan struct{}),
	}
}

func (e *Engine) Start() {
	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return
	}
	e.running = true
	e.mu.Unlock()

	go e.playbackLoop()
}

func (e *Engine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.running {
		close(e.stopChan)
		e.running = false
	}
}

func (e *Engine) playbackLoop() {
	for {
		select {
		case <-e.stopChan:
			return
		case <-e.pauseChan:
			// Wait for resume signal
			select {
			case <-e.stopChan:
				return
			case <-e.resumeChan:
				e.mu.Lock()
				e.paused = false
				e.mu.Unlock()
				continue
			}
		default:
			e.mu.RLock()
			paused := e.paused
			e.mu.RUnlock()
			if paused {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			song, err := e.queue.GetNext()
			if err != nil {
				log.Printf("Error getting next song: %v", err)
				time.Sleep(1 * time.Second)
				continue
			}

			if song == nil {
				e.playFallbackSilence()
				continue
			}

			e.setCurrentSong(song)
			e.playSong(song)
			e.addToHistory(song, song.Duration)
		}
	}
}

func (e *Engine) playSong(song *queue.Song) {
	streamer, paceByBytes, err := newSongStreamer(song)
	if err != nil {
		log.Printf("Error creating audio streamer: %v", err)
		return
	}
	defer func() {
		if err := streamer.Close(); err != nil {
			log.Printf("Warning: failed to close streamer: %v", err)
		}
	}()

	buffer := make([]byte, 4096)
	startTime := time.Now()
	var bytesSent int64

	for {
		select {
		case <-e.stopChan:
			return
		default:
			n, err := streamer.Read(buffer)
			if err != nil {
				if err == io.EOF {
					return
				}
				log.Printf("Error reading from stream: %v", err)
				return
			}

			chunk := make([]byte, n)
			copy(chunk, buffer[:n])
			if !e.publishChunk(chunk) {
				return
			}

			if paceByBytes {
				bytesSent += int64(n)
				if !e.waitForBitrate(startTime, bytesSent) {
					return
				}
			}
		}
	}
}

func newSongStreamer(song *queue.Song) (audioStreamer, bool, error) {
	if song.Normalized {
		streamer, err := NewFileStreamer(song.Location)
		if err == nil {
			return streamer, true, nil
		}
		log.Printf("Error opening normalized audio, falling back to FFmpeg: %v", err)
	}

	streamer, err := NewFFmpegStreamer(song.Location)
	return streamer, false, err
}

func (e *Engine) playFallbackSilence() {
	streamer, err := NewFFmpegSilenceStreamer()
	if err != nil {
		log.Printf("Error creating silence streamer: %v", err)
		time.Sleep(1 * time.Second)
		return
	}
	defer func() {
		if err := streamer.Close(); err != nil {
			log.Printf("Warning: failed to close silence streamer: %v", err)
		}
	}()

	buffer := make([]byte, 4096)
	checkQueue := time.NewTicker(1 * time.Second)
	defer checkQueue.Stop()

	for {
		select {
		case <-e.stopChan:
			return
		case <-checkQueue.C:
			length, err := e.queue.Length()
			if err == nil && length > 0 {
				return
			}
			if err != nil {
				log.Printf("Error checking queue during silence fallback: %v", err)
			}
		default:
			n, err := streamer.Read(buffer)
			if err != nil {
				if err != io.EOF {
					log.Printf("Error reading silence stream: %v", err)
				}
				return
			}

			chunk := make([]byte, n)
			copy(chunk, buffer[:n])
			if !e.publishChunk(chunk) {
				return
			}
		}
	}
}

func (e *Engine) publishChunk(chunk []byte) bool {
	select {
	case e.chunkChan <- chunk:
		return true
	case <-e.stopChan:
		return false
	}
}

func (e *Engine) waitForBitrate(startTime time.Time, bytesSent int64) bool {
	delay := bitrateDelay(startTime, bytesSent, time.Now())
	if delay <= 0 {
		return true
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-timer.C:
		return true
	case <-e.stopChan:
		return false
	}
}

func bitrateDelay(startTime time.Time, bytesSent int64, now time.Time) time.Duration {
	expectedElapsed := time.Duration((bytesSent * 8 * int64(time.Second)) / stationBitrateBitsPerSec)
	actualElapsed := now.Sub(startTime)
	if expectedElapsed <= actualElapsed {
		return 0
	}
	return expectedElapsed - actualElapsed
}

func (e *Engine) setCurrentSong(song *queue.Song) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.currentSong = song

	// Update state in database
	_, err := e.db.Exec("INSERT OR REPLACE INTO state (key, value) VALUES ('current_song', ?)", song.ID)
	if err != nil {
		log.Printf("Error updating current song state: %v", err)
	}
}

func (e *Engine) GetCurrentSong() *queue.Song {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.currentSong
}

func (e *Engine) GetChunkChan() <-chan []byte {
	return e.chunkChan
}

func (e *Engine) addToHistory(song *queue.Song, durationPlayed int) {
	_, err := e.db.Exec("INSERT INTO history (song_id, duration_played) VALUES (?, ?)", song.ID, durationPlayed)
	if err != nil {
		log.Printf("Error adding to history: %v", err)
	}
}

func (e *Engine) Pause() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.running && !e.paused {
		e.paused = true
		select {
		case e.pauseChan <- struct{}{}:
		default:
		}
	}
}

func (e *Engine) Resume() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.running && e.paused {
		e.paused = false
		select {
		case e.resumeChan <- struct{}{}:
		default:
		}
	}
}

func (e *Engine) IsPaused() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.paused
}
