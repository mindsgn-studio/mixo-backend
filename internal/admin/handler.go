package admin

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dhowden/tag"
	"github.com/mindsgn-studio/mixo-backend/internal/config"
	"github.com/mindsgn-studio/mixo-backend/internal/crawler"
	"github.com/mindsgn-studio/mixo-backend/internal/playback"
	"github.com/mindsgn-studio/mixo-backend/internal/queue"
)

type Handler struct {
	db       *sql.DB
	queue    *queue.Manager
	cfg      *config.Config
	playback *playback.Engine
	crawler  *crawler.Crawler
}

func New(db *sql.DB, q *queue.Manager, cfg *config.Config) *Handler {
	return &Handler{db: db, queue: q, cfg: cfg}
}

func (h *Handler) SetPlayback(p *playback.Engine) {
	h.playback = p
}

func (h *Handler) SetCrawler(c *crawler.Crawler) {
	h.crawler = c
}

type AddSongRequest struct {
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	Album    string `json:"album"`
	CoverArt string `json:"cover_art"`
	Duration int    `json:"duration"`
	Location string `json:"location"`
}

type SongResponse struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	Album    string `json:"album"`
	CoverArt string `json:"cover_art"`
	Duration int    `json:"duration"`
	Location string `json:"location"`
}

type QueueItemResponse struct {
	ID       int          `json:"id"`
	Position int          `json:"position"`
	Song     SongResponse `json:"song"`
}

type HistoryItemResponse struct {
	ID             int          `json:"id"`
	Song           SongResponse `json:"song"`
	PlayedAt       time.Time    `json:"played_at"`
	DurationPlayed int          `json:"duration_played"`
}

type NowPlayingResponse struct {
	Song *SongResponse `json:"song,omitempty"`
}

// AddSong adds a new song to the database
func (h *Handler) AddSong(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req AddSongRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Title == "" || req.Artist == "" || req.Duration <= 0 || req.Location == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	result, err := h.db.Exec("INSERT INTO songs (title, artist, album, cover_art, duration, location) VALUES (?, ?, ?, ?, ?, ?)",
		req.Title, req.Artist, req.Album, req.CoverArt, req.Duration, req.Location)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to add song: %v", err), http.StatusInternalServerError)
		return
	}

	id, _ := result.LastInsertId()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(SongResponse{
		ID:       int(id),
		Title:    req.Title,
		Artist:   req.Artist,
		Album:    req.Album,
		CoverArt: req.CoverArt,
		Duration: req.Duration,
		Location: req.Location,
	}); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
	}
}

// ListSongs returns all songs in the database
func (h *Handler) ListSongs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rows, err := h.db.Query("SELECT id, title, artist, album, cover_art, duration, location FROM songs ORDER BY created_at DESC")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list songs: %v", err), http.StatusInternalServerError)
		return
	}
	defer func() {
		if err := rows.Close(); err != nil {
			http.Error(w, fmt.Sprintf("Failed to close rows: %v", err), http.StatusInternalServerError)
		}
	}()

	var songs []SongResponse
	for rows.Next() {
		var song SongResponse
		err := rows.Scan(&song.ID, &song.Title, &song.Artist, &song.Album, &song.CoverArt, &song.Duration, &song.Location)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to scan song: %v", err), http.StatusInternalServerError)
			return
		}
		songs = append(songs, song)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(songs); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
	}
}

