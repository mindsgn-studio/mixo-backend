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
		genre TEXT DEFAULT '',
		track_number INTEGER DEFAULT 0,
		track_total INTEGER DEFAULT 0,
		cover_art TEXT DEFAULT '',
		duration INTEGER NOT NULL DEFAULT 0,
		location TEXT NOT NULL,
		source_location TEXT DEFAULT '',
		normalized INTEGER NOT NULL DEFAULT 0,
		status TEXT DEFAULT 'deleted',
		is_favourite INTEGER NOT NULL DEFAULT 0,
		added_to_library_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_songs_location ON songs(location);
	CREATE INDEX IF NOT EXISTS idx_songs_source_location ON songs(source_location);
	CREATE INDEX IF NOT EXISTS idx_songs_status ON songs(status);
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
		duration_played INTEGER NOT NULL DEFAULT 0,
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
		_, _ = db.Exec("INSERT INTO songs (title, artist, duration, location, status) VALUES (?, 'Artist', 180, ?, 'library')",
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

	_, _ = db.Exec("INSERT INTO songs (title, artist, duration, location, status) VALUES ('Song1', 'Artist', 180, '/tmp/song1.mp3', 'library')")

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

	_, _ = db.Exec("INSERT INTO songs (title, artist, album, genre, track_number, duration, location, status) VALUES ('Song1', 'Artist1', 'Album1', 'Rock', 1, 180, '/tmp/song1.mp3', 'library')")
	_, _ = db.Exec("INSERT INTO songs (title, artist, album, genre, track_number, duration, location, status) VALUES ('Song2', 'Artist2', 'Album2', 'Pop', 2, 240, '/tmp/song2.mp3', 'library')")

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
	if songs[0].Genre != "Rock" {
		t.Errorf("expected Rock, got %s", songs[0].Genre)
	}
	if songs[0].TrackNumber != 1 {
		t.Errorf("expected track 1, got %d", songs[0].TrackNumber)
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

	// Insert a song with a non-existent location, status='library'
	_, _ = db.Exec("INSERT INTO songs (title, artist, duration, location, status) VALUES ('Song1', 'Artist', 180, '/nonexistent/path.mp3', 'library')")

	var status string
	_ = db.QueryRow("SELECT status FROM songs WHERE title = 'Song1'").Scan(&status)
	if status != "library" {
		t.Fatalf("expected status 'library' before removal, got %s", status)
	}

	err := c.removeMissingFiles()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Song should still exist but with status='deleted'
	_ = db.QueryRow("SELECT status FROM songs WHERE title = 'Song1'").Scan(&status)
	if status != "deleted" {
		t.Errorf("expected status 'deleted' after removal, got %s", status)
	}

	count, _ := c.GetSongCount()
	if count != 1 {
		t.Errorf("expected 1 song to remain (soft deleted), got %d", count)
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
	_, _ = db.Exec("INSERT INTO songs (title, artist, duration, location, status) VALUES ('Song1', 'Artist', 180, ?, 'library')", absPath)

	err := c.removeMissingFiles()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var status string
	_ = db.QueryRow("SELECT status FROM songs WHERE title = 'Song1'").Scan(&status)
	if status != "library" {
		t.Errorf("expected status to remain 'library', got %s", status)
	}

	count, _ := c.GetSongCount()
	if count != 1 {
		t.Errorf("expected 1 song to remain, got %d", count)
	}
}

func TestGetAllAlbums(t *testing.T) {
	db := setupTestDB(t)
	c := New(db, t.TempDir())

	_, _ = db.Exec("INSERT INTO songs (title, artist, album, duration, location, status) VALUES ('S1', 'A1', 'Album A', 180, '/tmp/s1.mp3', 'library')")
	_, _ = db.Exec("INSERT INTO songs (title, artist, album, duration, location, status) VALUES ('S2', 'A2', 'Album B', 200, '/tmp/s2.mp3', 'library')")
	_, _ = db.Exec("INSERT INTO songs (title, artist, album, duration, location, status) VALUES ('S3', 'A3', 'Album A', 150, '/tmp/s3.mp3', 'library')")
	_, _ = db.Exec("INSERT INTO songs (title, artist, duration, location, status) VALUES ('S4', 'A4', 160, '/tmp/s4.mp3', 'library')")

	albums, err := c.GetAllAlbums()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(albums) != 2 {
		t.Fatalf("expected 2 albums, got %d", len(albums))
	}
	if albums[0] != "Album A" {
		t.Errorf("expected 'Album A', got '%s'", albums[0])
	}
	if albums[1] != "Album B" {
		t.Errorf("expected 'Album B', got '%s'", albums[1])
	}
}

func TestGetAlbumSongs(t *testing.T) {
	db := setupTestDB(t)
	c := New(db, t.TempDir())

	_, _ = db.Exec("INSERT INTO songs (title, artist, album, genre, track_number, duration, location, status) VALUES ('S1', 'A1', 'Album1', 'Rock', 1, 180, '/tmp/s1.mp3', 'library')")
	_, _ = db.Exec("INSERT INTO songs (title, artist, album, genre, track_number, duration, location, status) VALUES ('S2', 'A1', 'Album1', 'Rock', 2, 200, '/tmp/s2.mp3', 'library')")
	_, _ = db.Exec("INSERT INTO songs (title, artist, album, genre, track_number, duration, location, status) VALUES ('S3', 'A2', 'Album2', 'Pop', 1, 150, '/tmp/s3.mp3', 'library')")

	songs, err := c.GetAlbumSongs("Album1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(songs) != 2 {
		t.Fatalf("expected 2 songs, got %d", len(songs))
	}
	if songs[0].Title != "S1" {
		t.Errorf("expected 'S1', got '%s'", songs[0].Title)
	}
	if songs[0].TrackNumber != 1 {
		t.Errorf("expected track 1, got %d", songs[0].TrackNumber)
	}
	if songs[1].TrackNumber != 2 {
		t.Errorf("expected track 2, got %d", songs[1].TrackNumber)
	}
}

func TestGetAlbumSongs_Empty(t *testing.T) {
	db := setupTestDB(t)
	c := New(db, t.TempDir())

	songs, err := c.GetAlbumSongs("Nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(songs) != 0 {
		t.Errorf("expected 0 songs, got %d", len(songs))
	}
}

func TestGetAllArtists(t *testing.T) {
	db := setupTestDB(t)
	c := New(db, t.TempDir())

	_, _ = db.Exec("INSERT INTO songs (title, artist, duration, location, status) VALUES ('S1', 'Zebra', 180, '/tmp/s1.mp3', 'library')")
	_, _ = db.Exec("INSERT INTO songs (title, artist, duration, location, status) VALUES ('S2', 'Apple', 200, '/tmp/s2.mp3', 'library')")
	_, _ = db.Exec("INSERT INTO songs (title, artist, duration, location, status) VALUES ('S3', 'Zebra', 150, '/tmp/s3.mp3', 'library')")

	artists, err := c.GetAllArtists()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(artists) != 2 {
		t.Fatalf("expected 2 artists, got %d", len(artists))
	}
	if artists[0] != "Apple" {
		t.Errorf("expected 'Apple', got '%s'", artists[0])
	}
	if artists[1] != "Zebra" {
		t.Errorf("expected 'Zebra', got '%s'", artists[1])
	}
}

func TestGetArtistSongs(t *testing.T) {
	db := setupTestDB(t)
	c := New(db, t.TempDir())

	_, _ = db.Exec("INSERT INTO songs (title, artist, album, genre, track_number, duration, location, status) VALUES ('S1', 'Artist1', 'Album1', 'Rock', 1, 180, '/tmp/s1.mp3', 'library')")
	_, _ = db.Exec("INSERT INTO songs (title, artist, album, genre, track_number, duration, location, status) VALUES ('S2', 'Artist1', 'Album2', 'Pop', 1, 200, '/tmp/s2.mp3', 'library')")
	_, _ = db.Exec("INSERT INTO songs (title, artist, album, genre, track_number, duration, location, status) VALUES ('S3', 'Artist2', 'Album3', 'Jazz', 1, 150, '/tmp/s3.mp3', 'library')")

	songs, err := c.GetArtistSongs("Artist1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(songs) != 2 {
		t.Fatalf("expected 2 songs, got %d", len(songs))
	}
	if songs[0].Album != "Album1" {
		t.Errorf("expected 'Album1', got '%s'", songs[0].Album)
	}
	if songs[1].Album != "Album2" {
		t.Errorf("expected 'Album2', got '%s'", songs[1].Album)
	}
}

func TestGetArtistAlbums(t *testing.T) {
	db := setupTestDB(t)
	c := New(db, t.TempDir())

	_, _ = db.Exec("INSERT INTO songs (title, artist, album, duration, location, status) VALUES ('S1', 'Artist1', 'Album A', 180, '/tmp/s1.mp3', 'library')")
	_, _ = db.Exec("INSERT INTO songs (title, artist, album, duration, location, status) VALUES ('S2', 'Artist1', 'Album B', 200, '/tmp/s2.mp3', 'library')")
	_, _ = db.Exec("INSERT INTO songs (title, artist, album, duration, location, status) VALUES ('S3', 'Artist2', 'Album C', 150, '/tmp/s3.mp3', 'library')")

	albums, err := c.GetArtistAlbums("Artist1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(albums) != 2 {
		t.Fatalf("expected 2 albums, got %d", len(albums))
	}
	if albums[0] != "Album A" {
		t.Errorf("expected 'Album A', got '%s'", albums[0])
	}
	if albums[1] != "Album B" {
		t.Errorf("expected 'Album B', got '%s'", albums[1])
	}
}

func TestGetRandomSongs_PrefersFavourites(t *testing.T) {
	db := setupTestDB(t)
	c := New(db, t.TempDir())

	_, _ = db.Exec("INSERT INTO songs (title, artist, duration, location, status, is_favourite) VALUES ('Fav1', 'A', 180, '/tmp/fav1.mp3', 'library', 1)")
	_, _ = db.Exec("INSERT INTO songs (title, artist, duration, location, status, is_favourite) VALUES ('Fav2', 'A', 180, '/tmp/fav2.mp3', 'library', 1)")
	_, _ = db.Exec("INSERT INTO songs (title, artist, duration, location, status, is_favourite) VALUES ('Normal1', 'A', 180, '/tmp/n1.mp3', 'library', 0)")

	songs, err := c.GetRandomSongs(2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(songs) != 2 {
		t.Fatalf("expected 2 songs, got %d", len(songs))
	}

	// Both favourites should be returned
	for _, id := range songs {
		if id != 1 && id != 2 {
			t.Errorf("expected favourite song (id 1 or 2), got %d", id)
		}
	}
}
