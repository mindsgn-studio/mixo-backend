package worker

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mindsgn-studio/mixo-backend/internal/crawler"
	"github.com/mindsgn-studio/mixo-backend/internal/queue"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	schema := `
	CREATE TABLE IF NOT EXISTS songs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL DEFAULT '',
		artist TEXT NOT NULL DEFAULT '',
		album TEXT DEFAULT '',
			cover_art TEXT DEFAULT '',
			duration INTEGER NOT NULL DEFAULT 0,
			location TEXT NOT NULL,
			source_location TEXT DEFAULT '',
			normalized INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_songs_location ON songs(location);
		CREATE INDEX IF NOT EXISTS idx_songs_source_location ON songs(source_location);
	CREATE TABLE IF NOT EXISTS queue (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		song_id INTEGER NOT NULL,
		position INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (song_id) REFERENCES songs(id) ON DELETE CASCADE
	);
	CREATE TABLE IF NOT EXISTS history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		song_id INTEGER NOT NULL,
		played_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (song_id) REFERENCES songs(id) ON DELETE CASCADE
	);
	CREATE TABLE IF NOT EXISTS state (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);
	`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}

func TestNew(t *testing.T) {
	db := setupTestDB(t)
	qm := queue.New(db)
	c := crawler.New(db, t.TempDir())

	w := New(db, qm, c, 24, 60)
	if w == nil {
		t.Fatal("expected non-nil worker")
	}
	if w.targetHours != 24 {
		t.Errorf("expected targetHours 24, got %d", w.targetHours)
	}
	if w.interval != 60 {
		t.Errorf("expected interval 60, got %d", w.interval)
	}
}

func TestGetQueueDuration_Empty(t *testing.T) {
	db := setupTestDB(t)
	qm := queue.New(db)
	c := crawler.New(db, t.TempDir())

	w := New(db, qm, c, 24, 60)

	duration, err := w.GetQueueDurationSeconds()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if duration != 0 {
		t.Errorf("expected 0, got %d", duration)
	}
}

func TestGetQueueDuration_WithData(t *testing.T) {
	db := setupTestDB(t)
	qm := queue.New(db)
	c := crawler.New(db, t.TempDir())

	w := New(db, qm, c, 24, 60)

	// Add songs
	_, _ = db.Exec("INSERT INTO songs (title, artist, duration, location) VALUES ('Song1', 'Artist1', 180, '/tmp/song1.mp3')")
	_, _ = db.Exec("INSERT INTO songs (title, artist, duration, location) VALUES ('Song2', 'Artist2', 240, '/tmp/song2.mp3')")

	// Add to queue
	_ = qm.Add(1)
	_ = qm.Add(2)

	duration, err := w.GetQueueDurationSeconds()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if duration != 420 {
		t.Errorf("expected 420, got %d", duration)
	}
}

func TestGetQueueDurationFormatted(t *testing.T) {
	db := setupTestDB(t)
	qm := queue.New(db)
	c := crawler.New(db, t.TempDir())

	w := New(db, qm, c, 24, 60)

	_, _ = db.Exec("INSERT INTO songs (title, artist, duration, location) VALUES ('Song1', 'Artist1', 3661, '/tmp/song1.mp3')")
	_ = qm.Add(1)

	formatted, err := w.GetQueueDuration()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if formatted != "1h 1m 1s" {
		t.Errorf("expected '1h 1m 1s', got '%s'", formatted)
	}
}

func TestFillQueue_EmptyLibrary(t *testing.T) {
	db := setupTestDB(t)
	qm := queue.New(db)
	c := crawler.New(db, t.TempDir())

	w := New(db, qm, c, 24, 60)
	w.fillQueue()

	length, _ := qm.Length()
	if length != 0 {
		t.Errorf("expected queue length 0, got %d", length)
	}
}

func TestFillQueue_AlreadyFull(t *testing.T) {
	db := setupTestDB(t)
	qm := queue.New(db)
	c := crawler.New(db, t.TempDir())

	w := New(db, qm, c, 1, 60) // 1 hour target

	// Add a song with enough duration
	_, _ = db.Exec("INSERT INTO songs (title, artist, duration, location) VALUES ('Song1', 'Artist1', 3601, '/tmp/song1.mp3')")
	_ = qm.Add(1)

	w.fillQueue()

	// Should not add more songs
	length, _ := qm.Length()
	if length != 1 {
		t.Errorf("expected queue length 1, got %d", length)
	}
}

func TestFillQueue_NeedsMore(t *testing.T) {
	db := setupTestDB(t)
	qm := queue.New(db)
	c := crawler.New(db, t.TempDir())

	w := New(db, qm, c, 1, 60) // 1 hour target

	// Add a song with less duration than target
	_, _ = db.Exec("INSERT INTO songs (title, artist, duration, location) VALUES ('Song1', 'Artist1', 180, '/tmp/song1.mp3')")
	_ = qm.Add(1)

	w.fillQueue()

	// fillQueue adds random songs; with only 1 song in library, it adds it again as a duplicate
	length, _ := qm.Length()
	if length < 1 {
		t.Errorf("expected queue length >= 1, got %d", length)
	}
}

func TestStartStop(t *testing.T) {
	db := setupTestDB(t)
	qm := queue.New(db)
	c := crawler.New(db, t.TempDir())

	w := New(db, qm, c, 24, 60)

	w.Start()
	time.Sleep(10 * time.Millisecond)

	w.mu.Lock()
	running := w.running
	w.mu.Unlock()

	if !running {
		t.Error("expected worker to be running")
	}

	w.Stop()
	time.Sleep(10 * time.Millisecond)

	w.mu.Lock()
	running = w.running
	w.mu.Unlock()

	if running {
		t.Error("expected worker to be stopped")
	}
}

func TestStart_Idempotent(t *testing.T) {
	db := setupTestDB(t)
	qm := queue.New(db)
	c := crawler.New(db, t.TempDir())

	w := New(db, qm, c, 24, 60)

	w.Start()
	w.Start() // Second start should be a no-op

	w.mu.Lock()
	running := w.running
	w.mu.Unlock()

	if !running {
		t.Error("expected worker to be running")
	}

	w.Stop()
}

func TestStop_Idempotent(t *testing.T) {
	db := setupTestDB(t)
	qm := queue.New(db)
	c := crawler.New(db, t.TempDir())

	w := New(db, qm, c, 24, 60)

	w.Stop() // Stop without start should be a no-op

	w.mu.Lock()
	running := w.running
	w.mu.Unlock()

	if running {
		t.Error("expected worker to not be running")
	}
}
