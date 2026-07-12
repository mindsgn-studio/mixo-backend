package worker

import (
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/mindsgn-studio/mixo-backend/internal/crawler"
	"github.com/mindsgn-studio/mixo-backend/internal/queue"
)

type QueueWorker struct {
	db          *sql.DB
	queue       *queue.Manager
	crawler     *crawler.Crawler
	targetHours int
	interval    int
	mu          sync.Mutex
	running     bool
	stopChan    chan struct{}
}

func New(db *sql.DB, q *queue.Manager, c *crawler.Crawler, targetHours, interval int) *QueueWorker {
	return &QueueWorker{
		db:          db,
		queue:       q,
		crawler:     c,
		targetHours: targetHours,
		interval:    interval,
		stopChan:    make(chan struct{}),
	}
}

func (w *QueueWorker) Start() {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return
	}
	w.running = true
	w.mu.Unlock()

	go w.run()
	log.Printf("Queue worker started (target: %dh, interval: %dm)", w.targetHours, w.interval)
}

func (w *QueueWorker) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.running {
		close(w.stopChan)
		w.running = false
		log.Println("Queue worker stopped")
	}
}

func (w *QueueWorker) run() {
	w.fillQueue()

	ticker := time.NewTicker(time.Duration(w.interval) * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopChan:
			return
		case <-ticker.C:
			w.fillQueue()
		}
	}
}

func (w *QueueWorker) fillQueue() {
	w.mu.Lock()
	defer w.mu.Unlock()

	queueDuration, err := w.getQueueDuration()
	if err != nil {
		log.Printf("Error getting queue duration: %v", err)
		return
	}

	targetSeconds := w.targetHours * 3600
	if queueDuration >= targetSeconds {
		return
	}

	needed := targetSeconds - queueDuration

	songsNeeded := needed / 180
	if songsNeeded < 1 {
		songsNeeded = 1
	}

	randomSongs, err := w.crawler.GetRandomSongs(songsNeeded)
	if err != nil {
		log.Printf("Error getting random songs: %v", err)
		return
	}

	added := 0
	for _, songID := range randomSongs {
		if err := w.queue.Add(songID); err != nil {
			log.Printf("Error adding song %d to queue: %v", songID, err)
			continue
		}
		added++
	}

}

func (w *QueueWorker) getQueueDuration() (int, error) {
	var totalDuration int
	err := w.db.QueryRow(`
		SELECT COALESCE(SUM(s.duration), 0)
		FROM queue q
		JOIN songs s ON q.song_id = s.id
	`).Scan(&totalDuration)
	if err != nil {
		return 0, fmt.Errorf("failed to get queue duration: %w", err)
	}
	return totalDuration, nil
}

func (w *QueueWorker) GetQueueDuration() (string, error) {
	totalSeconds, err := w.getQueueDuration()
	if err != nil {
		return "", err
	}

	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60

	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds), nil
	}
	return fmt.Sprintf("%dm %ds", minutes, seconds), nil
}

func (w *QueueWorker) GetQueueDurationSeconds() (int, error) {
	return w.getQueueDuration()
}
