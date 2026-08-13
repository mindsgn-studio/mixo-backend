package queue

import (
	"database/sql"
	"fmt"
	"sync"
)

type Song struct {
	ID         int
	Title      string
	Artist     string
	Album      string
	TrackNumber int
	Duration   int
	Location   string
	Normalized bool
}

type QueueItem struct {
	ID       int
	Song     Song
	Position int
}

type Manager struct {
	db *sql.DB
	mu sync.RWMutex
}

func New(db *sql.DB) *Manager {
	return &Manager{db: db}
}

func (m *Manager) Add(songID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var maxPos int
	err := m.db.QueryRow("SELECT COALESCE(MAX(position), 0) FROM queue").Scan(&maxPos)
	if err != nil {
		return fmt.Errorf("failed to get max position: %w", err)
	}

	_, err = m.db.Exec("INSERT INTO queue (song_id, position) VALUES (?, ?)", songID, maxPos+1)
	if err != nil {
		return fmt.Errorf("failed to add to queue: %w", err)
	}

	return nil
}

func (m *Manager) Remove(id int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var position int
	err := m.db.QueryRow("SELECT position FROM queue WHERE id = ?", id).Scan(&position)
	if err != nil {
		return fmt.Errorf("failed to get queue item position: %w", err)
	}

	_, err = m.db.Exec("DELETE FROM queue WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to remove from queue: %w", err)
	}

	_, err = m.db.Exec("UPDATE queue SET position = position - 1 WHERE position > ?", position)
	if err != nil {
		return fmt.Errorf("failed to update positions: %w", err)
	}

	return nil
}

func (m *Manager) Reorder(id int, newPosition int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var oldPosition int
	err := m.db.QueryRow("SELECT position FROM queue WHERE id = ?", id).Scan(&oldPosition)
	if err != nil {
		return fmt.Errorf("failed to get queue item position: %w", err)
	}

	if oldPosition == newPosition {
		return nil
	}

	tx, err := m.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if oldPosition < newPosition {
		_, err = tx.Exec("UPDATE queue SET position = position - 1 WHERE position > ? AND position <= ?", oldPosition, newPosition)
		if err != nil {
			return fmt.Errorf("failed to shift positions down: %w", err)
		}
	} else {
		_, err = tx.Exec("UPDATE queue SET position = position + 1 WHERE position >= ? AND position < ?", newPosition, oldPosition)
		if err != nil {
			return fmt.Errorf("failed to shift positions up: %w", err)
		}
	}

	_, err = tx.Exec("UPDATE queue SET position = ? WHERE id = ?", newPosition, id)
	if err != nil {
		return fmt.Errorf("failed to set new position: %w", err)
	}

	return tx.Commit()
}

func (m *Manager) GetNext() (*Song, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var id, songID int
	err := m.db.QueryRow("SELECT id, song_id FROM queue ORDER BY position ASC LIMIT 1").Scan(&id, &songID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get next queue item: %w", err)
	}

	var song Song
	err = m.db.QueryRow("SELECT id, title, artist, album, track_number, duration, location, normalized FROM songs WHERE id = ?", songID).
		Scan(&song.ID, &song.Title, &song.Artist, &song.Album, &song.TrackNumber, &song.Duration, &song.Location, &song.Normalized)
	if err != nil {
		return nil, fmt.Errorf("failed to get song details: %w", err)
	}

	_, err = m.db.Exec("DELETE FROM queue WHERE id = ?", id)
	if err != nil {
		return nil, fmt.Errorf("failed to remove from queue: %w", err)
	}

	_, err = m.db.Exec("UPDATE queue SET position = position - 1")
	if err != nil {
		return nil, fmt.Errorf("failed to update positions: %w", err)
	}

	return &song, nil
}

func (m *Manager) GetAll() ([]QueueItem, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rows, err := m.db.Query(`
		SELECT q.id, q.position, s.id, s.title, s.artist, s.album, s.track_number, s.duration, s.location, s.normalized
		FROM queue q
		JOIN songs s ON q.song_id = s.id
		ORDER BY q.position ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get queue: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var items []QueueItem
	for rows.Next() {
		var item QueueItem
		err := rows.Scan(&item.ID, &item.Position, &item.Song.ID, &item.Song.Title, &item.Song.Artist, &item.Song.Album, &item.Song.TrackNumber, &item.Song.Duration, &item.Song.Location, &item.Song.Normalized)
		if err != nil {
			return nil, fmt.Errorf("failed to scan queue item: %w", err)
		}
		items = append(items, item)
	}

	return items, nil
}

func (m *Manager) Length() (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var length int
	err := m.db.QueryRow("SELECT COUNT(*) FROM queue").Scan(&length)
	if err != nil {
		return 0, fmt.Errorf("failed to get queue length: %w", err)
	}

	return length, nil
}

func (m *Manager) GetTotalDuration() (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var totalDuration int
	err := m.db.QueryRow("SELECT COALESCE(SUM(s.duration), 0) FROM queue q JOIN songs s ON q.song_id = s.id").Scan(&totalDuration)
	if err != nil {
		return 0, fmt.Errorf("failed to get total duration: %w", err)
	}

	return totalDuration, nil
}

func (m *Manager) GetDurationBeforePosition(position int) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var totalDuration int
	err := m.db.QueryRow("SELECT COALESCE(SUM(s.duration), 0) FROM queue q JOIN songs s ON q.song_id = s.id WHERE q.position < ?", position).Scan(&totalDuration)
	if err != nil {
		return 0, fmt.Errorf("failed to get duration before position: %w", err)
	}

	return totalDuration, nil
}
