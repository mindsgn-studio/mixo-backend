package crawler

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
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
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_songs_location ON songs(location);
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
	c := New(db, "/tmp/test-songs")
	if c == nil {
		t.Fatal("expected non-nil crawler")
	}
	if c.db != db {
		t.Error("expected db to be set")
	}
	if c.songDir != "/tmp/test-songs" {
		t.Error("expected songDir to be set")
	}
}

func TestGetTotalDuration_Empty(t *testing.T) {
	db := setupTestDB(t)
	c := New(db, t.TempDir())

	duration, err := c.GetTotalDuration(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if duration != 0 {
		t.Errorf("expected 0, got %d", duration)
	}
}

func TestGetTotalDuration_WithData(t *testing.T) {
	db := setupTestDB(t)
	c := New(db, t.TempDir())

	_, _ = db.Exec("INSERT INTO songs (title, artist, duration, location) VALUES ('Song1', 'Artist1', 180, '/tmp/song1.mp3')")
	_, _ = db.Exec("INSERT INTO songs (title, artist, duration, location) VALUES ('Song2', 'Artist2', 240, '/tmp/song2.mp3')")

	duration, err := c.GetTotalDuration(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if duration != 420 {
		t.Errorf("expected 420, got %d", duration)
	}
}

func TestGetTotalDurationFormatted(t *testing.T) {
	db := setupTestDB(t)
	c := New(db, t.TempDir())

	_, _ = db.Exec("INSERT INTO songs (title, artist, duration, location) VALUES ('Song1', 'Artist1', 3661, '/tmp/song1.mp3')")

	formatted, err := c.GetTotalDurationFormatted()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if formatted != "1h 1m 1s" {
		t.Errorf("expected '1h 1m 1s', got '%s'", formatted)
	}
}

func TestGetTotalDurationFormatted_NoHours(t *testing.T) {
	db := setupTestDB(t)
	c := New(db, t.TempDir())

	_, _ = db.Exec("INSERT INTO songs (title, artist, duration, location) VALUES ('Song1', 'Artist1', 125, '/tmp/song1.mp3')")

	formatted, err := c.GetTotalDurationFormatted()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if formatted != "2m 5s" {
		t.Errorf("expected '2m 5s', got '%s'", formatted)
	}
}

func TestGetRandomSongs_Empty(t *testing.T) {
	db := setupTestDB(t)
	c := New(db, t.TempDir())

	songs, err := c.GetRandomSongs(5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(songs) != 0 {
		t.Errorf("expected 0 songs, got %d", len(songs))
	}
}

func TestGetRandomSongs(t *testing.T) {
	db := setupTestDB(t)
	c := New(db, t.TempDir())

	for i := 0; i < 10; i++ {
		_, _ = db.Exec("INSERT INTO songs (title, artist, duration, location) VALUES (?, 'Artist', 180, ?)",
			"Song"+string(rune('A'+i)), "/tmp/song"+string(rune('A'+i))+".mp3")
	}

	songs, err := c.GetRandomSongs(3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(songs) != 3 {
		t.Errorf("expected 3 songs, got %d", len(songs))
	}
}

func TestGetRandomSongs_RequestMoreThanAvailable(t *testing.T) {
	db := setupTestDB(t)
	c := New(db, t.TempDir())

	_, _ = db.Exec("INSERT INTO songs (title, artist, duration, location) VALUES ('Song1', 'Artist', 180, '/tmp/song1.mp3')")

	songs, err := c.GetRandomSongs(5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(songs) != 1 {
		t.Errorf("expected 1 song, got %d", len(songs))
	}
}

func TestGetSongCount(t *testing.T) {
	db := setupTestDB(t)
	c := New(db, t.TempDir())

	count, err := c.GetSongCount()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}

	_, _ = db.Exec("INSERT INTO songs (title, artist, duration, location) VALUES ('Song1', 'Artist', 180, '/tmp/song1.mp3')")
	_, _ = db.Exec("INSERT INTO songs (title, artist, duration, location) VALUES ('Song2', 'Artist', 180, '/tmp/song2.mp3')")

	count, err = c.GetSongCount()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2, got %d", count)
	}
}

func TestGetAllSongs(t *testing.T) {
	db := setupTestDB(t)
	c := New(db, t.TempDir())

	_, _ = db.Exec("INSERT INTO songs (title, artist, album, duration, location) VALUES ('Song1', 'Artist1', 'Album1', 180, '/tmp/song1.mp3')")
	_, _ = db.Exec("INSERT INTO songs (title, artist, album, duration, location) VALUES ('Song2', 'Artist2', 'Album2', 240, '/tmp/song2.mp3')")

	songs, err := c.GetAllSongs()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(songs) != 2 {
		t.Fatalf("expected 2 songs, got %d", len(songs))
	}

	// Check ordering (by artist, then title)
	if songs[0].Artist != "Artist1" {
		t.Errorf("expected Artist1, got %s", songs[0].Artist)
	}
	if songs[0].Album != "Album1" {
		t.Errorf("expected Album1, got %s", songs[0].Album)
	}
	if songs[1].Artist != "Artist2" {
		t.Errorf("expected Artist2, got %s", songs[1].Artist)
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0:00"},
		{61, "1:01"},
		{3661, "1:01:01"},
		{7200, "2:00:00"},
	}

	for _, tt := range tests {
		got := formatDuration(tt.input)
		if got != tt.expected {
			t.Errorf("formatDuration(%d) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestScanDirectory_Empty(t *testing.T) {
	db := setupTestDB(t)
	dir := t.TempDir()
	c := New(db, dir)

	err := c.ScanDirectories([]string{dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	count, _ := c.GetSongCount()
	if count != 0 {
		t.Errorf("expected 0 songs, got %d", count)
	}
}

func TestScanDirectory_NonExistentDir(t *testing.T) {
	db := setupTestDB(t)
	c := New(db, t.TempDir())

	err := c.ScanDirectories([]string{"/nonexistent/path"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestScanDirectory_SkipsNonMp3(t *testing.T) {
	db := setupTestDB(t)
	dir := t.TempDir()
	c := New(db, dir)

	// Create a non-mp3 file
	txtFile := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(txtFile, []byte("not a song"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	err := c.ScanDirectories([]string{dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	count, _ := c.GetSongCount()
	if count != 0 {
		t.Errorf("expected 0 songs, got %d", count)
	}
}

func TestRemoveMissingFiles(t *testing.T) {
	db := setupTestDB(t)
	c := New(db, t.TempDir())

	// Insert a song with a non-existent location
	_, _ = db.Exec("INSERT INTO songs (title, artist, duration, location) VALUES ('Song1', 'Artist', 180, '/nonexistent/path.mp3')")

	count, _ := c.GetSongCount()
	if count != 1 {
		t.Fatalf("expected 1 song before removal, got %d", count)
	}

	err := c.removeMissingFiles()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	count, _ = c.GetSongCount()
	if count != 0 {
		t.Errorf("expected 0 songs after removal, got %d", count)
	}
}

func TestRemoveMissingFiles_KeepsExisting(t *testing.T) {
	db := setupTestDB(t)
	dir := t.TempDir()
	c := New(db, dir)

	// Create a real file
	mp3File := filepath.Join(dir, "real.mp3")
	if err := os.WriteFile(mp3File, []byte("fake mp3 data"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	absPath, _ := filepath.Abs(mp3File)
	_, _ = db.Exec("INSERT INTO songs (title, artist, duration, location) VALUES ('Song1', 'Artist', 180, ?)", absPath)

	err := c.removeMissingFiles()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	count, _ := c.GetSongCount()
	if count != 1 {
		t.Errorf("expected 1 song to remain, got %d", count)
	}
}
