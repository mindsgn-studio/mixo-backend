package admin

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/mindsgn-studio/mixo-backend/internal/config"
	"github.com/mindsgn-studio/mixo-backend/internal/crawler"
	"github.com/mindsgn-studio/mixo-backend/internal/queue"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestHandler(t *testing.T) *Handler {
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
	CREATE TABLE IF NOT EXISTS queue (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		song_id INTEGER NOT NULL,
		position INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		song_id INTEGER NOT NULL,
		played_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		duration_played INTEGER NOT NULL DEFAULT 0
	);
	CREATE TABLE IF NOT EXISTS state (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS playlists (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS playlist_songs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		playlist_id INTEGER NOT NULL,
		song_id INTEGER NOT NULL,
		position INTEGER NOT NULL DEFAULT 0
	);
	CREATE TABLE IF NOT EXISTS listener_snapshots (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		listener_count INTEGER NOT NULL DEFAULT 0,
		current_song_id INTEGER DEFAULT 0,
		recorded_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	cfg := &config.Config{
		SongDir: t.TempDir(),
	}
	qm := queue.New(db)
	h := New(db, qm, cfg)
	h.SetCrawler(crawler.New(db, cfg.SongDir))

	return h
}

func addTestSongToDB(t *testing.T, db *sql.DB, title, artist string, status string, duration int) int {
	t.Helper()
	result, err := db.Exec("INSERT INTO songs (title, artist, duration, location, status) VALUES (?, ?, ?, ?, ?)",
		title, artist, duration, "/tmp/"+title+".mp3", status)
	if err != nil {
		t.Fatalf("failed to add test song: %v", err)
	}
	id, _ := result.LastInsertId()
	return int(id)
}

func TestAdminPage(t *testing.T) {
	h := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	w := httptest.NewRecorder()
	h.AdminPage(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Radio Dashboard") {
		t.Error("expected 'Radio Dashboard' in response")
	}
}

func TestAdminPage_MethodNotAllowed(t *testing.T) {
	h := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/admin", nil)
	w := httptest.NewRecorder()
	h.AdminPage(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestLibraryPage(t *testing.T) {
	h := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/admin/library", nil)
	w := httptest.NewRecorder()
	h.LibraryPage(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Music Library") {
		t.Error("expected 'Music Library' in response")
	}
}

func TestLibrarySongsFragment(t *testing.T) {
	h := setupTestHandler(t)

	addTestSongToDB(t, h.db, "Test Song", "Test Artist", "library", 180)
	addTestSongToDB(t, h.db, "Deleted Song", "Deleted Artist", "deleted", 200)

	req := httptest.NewRequest(http.MethodGet, "/admin/library/songs?q=Test", nil)
	w := httptest.NewRecorder()
	h.LibrarySongsFragment(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Test Song") {
		t.Error("expected 'Test Song' in response")
	}
	if strings.Contains(w.Body.String(), "Deleted Song") {
		t.Error("should not contain 'Deleted Song'")
	}
}

func TestToggleFavourite(t *testing.T) {
	h := setupTestHandler(t)

	songID := addTestSongToDB(t, h.db, "Fav Song", "Artist", "library", 180)

	// Toggle on
	req := httptest.NewRequest(http.MethodPost, "/admin/songs/"+strconv.Itoa(songID)+"/favourite", nil)
	w := httptest.NewRecorder()
	h.ToggleFavouriteHTMX(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var fav int
	_ = h.db.QueryRow("SELECT is_favourite FROM songs WHERE id = ?", songID).Scan(&fav)
	if fav != 1 {
		t.Errorf("expected is_favourite=1, got %d", fav)
	}

	// Toggle off
	req = httptest.NewRequest(http.MethodPost, "/admin/songs/"+strconv.Itoa(songID)+"/favourite", nil)
	w = httptest.NewRecorder()
	h.ToggleFavouriteHTMX(w, req)

	_ = h.db.QueryRow("SELECT is_favourite FROM songs WHERE id = ?", songID).Scan(&fav)
	if fav != 0 {
		t.Errorf("expected is_favourite=0, got %d", fav)
	}
}

func TestCreatePlaylistHTMX(t *testing.T) {
	h := setupTestHandler(t)

	form := url.Values{}
	form.Set("name", "My Playlist")
	req := httptest.NewRequest(http.MethodPost, "/admin/playlists/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.CreatePlaylistHTMX(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var name string
	err := h.db.QueryRow("SELECT name FROM playlists WHERE name = 'My Playlist'").Scan(&name)
	if err != nil {
		t.Fatalf("expected playlist to exist: %v", err)
	}
}

func TestCreatePlaylistHTMX_EmptyName(t *testing.T) {
	h := setupTestHandler(t)

	form := url.Values{}
	form.Set("name", "")
	req := httptest.NewRequest(http.MethodPost, "/admin/playlists/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.CreatePlaylistHTMX(w, req)

	if !strings.Contains(w.Body.String(), "error") {
		t.Error("expected error for empty name")
	}
}

func TestDeletePlaylistHTMX(t *testing.T) {
	h := setupTestHandler(t)

	_, _ = h.db.Exec("INSERT INTO playlists (name) VALUES ('To Delete')")

	req := httptest.NewRequest(http.MethodDelete, "/admin/playlists/1", nil)
	w := httptest.NewRecorder()
	h.DeletePlaylistHTMX(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var count int
	_ = h.db.QueryRow("SELECT COUNT(*) FROM playlists WHERE id = 1").Scan(&count)
	if count != 0 {
		t.Errorf("expected playlist to be deleted, count=%d", count)
	}
}

func TestAddToLibraryHTMX(t *testing.T) {
	h := setupTestHandler(t)

	songID := addTestSongToDB(t, h.db, "Deleted Song", "Artist", "deleted", 180)

	req := httptest.NewRequest(http.MethodPost, "/admin/library/"+strconv.Itoa(songID)+"/add", nil)
	w := httptest.NewRecorder()
	h.AddToLibraryHTMX(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var status string
	_ = h.db.QueryRow("SELECT status FROM songs WHERE id = ?", songID).Scan(&status)
	if status != "library" {
		t.Errorf("expected status 'library', got '%s'", status)
	}
}

func TestRemoveFromLibraryHTMX(t *testing.T) {
	h := setupTestHandler(t)

	songID := addTestSongToDB(t, h.db, "Library Song", "Artist", "library", 180)

	req := httptest.NewRequest(http.MethodPost, "/admin/library/"+strconv.Itoa(songID)+"/remove", nil)
	w := httptest.NewRecorder()
	h.RemoveFromLibraryHTMX(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var status string
	_ = h.db.QueryRow("SELECT status FROM songs WHERE id = ?", songID).Scan(&status)
	if status != "deleted" {
		t.Errorf("expected status 'deleted', got '%s'", status)
	}
}

func TestEditSongPage(t *testing.T) {
	h := setupTestHandler(t)

	songID := addTestSongToDB(t, h.db, "Edit Me", "Artist", "library", 180)

	req := httptest.NewRequest(http.MethodGet, "/admin/songs/"+strconv.Itoa(songID)+"/edit", nil)
	w := httptest.NewRecorder()
	h.EditSongPage(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Edit Me") {
		t.Error("expected 'Edit Me' in response")
	}
}

func TestUpdateSongHTMX(t *testing.T) {
	h := setupTestHandler(t)

	songID := addTestSongToDB(t, h.db, "Old Title", "Old Artist", "library", 180)

	form := url.Values{}
	form.Set("title", "New Title")
	form.Set("artist", "New Artist")
	form.Set("album", "New Album")
	form.Set("genre", "Rock")
	form.Set("track_number", "5")
	form.Set("track_total", "12")
	req := httptest.NewRequest(http.MethodPost, "/admin/songs/"+strconv.Itoa(songID)+"/edit", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.UpdateSongHTMX(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var title, artist, album, genre string
	var trackNum, trackTotal int
	err := h.db.QueryRow("SELECT title, artist, album, genre, track_number, track_total FROM songs WHERE id = ?", songID).
		Scan(&title, &artist, &album, &genre, &trackNum, &trackTotal)
	if err != nil {
		t.Fatalf("failed to query song: %v", err)
	}
	if title != "New Title" {
		t.Errorf("expected 'New Title', got '%s'", title)
	}
	if artist != "New Artist" {
		t.Errorf("expected 'New Artist', got '%s'", artist)
	}
	if album != "New Album" {
		t.Errorf("expected 'New Album', got '%s'", album)
	}
	if genre != "Rock" {
		t.Errorf("expected 'Rock', got '%s'", genre)
	}
	if trackNum != 5 {
		t.Errorf("expected track 5, got %d", trackNum)
	}
	if trackTotal != 12 {
		t.Errorf("expected total 12, got %d", trackTotal)
	}
}

func TestAnalyticsDataEndpoint(t *testing.T) {
	h := setupTestHandler(t)

	_, _ = h.db.Exec("INSERT INTO listener_snapshots (listener_count, current_song_id) VALUES (5, 1)")
	_, _ = h.db.Exec("INSERT INTO listener_snapshots (listener_count, current_song_id) VALUES (10, 2)")

	req := httptest.NewRequest(http.MethodGet, "/admin/analytics/data?period=hour", nil)
	w := httptest.NewRecorder()
	h.AnalyticsDataEndpoint(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "count") {
		t.Error("expected 'count' in response")
	}
}

func TestAnalyticsPage(t *testing.T) {
	h := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/admin/analytics", nil)
	w := httptest.NewRecorder()
	h.AnalyticsPage(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Listener Analytics") {
		t.Error("expected 'Listener Analytics' in response")
	}
}

func TestPlaylistsPage(t *testing.T) {
	h := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/admin/playlists", nil)
	w := httptest.NewRecorder()
	h.PlaylistsPage(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Playlists") {
		t.Error("expected 'Playlists' in response")
	}
}

func TestDeletedPage(t *testing.T) {
	h := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/admin/deleted", nil)
	w := httptest.NewRecorder()
	h.DeletedPage(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Deleted") {
		t.Error("expected 'Deleted' in response")
	}
}

func TestHistoryPage(t *testing.T) {
	h := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/admin/history", nil)
	w := httptest.NewRecorder()
	h.HistoryPage(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Play History") {
		t.Error("expected 'Play History' in response")
	}
}

func TestAlbumsPage(t *testing.T) {
	h := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/admin/library/album", nil)
	w := httptest.NewRecorder()
	h.AlbumsPage(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestArtistsPage(t *testing.T) {
	h := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/admin/library/artist", nil)
	w := httptest.NewRecorder()
	h.ArtistsPage(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestDeleteSongHTMX(t *testing.T) {
	h := setupTestHandler(t)

	songID := addTestSongToDB(t, h.db, "To Delete", "Artist", "library", 180)

	req := httptest.NewRequest(http.MethodDelete, "/admin/songs/"+strconv.Itoa(songID), nil)
	w := httptest.NewRecorder()
	h.DeleteSongHTMX(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var status string
	_ = h.db.QueryRow("SELECT status FROM songs WHERE id = ?", songID).Scan(&status)
	if status != "deleted" {
		t.Errorf("expected status 'deleted', got '%s'", status)
	}
}

func TestAddToQueueHTMX(t *testing.T) {
	h := setupTestHandler(t)

	songID := addTestSongToDB(t, h.db, "Queue Me", "Artist", "library", 180)

	req := httptest.NewRequest(http.MethodPost, "/admin/queue/"+strconv.Itoa(songID), nil)
	w := httptest.NewRecorder()
	h.AddToQueueHTMX(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	length, _ := h.queue.Length()
	if length != 1 {
		t.Errorf("expected queue length 1, got %d", length)
	}
}

func TestRemoveFromQueueHTMX(t *testing.T) {
	h := setupTestHandler(t)

	songID := addTestSongToDB(t, h.db, "In Queue", "Artist", "library", 180)
	_ = h.queue.Add(songID)

	items, _ := h.queue.GetAll()
	if len(items) != 1 {
		t.Fatalf("expected 1 item in queue, got %d", len(items))
	}

	req := httptest.NewRequest(http.MethodDelete, "/admin/queue/"+strconv.Itoa(items[0].ID), nil)
	w := httptest.NewRecorder()
	h.RemoveFromQueueHTMX(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	length, _ := h.queue.Length()
	if length != 0 {
		t.Errorf("expected queue length 0, got %d", length)
	}
}

func TestQueueFragment_Empty(t *testing.T) {
	h := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/admin/queue", nil)
	w := httptest.NewRecorder()
	h.QueueFragment(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Queue is empty") {
		t.Error("expected 'Queue is empty' in response")
	}
}

func TestReorderQueueHTMX(t *testing.T) {
	h := setupTestHandler(t)

	songID1 := addTestSongToDB(t, h.db, "Song 1", "Artist 1", "library", 180)
	songID2 := addTestSongToDB(t, h.db, "Song 2", "Artist 2", "library", 200)
	songID3 := addTestSongToDB(t, h.db, "Song 3", "Artist 3", "library", 150)
	_ = h.queue.Add(songID1)
	_ = h.queue.Add(songID2)
	_ = h.queue.Add(songID3)

	items, _ := h.queue.GetAll()

	form := url.Values{}
	form.Set("queue_id", strconv.Itoa(items[0].ID))
	form.Set("new_position", "3")
	req := httptest.NewRequest(http.MethodPost, "/admin/queue/reorder", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	h.ReorderQueueHTMX(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	items, _ = h.queue.GetAll()
	if items[0].Song.ID != songID2 {
		t.Errorf("expected song 2 at position 1, got %d", items[0].Song.ID)
	}
}

func TestRenderLibraryFragment_Search(t *testing.T) {
	h := setupTestHandler(t)

	addTestSongToDB(t, h.db, "Rock Song", "Rock Artist", "library", 180)
	addTestSongToDB(t, h.db, "Pop Song", "Pop Artist", "library", 200)

	result := h.renderLibraryFragment("Rock", "", "")
	if !strings.Contains(result, "Rock Song") {
		t.Error("expected 'Rock Song' in search results")
	}
	if strings.Contains(result, "Pop Song") {
		t.Error("should not contain 'Pop Song' in search results")
	}
}

func TestRenderLibraryFragment_Sort(t *testing.T) {
	h := setupTestHandler(t)

	addTestSongToDB(t, h.db, "B Song", "B Artist", "library", 180)
	addTestSongToDB(t, h.db, "A Song", "A Artist", "library", 200)

	result := h.renderLibraryFragment("", "", "title")
	if !strings.Contains(result, "A Song") {
		t.Error("expected 'A Song' in results")
	}
}

func TestNowPlayingFragment(t *testing.T) {
	h := setupTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/admin/now-playing", nil)
	w := httptest.NewRecorder()
	h.NowPlayingFragment(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestSaveListenerSnapshot(t *testing.T) {
	h := setupTestHandler(t)

	h.SaveListenerSnapshot(h.db, 42, 5)

	var count, songID int
	err := h.db.QueryRow("SELECT listener_count, current_song_id FROM listener_snapshots LIMIT 1").Scan(&count, &songID)
	if err != nil {
		t.Fatalf("expected snapshot to exist: %v", err)
	}
	if count != 42 {
		t.Errorf("expected listener_count 42, got %d", count)
	}
	if songID != 5 {
		t.Errorf("expected current_song_id 5, got %d", songID)
	}
}

func TestDeletedSongsFragment(t *testing.T) {
	h := setupTestHandler(t)

	addTestSongToDB(t, h.db, "Deleted One", "Artist", "deleted", 180)

	req := httptest.NewRequest(http.MethodGet, "/admin/deleted/songs", nil)
	w := httptest.NewRecorder()
	h.DeletedSongsFragment(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Deleted One") {
		t.Error("expected 'Deleted One' in response")
	}
}

func TestHistoryFragment(t *testing.T) {
	h := setupTestHandler(t)

	songID := addTestSongToDB(t, h.db, "Played Song", "Artist", "library", 180)
	_, _ = h.db.Exec("INSERT INTO history (song_id, duration_played) VALUES (?, 60)", songID)

	req := httptest.NewRequest(http.MethodGet, "/admin/history/items", nil)
	w := httptest.NewRecorder()
	h.HistoryFragment(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Played Song") {
		t.Error("expected 'Played Song' in response")
	}
}