// DeleteSong removes a song from the database
func (h *Handler) DeleteSong(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.URL.Path[len("/api/songs/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid song ID", http.StatusBadRequest)
		return
	}

	result, err := h.db.Exec("DELETE FROM songs WHERE id = ?", id)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete song: %v", err), http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Song not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// AddToQueue adds a song to the playback queue
func (h *Handler) AddToQueue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	songIDStr := r.URL.Path[len("/api/queue/"):]
	songID, err := strconv.Atoi(songIDStr)
	if err != nil {
		http.Error(w, "Invalid song ID", http.StatusBadRequest)
		return
	}

	// Check if song exists
	var exists bool
	err = h.db.QueryRow("SELECT EXISTS(SELECT 1 FROM songs WHERE id = ?)", songID).Scan(&exists)
	if err != nil || !exists {
		http.Error(w, "Song not found", http.StatusNotFound)
		return
	}

	if err := h.queue.Add(songID); err != nil {
		http.Error(w, fmt.Sprintf("Failed to add to queue: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// GetQueue returns the current playback queue
func (h *Handler) GetQueue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	items, err := h.queue.GetAll()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get queue: %v", err), http.StatusInternalServerError)
		return
	}

	var response []QueueItemResponse
	for _, item := range items {
		response = append(response, QueueItemResponse{
			ID:       item.ID,
			Position: item.Position,
			Song: SongResponse{
				ID:       item.Song.ID,
				Title:    item.Song.Title,
				Artist:   item.Song.Artist,
				Duration: item.Song.Duration,
				Location: item.Song.Location,
			},
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
	}
}

// RemoveFromQueue removes a song from the playback queue
func (h *Handler) RemoveFromQueue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.URL.Path[len("/api/queue/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid queue item ID", http.StatusBadRequest)
		return
	}

	if err := h.queue.Remove(id); err != nil {
		http.Error(w, fmt.Sprintf("Failed to remove from queue: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// NowPlaying returns the currently playing song
func (h *Handler) NowPlaying(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var songID int
	err := h.db.QueryRow("SELECT value FROM state WHERE key = 'current_song'").Scan(&songID)
	if err != nil {
		if err == sql.ErrNoRows {
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(NowPlayingResponse{Song: nil}); err != nil {
				http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
			}
			return
		}
		http.Error(w, fmt.Sprintf("Failed to get current song: %v", err), http.StatusInternalServerError)
		return
	}

	var song SongResponse
	err = h.db.QueryRow("SELECT id, title, artist, album, cover_art, duration, location FROM songs WHERE id = ?", songID).
		Scan(&song.ID, &song.Title, &song.Artist, &song.Album, &song.CoverArt, &song.Duration, &song.Location)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get song details: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(NowPlayingResponse{Song: &song}); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
	}
}

// GetHistory returns playback history
func (h *Handler) GetHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	rows, err := h.db.Query(`
		SELECT h.id, h.played_at, h.duration_played, s.id, s.title, s.artist, s.duration, s.location
		FROM history h
		JOIN songs s ON h.song_id = s.id
		ORDER BY h.played_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get history: %v", err), http.StatusInternalServerError)
		return
	}
	defer func() {
		if err := rows.Close(); err != nil {
			http.Error(w, fmt.Sprintf("Failed to close rows: %v", err), http.StatusInternalServerError)
		}
	}()

	var history []HistoryItemResponse
	for rows.Next() {
		var item HistoryItemResponse
		err := rows.Scan(&item.ID, &item.PlayedAt, &item.DurationPlayed,
			&item.Song.ID, &item.Song.Title, &item.Song.Artist, &item.Song.Duration, &item.Song.Location)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to scan history item: %v", err), http.StatusInternalServerError)
			return
		}
		history = append(history, item)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(history); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
	}
}

// UploadSong handles MP3 file uploads
func (h *Handler) UploadSong(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form (max 10MB)
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "No file provided", http.StatusBadRequest)
		return
	}
	defer func() {
		if err := file.Close(); err != nil {
			http.Error(w, fmt.Sprintf("Failed to close file: %v", err), http.StatusInternalServerError)
		}
	}()

	// Validate file extension
	ext := filepath.Ext(header.Filename)
	if ext != ".mp3" {
		http.Error(w, "Only MP3 files are allowed", http.StatusBadRequest)
		return
	}

	// Read file to extract metadata
	metadata, err := tag.ReadFrom(file)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read metadata: %v", err), http.StatusBadRequest)
		return
	}

	// Extract metadata
	title := metadata.Title()
	if title == "" {
		title = header.Filename[:len(header.Filename)-len(ext)]
	}
	artist := metadata.Artist()
	if artist == "" {
		artist = "Unknown Artist"
	}
	album := metadata.Album()
	coverArt := ""

	// Extract cover art if available
	if metadata.Picture() != nil {
		coverPath := filepath.Join(h.cfg.SongDir, ".covers", fmt.Sprintf("%d.jpg", time.Now().UnixNano()))
		if err := os.MkdirAll(filepath.Dir(coverPath), 0755); err == nil {
			if err := os.WriteFile(coverPath, metadata.Picture().Data, 0644); err == nil {
				coverArt = coverPath
			}
		}
	}

	// Reset file pointer for duration check
	if _, err := file.Seek(0, 0); err != nil {
		http.Error(w, fmt.Sprintf("Failed to reset file pointer: %v", err), http.StatusInternalServerError)
		return
	}

	// Get duration using FFprobe
	duration, err := getDuration(file)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get duration: %v", err), http.StatusBadRequest)
		return
	}

	// Reset file pointer for copying
	if _, err := file.Seek(0, 0); err != nil {
		http.Error(w, fmt.Sprintf("Failed to reset file pointer: %v", err), http.StatusInternalServerError)
		return
	}

	// Ensure song directory exists
	if err := os.MkdirAll(h.cfg.SongDir, 0755); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create song directory: %v", err), http.StatusInternalServerError)
		return
	}

	// Generate unique filename
	filename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), header.Filename)
	filePath := filepath.Join(h.cfg.SongDir, filename)

	// Create the file
	dst, err := os.Create(filePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create file: %v", err), http.StatusInternalServerError)
		return
	}
	defer func() {
		if err := dst.Close(); err != nil {
			http.Error(w, fmt.Sprintf("Failed to close destination file: %v", err), http.StatusInternalServerError)
		}
	}()

	// Copy the uploaded file
	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save file: %v", err), http.StatusInternalServerError)
		return
	}

	// Save to database
	result, err := h.db.Exec("INSERT INTO songs (title, artist, album, cover_art, duration, location) VALUES (?, ?, ?, ?, ?, ?)",
		title, artist, album, coverArt, duration, filePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to add song: %v", err), http.StatusInternalServerError)
		return
	}

	id, _ := result.LastInsertId()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(SongResponse{
		ID:       int(id),
		Title:    title,
		Artist:   artist,
		Album:    album,
		CoverArt: coverArt,
		Duration: duration,
		Location: filePath,
	}); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
	}
}

// getDuration uses FFprobe to get the duration of an audio file
func getDuration(file io.Reader) (int, error) {
	// Create a temporary file to store the content
	tmpFile, err := os.CreateTemp("", "upload-*.mp3")
	if err != nil {
		return 0, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() {
		if err := tmpFile.Close(); err != nil {
			fmt.Printf("Warning: failed to close temp file: %v\n", err)
		}
	}()
	defer func() {
		if err := os.Remove(tmpFile.Name()); err != nil {
			fmt.Printf("Warning: failed to remove temp file: %v\n", err)
		}
	}()

	// Copy the reader to the temp file
	if _, err := io.Copy(tmpFile, file); err != nil {
		return 0, fmt.Errorf("failed to copy to temp file: %w", err)
	}

	// Run FFprobe to get duration
	cmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", tmpFile.Name())
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("ffprobe failed: %w", err)
	}

	// Parse the duration
	durationStr := strings.TrimSpace(string(output))
	duration, err := strconv.ParseFloat(durationStr, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse duration: %w", err)
	}

	return int(duration), nil
}

// ==================== HTMX HANDLERS ====================

// AdminPage renders the full admin page
func (h *Handler) AdminPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	nowPlayingHTML := h.renderNowPlayingFragment()
	songsHTML := h.renderSongsFragment()
	queueHTML := h.renderQueueFragment()

	w.Header().Set("Content-Type", "text/html")
	result := adminPageTemplate
	result = strings.Replace(result, "{{NOW_PLAYING}}", nowPlayingHTML, 1)
	result = strings.Replace(result, "{{SONGS}}", songsHTML, 1)
	result = strings.Replace(result, "{{QUEUE}}", queueHTML, 1)
	fmt.Fprint(w, result) //nolint:errcheck
}

// SongsFragment returns HTML table of songs
func (h *Handler) SongsFragment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	_, _ = w.Write([]byte(h.renderSongsFragment()))
}

// QueueFragment returns HTML table of queue
func (h *Handler) QueueFragment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	_, _ = w.Write([]byte(h.renderQueueFragment()))
}

// NowPlayingFragment returns now playing status HTML
func (h *Handler) NowPlayingFragment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	_, _ = w.Write([]byte(h.renderNowPlayingFragment()))
}

// CoverArtHandler serves cover art images for songs
func (h *Handler) CoverArtHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.URL.Path[len("/admin/cover/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid song ID", http.StatusBadRequest)
		return
	}

	var coverArt string
	err = h.db.QueryRow("SELECT cover_art FROM songs WHERE id = ?", id).Scan(&coverArt)
	if err != nil || coverArt == "" {
		http.NotFound(w, r)
		return
	}

	http.ServeFile(w, r, coverArt)
}

// PlayControl handles play/stop toggle
func (h *Handler) PlayControl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.playback == nil {
		http.Error(w, "Playback engine not available", http.StatusServiceUnavailable)
		return
	}

	if h.playback.IsPaused() {
		h.playback.Resume()
	} else {
		h.playback.Pause()
	}

	w.Header().Set("Content-Type", "text/html")
	_, _ = w.Write([]byte(h.renderNowPlayingFragment()))
}

// UploadSongHTMX handles file upload for HTMX
func (h *Handler) UploadSongHTMX(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form (max 10MB)
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, "Failed to parse form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, "No file provided")
		return
	}
	defer func() {
		if err := file.Close(); err != nil {
			http.Error(w, fmt.Sprintf("Failed to close file: %v", err), http.StatusInternalServerError)
		}
	}()

	// Validate file extension
	ext := filepath.Ext(header.Filename)
	if ext != ".mp3" {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, "Only MP3 files are allowed")
		return
	}

	// Read file to extract metadata
	metadata, err := tag.ReadFrom(file)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, fmt.Sprintf("Failed to read metadata: %v", err))
		return
	}

	// Extract metadata
	title := metadata.Title()
	if title == "" {
		title = header.Filename[:len(header.Filename)-len(ext)]
	}
	artist := metadata.Artist()
	if artist == "" {
		artist = "Unknown Artist"
	}

	// Reset file pointer for duration check
	if _, err := file.Seek(0, 0); err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, "Failed to reset file pointer")
		return
	}

	// Get duration using FFprobe
	duration, err := getDuration(file)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, fmt.Sprintf("Failed to get duration: %v", err))
		return
	}

	// Reset file pointer for copying
	if _, err := file.Seek(0, 0); err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, "Failed to reset file pointer")
		return
	}

	// Ensure song directory exists
	if err := os.MkdirAll(h.cfg.SongDir, 0755); err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, fmt.Sprintf("Failed to create song directory: %v", err))
		return
	}

	// Generate unique filename
	filename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), header.Filename)
	filePath := filepath.Join(h.cfg.SongDir, filename)

	// Create the file
	dst, err := os.Create(filePath)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, fmt.Sprintf("Failed to create file: %v", err))
		return
	}
	defer func() {
		if err := dst.Close(); err != nil {
			http.Error(w, fmt.Sprintf("Failed to close file: %v", err), http.StatusInternalServerError)
		}
	}()
	if _, err := io.Copy(dst, file); err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, fmt.Sprintf("Failed to save file: %v", err))
		return
	}

	// Save to database
	_, err = h.db.Exec("INSERT INTO songs (title, artist, duration, location) VALUES (?, ?, ?, ?)",
		title, artist, duration, filePath)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, fmt.Sprintf("Failed to add song: %v", err))
		return
	}

	w.Header().Set("Content-Type", "text/html")
	_, _ = fmt.Fprintf(w, messageSuccessTemplate, fmt.Sprintf("Song '%s' uploaded successfully!", title))
}

// AddToQueueHTMX adds a song to queue and returns HTML message
func (h *Handler) AddToQueueHTMX(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract song ID from URL path /admin/queue/{id}
	path := r.URL.Path
	prefix := "/admin/queue/"
	if !strings.HasPrefix(path, prefix) {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	idStr := path[len(prefix):]
	songID, err := strconv.Atoi(idStr)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, "Invalid song ID")
		return
	}

	// Check if song exists
	var exists bool
	err = h.db.QueryRow("SELECT EXISTS(SELECT 1 FROM songs WHERE id = ?)", songID).Scan(&exists)
	if err != nil || !exists {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, "Song not found")
		return
	}

	var title string
	err = h.db.QueryRow("SELECT title FROM songs WHERE id = ?", songID).Scan(&title)
	if err != nil {
		title = "Song"
	}

	if err := h.queue.Add(songID); err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, fmt.Sprintf("Failed to add to queue: %v", err))
		return
	}

	w.Header().Set("Content-Type", "text/html")
	_, _ = fmt.Fprintf(w, messageSuccessTemplate, fmt.Sprintf("'%s' added to queue!", title))
}

// RemoveFromQueueHTMX removes a song from queue and returns HTML message
func (h *Handler) RemoveFromQueueHTMX(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract queue item ID from URL path /admin/queue/{id}
	path := r.URL.Path
	prefix := "/admin/queue/"
	if !strings.HasPrefix(path, prefix) {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	idStr := path[len(prefix):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, "Invalid queue item ID")
		return
	}

	if err := h.queue.Remove(id); err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, fmt.Sprintf("Failed to remove from queue: %v", err))
		return
	}

	w.Header().Set("Content-Type", "text/html")
	_, _ = fmt.Fprintf(w, messageSuccessTemplate, "Removed from queue!")
}

// DeleteSongHTMX deletes a song from database only (keeps file on disk)
func (h *Handler) DeleteSongHTMX(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract song ID from URL path /admin/songs/{id}
	path := r.URL.Path
	prefix := "/admin/songs/"
	if !strings.HasPrefix(path, prefix) {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	idStr := path[len(prefix):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, "Invalid song ID")
		return
	}

	// Delete from database only (not from filesystem)
	result, err := h.db.Exec("DELETE FROM songs WHERE id = ?", id)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, fmt.Sprintf("Failed to delete song: %v", err))
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, "Song not found")
		return
	}

	w.Header().Set("Content-Type", "text/html")
	_, _ = fmt.Fprintf(w, messageSuccessTemplate, "Song removed from library!")
}

// ==================== HELPER METHODS ====================

func (h *Handler) renderSongsFragment() string {
	rows, err := h.db.Query("SELECT id, title, artist, album, cover_art, duration, location FROM songs ORDER BY created_at DESC")
	if err != nil {
		return emptySongsTemplate
	}
	defer func() {
		_ = rows.Close()
	}()

	var rowsHTML string
	count := 0
	for rows.Next() {
		var song SongResponse
		err := rows.Scan(&song.ID, &song.Title, &song.Artist, &song.Album, &song.CoverArt, &song.Duration, &song.Location)
		if err != nil {
			continue
		}
		rowsHTML += fmt.Sprintf(songRowTemplate, song.Title, song.Artist, song.Duration, song.ID, song.ID)
		count++
	}

	if count == 0 {
		return emptySongsTemplate
	}

	return fmt.Sprintf(songsTableTemplate, rowsHTML)
}

func (h *Handler) renderQueueFragment() string {
	items, err := h.queue.GetAll()
	if err != nil {
		return emptyQueueTemplate
	}

	if len(items) == 0 {
		return emptyQueueTemplate
	}

	var rowsHTML string
	for _, item := range items {
		rowsHTML += fmt.Sprintf(queueRowTemplate, item.Position, item.Song.Title, item.Song.Artist, item.Song.Duration, item.ID)
	}

	return fmt.Sprintf(queueTableTemplate, rowsHTML)
}

func (h *Handler) renderNowPlayingFragment() string {
	var songID int
	err := h.db.QueryRow("SELECT value FROM state WHERE key = 'current_song'").Scan(&songID)
	if err != nil {
		// No song playing
		if h.playback != nil && h.playback.IsPaused() {
			return fmt.Sprintf(nowPlayingEmptyTemplate, "paused", "Paused", "play", "▶ Play")
		}
		return fmt.Sprintf(nowPlayingEmptyTemplate, "playing", "Waiting for queue", "stop", "⏸ Pause")
	}

	var song SongResponse
	err = h.db.QueryRow("SELECT id, title, artist, album, cover_art, duration, location FROM songs WHERE id = ?", songID).
		Scan(&song.ID, &song.Title, &song.Artist, &song.Album, &song.CoverArt, &song.Duration, &song.Location)
	if err != nil {
		if h.playback != nil && h.playback.IsPaused() {
			return fmt.Sprintf(nowPlayingEmptyTemplate, "paused", "Paused", "play", "▶ Play")
		}
		return fmt.Sprintf(nowPlayingEmptyTemplate, "playing", "Waiting for queue", "stop", "⏸ Pause")
	}

	coverHTML := ""
	if song.CoverArt != "" {
		coverHTML = fmt.Sprintf(`<img src="/admin/cover/%d" alt="Cover" style="width:100px;height:100px;border-radius:5px;margin-bottom:10px;">`, song.ID)
	}

	albumText := ""
	if song.Album != "" {
		albumText = fmt.Sprintf("Album: %s", song.Album)
	}

	if h.playback != nil && h.playback.IsPaused() {
		return fmt.Sprintf(nowPlayingTemplate, coverHTML, song.Title, song.Artist, albumText, song.Duration, "paused", "Paused", "play", "▶ Play")
	}
	return fmt.Sprintf(nowPlayingTemplate, coverHTML, song.Title, song.Artist, albumText, song.Duration, "playing", "Playing", "stop", "⏸ Pause")
}
