package admin

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

type ListenerCounter interface {
	ClientCount() int
}

type Handler struct {
	db           *sql.DB
	queue        *queue.Manager
	cfg          *config.Config
	playback     *playback.Engine
	crawler      *crawler.Crawler
	broadcaster  ListenerCounter
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

func (h *Handler) SetBroadcaster(b ListenerCounter) {
	h.broadcaster = b
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
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Artist      string `json:"artist"`
	Album       string `json:"album"`
	Genre       string `json:"genre"`
	TrackNumber int    `json:"track_number"`
	TrackTotal  int    `json:"track_total"`
	CoverArt    string `json:"cover_art"`
	Duration    int    `json:"duration"`
	DurationFmt string `json:"duration_fmt"`
	Location    string `json:"location"`
	Status      string `json:"status"`
	IsFavourite bool   `json:"is_favourite"`
}

type QueueItemResponse struct {
	ID               int          `json:"id"`
	Position         int          `json:"position"`
	Song             SongResponse `json:"song"`
	EstimatedPlayAt  string       `json:"estimated_play_at"`
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

func formatDuration(seconds int) string {
	m := seconds / 60
	s := seconds % 60
	return fmt.Sprintf("%02d:%02d", m, s)
}

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

	stationPath, err := playback.NormalizeToStationMP3(req.Location, playback.StationCacheDir(h.cfg.SongDir))
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to normalize audio: %v", err), http.StatusInternalServerError)
		return
	}

	result, err := h.db.Exec("INSERT INTO songs (title, artist, album, cover_art, duration, location, source_location, normalized, status) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'library')",
		req.Title, req.Artist, req.Album, req.CoverArt, req.Duration, stationPath, req.Location, 1)
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
		Location: stationPath,
	}); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
	}
}

func (h *Handler) ListSongs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rows, err := h.db.Query("SELECT id, title, artist, album, genre, track_number, track_total, cover_art, duration, location, status, is_favourite FROM songs WHERE status = 'library' ORDER BY artist, title")
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
		var fav int
		err := rows.Scan(&song.ID, &song.Title, &song.Artist, &song.Album, &song.Genre, &song.TrackNumber, &song.TrackTotal, &song.CoverArt, &song.Duration, &song.Location, &song.Status, &fav)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to scan song: %v", err), http.StatusInternalServerError)
			return
		}
		song.DurationFmt = formatDuration(song.Duration)
		song.IsFavourite = fav == 1
		songs = append(songs, song)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(songs); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
	}
}

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

	result, err := h.db.Exec("UPDATE songs SET status = 'deleted' WHERE id = ?", id)
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
				ID:          item.Song.ID,
				Title:       item.Song.Title,
				Artist:      item.Song.Artist,
				Album:       item.Song.Album,
				TrackNumber: item.Song.TrackNumber,
				Duration:    item.Song.Duration,
				Location:    item.Song.Location,
			},
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
	}
}

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
	var fav int
	err = h.db.QueryRow("SELECT id, title, artist, album, genre, track_number, track_total, cover_art, duration, location, status, is_favourite FROM songs WHERE id = ?", songID).
		Scan(&song.ID, &song.Title, &song.Artist, &song.Album, &song.Genre, &song.TrackNumber, &song.TrackTotal, &song.CoverArt, &song.Duration, &song.Location, &song.Status, &fav)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get song details: %v", err), http.StatusInternalServerError)
		return
	}
	song.IsFavourite = fav == 1
	song.DurationFmt = formatDuration(song.Duration)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(NowPlayingResponse{Song: &song}); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
	}
}

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
		SELECT h.id, h.played_at, h.duration_played, s.id, s.title, s.artist, s.album, s.genre, s.track_number, s.duration, s.location
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
			&item.Song.ID, &item.Song.Title, &item.Song.Artist, &item.Song.Album, &item.Song.Genre, &item.Song.TrackNumber, &item.Song.Duration, &item.Song.Location)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to scan history item: %v", err), http.StatusInternalServerError)
			return
		}
		item.Song.DurationFmt = formatDuration(item.Song.Duration)
		history = append(history, item)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(history); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
	}
}

func (h *Handler) UploadSong(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

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

	ext := filepath.Ext(header.Filename)
	if ext != ".mp3" {
		http.Error(w, "Only MP3 files are allowed", http.StatusBadRequest)
		return
	}

	metadata, err := tag.ReadFrom(file)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read metadata: %v", err), http.StatusBadRequest)
		return
	}

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

	if metadata.Picture() != nil {
		coverPath := filepath.Join(h.cfg.SongDir, ".covers", fmt.Sprintf("%d.jpg", time.Now().UnixNano()))
		if err := os.MkdirAll(filepath.Dir(coverPath), 0755); err == nil {
			if err := os.WriteFile(coverPath, metadata.Picture().Data, 0644); err == nil {
				coverArt = coverPath
			}
		}
	}

	if _, err := file.Seek(0, 0); err != nil {
		http.Error(w, fmt.Sprintf("Failed to reset file pointer: %v", err), http.StatusInternalServerError)
		return
	}

	duration, err := getDuration(file)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get duration: %v", err), http.StatusBadRequest)
		return
	}

	if _, err := file.Seek(0, 0); err != nil {
		http.Error(w, fmt.Sprintf("Failed to reset file pointer: %v", err), http.StatusInternalServerError)
		return
	}

	if err := os.MkdirAll(h.cfg.SongDir, 0755); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create song directory: %v", err), http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), header.Filename)
	filePath := filepath.Join(h.cfg.SongDir, filename)

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

	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save file: %v", err), http.StatusInternalServerError)
		return
	}

	stationPath, err := playback.NormalizeToStationMP3(filePath, playback.StationCacheDir(h.cfg.SongDir))
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to normalize audio: %v", err), http.StatusInternalServerError)
		return
	}

	result, err := h.db.Exec("INSERT INTO songs (title, artist, album, cover_art, duration, location, source_location, normalized, status, added_to_library_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'library', CURRENT_TIMESTAMP)",
		title, artist, album, coverArt, duration, stationPath, filePath, 1)
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
		Location: stationPath,
	}); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
	}
}

func getDuration(file io.Reader) (int, error) {
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

	if _, err := io.Copy(tmpFile, file); err != nil {
		return 0, fmt.Errorf("failed to copy to temp file: %w", err)
	}

	cmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", tmpFile.Name())
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("ffprobe failed: %w", err)
	}

	durationStr := strings.TrimSpace(string(output))
	duration, err := strconv.ParseFloat(durationStr, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse duration: %w", err)
	}

	return int(duration), nil
}

// ==================== HTMX HANDLERS ====================

func (h *Handler) AdminPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	nowPlayingHTML := h.renderNowPlayingFragment()
	queueHTML := h.renderQueueFragment()

	w.Header().Set("Content-Type", "text/html")
	result := adminPageTemplate
	result = strings.Replace(result, "{{NOW_PLAYING}}", nowPlayingHTML, 1)
	result = strings.Replace(result, "{{QUEUE}}", queueHTML, 1)
	fmt.Fprint(w, result)
}

func (h *Handler) SongsFragment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	_, _ = w.Write([]byte(h.renderSongsFragment()))
}

func (h *Handler) QueueFragment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	_, _ = w.Write([]byte(h.renderQueueFragment()))
}

func (h *Handler) NowPlayingFragment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var songID int
	_ = h.db.QueryRow("SELECT value FROM state WHERE key = 'current_song'").Scan(&songID)
	etag := fmt.Sprintf(`"song-%d"`, songID)

	if match := r.Header.Get("If-None-Match"); match == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Content-Type", "text/html")
	_, _ = w.Write([]byte(h.renderNowPlayingFragment()))
}

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

	w.Header().Set("Cache-Control", "public, max-age=3600")
	http.ServeFile(w, r, coverArt)
}

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

func (h *Handler) UploadSongHTMX(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

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

	ext := filepath.Ext(header.Filename)
	if ext != ".mp3" {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, "Only MP3 files are allowed")
		return
	}

	metadata, err := tag.ReadFrom(file)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, fmt.Sprintf("Failed to read metadata: %v", err))
		return
	}

	title := metadata.Title()
	if title == "" {
		title = header.Filename[:len(header.Filename)-len(ext)]
	}
	artist := metadata.Artist()
	if artist == "" {
		artist = "Unknown Artist"
	}

	if _, err := file.Seek(0, 0); err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, "Failed to reset file pointer")
		return
	}

	duration, err := getDuration(file)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, fmt.Sprintf("Failed to get duration: %v", err))
		return
	}

	if _, err := file.Seek(0, 0); err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, "Failed to reset file pointer")
		return
	}

	if err := os.MkdirAll(h.cfg.SongDir, 0755); err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, fmt.Sprintf("Failed to create song directory: %v", err))
		return
	}

	filename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), header.Filename)
	filePath := filepath.Join(h.cfg.SongDir, filename)

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

	stationPath, err := playback.NormalizeToStationMP3(filePath, playback.StationCacheDir(h.cfg.SongDir))
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, fmt.Sprintf("Failed to normalize audio: %v", err))
		return
	}

	_, err = h.db.Exec("INSERT INTO songs (title, artist, duration, location, source_location, normalized, status, added_to_library_at) VALUES (?, ?, ?, ?, ?, ?, 'library', CURRENT_TIMESTAMP)",
		title, artist, duration, stationPath, filePath, 1)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, fmt.Sprintf("Failed to add song: %v", err))
		return
	}

	w.Header().Set("Content-Type", "text/html")
	_, _ = fmt.Fprintf(w, messageSuccessTemplate, fmt.Sprintf("Song '%s' uploaded successfully!", title))
}

func (h *Handler) AddToQueueHTMX(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

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

func (h *Handler) RemoveFromQueueHTMX(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

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

func (h *Handler) RescanHTMX(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if h.crawler == nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, "Crawler not available")
		return
	}

	crawlDirs := strings.Split(h.cfg.CrawlDirs, ",")
	if len(crawlDirs) == 0 || (len(crawlDirs) == 1 && crawlDirs[0] == "") {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, "No music directories configured (CRAWL_DIRS)")
		return
	}

	if err := h.crawler.ScanDirectories(crawlDirs); err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, fmt.Sprintf("Rescan failed: %v", err))
		return
	}

	count, err := h.crawler.GetSongCount()
	if err != nil {
		count = 0
	}

	w.Header().Set("Content-Type", "text/html")
	_, _ = fmt.Fprintf(w, messageSuccessTemplate, fmt.Sprintf("Rescan complete! %d songs in database.", count))
}

func (h *Handler) DeleteSongHTMX(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

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

	result, err := h.db.Exec("UPDATE songs SET status = 'deleted' WHERE id = ?", id)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, fmt.Sprintf("Failed to remove song: %v", err))
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

// ==================== QUEUE REORDER ====================

func (h *Handler) ReorderQueueHTMX(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, "Invalid form data")
		return
	}

	queueID, err := strconv.Atoi(r.FormValue("queue_id"))
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, "Invalid queue ID")
		return
	}

	newPosition, err := strconv.Atoi(r.FormValue("new_position"))
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, "Invalid position")
		return
	}

	if err := h.queue.Reorder(queueID, newPosition); err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, fmt.Sprintf("Failed to reorder: %v", err))
		return
	}

	w.Header().Set("Content-Type", "text/html")
	_, _ = w.Write([]byte(h.renderQueueFragment()))
}

// ==================== FAVOURITES ====================

func (h *Handler) ToggleFavouriteHTMX(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := r.URL.Path
	prefix := "/admin/songs/"
	suffix := "/favourite"
	if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, suffix) {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	idStr := path[len(prefix) : len(path)-len(suffix)]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, "Invalid song ID")
		return
	}

	var currentFav int
	err = h.db.QueryRow("SELECT is_favourite FROM songs WHERE id = ?", id).Scan(&currentFav)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, "Song not found")
		return
	}

	newFav := 0
	if currentFav == 0 {
		newFav = 1
	}

	_, err = h.db.Exec("UPDATE songs SET is_favourite = ? WHERE id = ?", newFav, id)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, fmt.Sprintf("Failed to update favourite: %v", err))
		return
	}

	if newFav == 1 {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageSuccessTemplate, "Added to favourites!")
	} else {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageSuccessTemplate, "Removed from favourites!")
	}
}

// ==================== PLAYLISTS ====================

func (h *Handler) PlaylistsPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	playlistsHTML := h.renderPlaylistsFragment()

	w.Header().Set("Content-Type", "text/html")
	result := playlistsPageTemplate
	result = strings.Replace(result, "{{PLAYLISTS}}", playlistsHTML, 1)
	fmt.Fprint(w, result)
}

func (h *Handler) PlaylistsFragment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	_, _ = w.Write([]byte(h.renderPlaylistsFragment()))
}

func (h *Handler) CreatePlaylistHTMX(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, "Invalid form data")
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, "Playlist name is required")
		return
	}

	_, err := h.db.Exec("INSERT INTO playlists (name) VALUES (?)", name)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, fmt.Sprintf("Failed to create playlist: %v", err))
		return
	}

	w.Header().Set("Content-Type", "text/html")
	_, _ = fmt.Fprintf(w, messageSuccessTemplate, fmt.Sprintf("Playlist '%s' created!", name))
}

func (h *Handler) DeletePlaylistHTMX(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := r.URL.Path
	prefix := "/admin/playlists/"
	if !strings.HasPrefix(path, prefix) {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	idStr := path[len(prefix):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, "Invalid playlist ID")
		return
	}

	_, err = h.db.Exec("DELETE FROM playlists WHERE id = ?", id)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, fmt.Sprintf("Failed to delete playlist: %v", err))
		return
	}

	w.Header().Set("Content-Type", "text/html")
	_, _ = fmt.Fprintf(w, messageSuccessTemplate, "Playlist deleted!")
}

func (h *Handler) PlaylistDetailPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := r.URL.Path
	prefix := "/admin/playlists/"
	if !strings.HasPrefix(path, prefix) {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	idStr := path[len(prefix):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid playlist ID", http.StatusBadRequest)
		return
	}

	var name string
	err = h.db.QueryRow("SELECT name FROM playlists WHERE id = ?", id).Scan(&name)
	if err != nil {
		http.Error(w, "Playlist not found", http.StatusNotFound)
		return
	}

	songsHTML := h.renderPlaylistSongsFragment(id)

	w.Header().Set("Content-Type", "text/html")
	result := playlistDetailPageTemplate
	result = strings.Replace(result, "{{PLAYLIST_NAME}}", name, 1)
	result = strings.Replace(result, "{{PLAYLIST_ID}}", idStr, 1)
	result = strings.Replace(result, "{{SONGS}}", songsHTML, 1)
	fmt.Fprint(w, result)
}

func (h *Handler) PlaylistSongsFragment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := r.URL.Path
	prefix := "/admin/playlists/"
	suffix := "/songs"
	if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, suffix) {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	idStr := path[len(prefix) : len(path)-len(suffix)]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid playlist ID", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	_, _ = w.Write([]byte(h.renderPlaylistSongsFragment(id)))
}

func (h *Handler) AddToPlaylistHTMX(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := r.URL.Path
	prefix := "/admin/playlists/"
	mid := "/add/"
	if !strings.HasPrefix(path, prefix) || !strings.Contains(path, mid) {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	rest := path[len(prefix):]
	parts := strings.SplitN(rest, mid, 2)
	if len(parts) != 2 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	playlistID, err := strconv.Atoi(parts[0])
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, "Invalid playlist ID")
		return
	}

	songID, err := strconv.Atoi(parts[1])
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, "Invalid song ID")
		return
	}

	var maxPos int
	_ = h.db.QueryRow("SELECT COALESCE(MAX(position), 0) FROM playlist_songs WHERE playlist_id = ?", playlistID).Scan(&maxPos)

	_, err = h.db.Exec("INSERT INTO playlist_songs (playlist_id, song_id, position) VALUES (?, ?, ?)", playlistID, songID, maxPos+1)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, fmt.Sprintf("Failed to add to playlist: %v", err))
		return
	}

	w.Header().Set("Content-Type", "text/html")
	_, _ = fmt.Fprintf(w, messageSuccessTemplate, "Song added to playlist!")
}

func (h *Handler) RemoveFromPlaylistHTMX(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := r.URL.Path
	prefix := "/admin/playlists/"
	mid := "/remove/"
	if !strings.HasPrefix(path, prefix) || !strings.Contains(path, mid) {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	rest := path[len(prefix):]
	parts := strings.SplitN(rest, mid, 2)
	if len(parts) != 2 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	playlistID, err := strconv.Atoi(parts[0])
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, "Invalid playlist ID")
		return
	}

	songID, err := strconv.Atoi(parts[1])
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, "Invalid song ID")
		return
	}

	_, err = h.db.Exec("DELETE FROM playlist_songs WHERE playlist_id = ? AND song_id = ?", playlistID, songID)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, fmt.Sprintf("Failed to remove from playlist: %v", err))
		return
	}

	w.Header().Set("Content-Type", "text/html")
	_, _ = fmt.Fprintf(w, messageSuccessTemplate, "Song removed from playlist!")
}

func (h *Handler) QueuePlaylistHTMX(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := r.URL.Path
	prefix := "/admin/playlists/"
	suffix := "/queue"
	if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, suffix) {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	idStr := path[len(prefix) : len(path)-len(suffix)]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, "Invalid playlist ID")
		return
	}

	rows, err := h.db.Query("SELECT song_id FROM playlist_songs WHERE playlist_id = ? ORDER BY position", id)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, fmt.Sprintf("Failed to get playlist songs: %v", err))
		return
	}
	defer func() {
		_ = rows.Close()
	}()

	added := 0
	for rows.Next() {
		var songID int
		if err := rows.Scan(&songID); err != nil {
			continue
		}
		if err := h.queue.Add(songID); err == nil {
			added++
		}
	}

	w.Header().Set("Content-Type", "text/html")
	_, _ = fmt.Fprintf(w, messageSuccessTemplate, fmt.Sprintf("Added %d songs to queue!", added))
}

func (h *Handler) ReorderPlaylistHTMX(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, "Invalid form data")
		return
	}

	playlistID, err := strconv.Atoi(r.FormValue("playlist_id"))
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, "Invalid playlist ID")
		return
	}

	songID, err := strconv.Atoi(r.FormValue("song_id"))
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, "Invalid song ID")
		return
	}

	newPosition, err := strconv.Atoi(r.FormValue("new_position"))
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, "Invalid position")
		return
	}

	var oldPosition int
	err = h.db.QueryRow("SELECT position FROM playlist_songs WHERE playlist_id = ? AND song_id = ?", playlistID, songID).Scan(&oldPosition)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, "Song not in playlist")
		return
	}

	tx, err := h.db.Begin()
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, "Database error")
		return
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if oldPosition < newPosition {
		_, err = tx.Exec("UPDATE playlist_songs SET position = position - 1 WHERE playlist_id = ? AND position > ? AND position <= ?", playlistID, oldPosition, newPosition)
	} else {
		_, err = tx.Exec("UPDATE playlist_songs SET position = position + 1 WHERE playlist_id = ? AND position >= ? AND position < ?", playlistID, newPosition, oldPosition)
	}
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, "Failed to reorder")
		return
	}

	_, err = tx.Exec("UPDATE playlist_songs SET position = ? WHERE playlist_id = ? AND song_id = ?", newPosition, playlistID, songID)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, "Failed to reorder")
		return
	}

	if err = tx.Commit(); err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, "Failed to save reorder")
		return
	}

	w.Header().Set("Content-Type", "text/html")
	_, _ = w.Write([]byte(h.renderPlaylistSongsFragment(playlistID)))
}

// ==================== ALBUMS ====================

func (h *Handler) AlbumsPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	albumsHTML := h.renderAlbumsFragment()

	w.Header().Set("Content-Type", "text/html")
	result := albumsPageTemplate
	result = strings.Replace(result, "{{ALBUMS}}", albumsHTML, 1)
	fmt.Fprint(w, result)
}

func (h *Handler) AlbumsFragment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	_, _ = w.Write([]byte(h.renderAlbumsFragment()))
}

func (h *Handler) AlbumDetailPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	encodedName := r.URL.Path[len("/admin/library/album/"):]
	albumName, err := url.PathUnescape(encodedName)
	if err != nil {
		http.Error(w, "Invalid album name", http.StatusBadRequest)
		return
	}

	songs, err := h.crawler.GetAlbumSongs(albumName)
	if err != nil {
		http.Error(w, "Failed to get album songs", http.StatusInternalServerError)
		return
	}

	songsHTML := h.renderAlbumSongsFragment(songs, albumName)

	w.Header().Set("Content-Type", "text/html")
	result := albumDetailPageTemplate
	result = strings.Replace(result, "{{ALBUM_NAME}}", albumName, 1)
	result = strings.Replace(result, "{{SONGS}}", songsHTML, 1)
	fmt.Fprint(w, result)
}

func (h *Handler) AlbumSongsFragment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	encodedName := r.URL.Path[len("/admin/library/album/") : len(r.URL.Path)-len("/songs")]
	albumName, err := url.PathUnescape(encodedName)
	if err != nil {
		http.Error(w, "Invalid album name", http.StatusBadRequest)
		return
	}

	songs, err := h.crawler.GetAlbumSongs(albumName)
	if err != nil {
		http.Error(w, "Failed to get album songs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	_, _ = w.Write([]byte(h.renderAlbumSongsFragment(songs, albumName)))
}

// ==================== ARTISTS ====================

func (h *Handler) ArtistsPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	artistsHTML := h.renderArtistsFragment()

	w.Header().Set("Content-Type", "text/html")
	result := artistsPageTemplate
	result = strings.Replace(result, "{{ARTISTS}}", artistsHTML, 1)
	fmt.Fprint(w, result)
}

func (h *Handler) ArtistsFragment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	_, _ = w.Write([]byte(h.renderArtistsFragment()))
}

func (h *Handler) ArtistDetailPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	encodedName := r.URL.Path[len("/admin/library/artist/"):]
	artistName, err := url.PathUnescape(encodedName)
	if err != nil {
		http.Error(w, "Invalid artist name", http.StatusBadRequest)
		return
	}

	songs, err := h.crawler.GetArtistSongs(artistName)
	if err != nil {
		http.Error(w, "Failed to get artist songs", http.StatusInternalServerError)
		return
	}

	albums, err := h.crawler.GetArtistAlbums(artistName)
	if err != nil {
		albums = []string{}
	}

	songsHTML := h.renderArtistSongsFragment(songs, artistName)

	w.Header().Set("Content-Type", "text/html")
	result := artistDetailPageTemplate
	result = strings.Replace(result, "{{ARTIST_NAME}}", artistName, 1)
	result = strings.Replace(result, "{{SONGS}}", songsHTML, 1)

	albumLinks := ""
	for _, album := range albums {
		albumLinks += fmt.Sprintf(`<a href="/admin/library/album/%s" class="album-card">%s</a> `, url.PathEscape(album), album)
	}
	result = strings.Replace(result, "{{ALBUM_LINKS}}", albumLinks, 1)
	fmt.Fprint(w, result)
}

func (h *Handler) ArtistSongsFragment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	encodedName := r.URL.Path[len("/admin/library/artist/") : len(r.URL.Path)-len("/songs")]
	artistName, err := url.PathUnescape(encodedName)
	if err != nil {
		http.Error(w, "Invalid artist name", http.StatusBadRequest)
		return
	}

	songs, err := h.crawler.GetArtistSongs(artistName)
	if err != nil {
		http.Error(w, "Failed to get artist songs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	_, _ = w.Write([]byte(h.renderArtistSongsFragment(songs, artistName)))
}

// ==================== LIBRARY (with song list for adding to queue/playlist) ====================

func (h *Handler) LibraryPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	libraryHTML := h.renderLibraryFragment("", "", "")

	w.Header().Set("Content-Type", "text/html")
	result := libraryPageTemplate
	result = strings.Replace(result, "{{LIBRARY}}", libraryHTML, 1)
	fmt.Fprint(w, result)
}

func (h *Handler) LibrarySongsFragment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query().Get("q")
	genre := r.URL.Query().Get("genre")
	sort := r.URL.Query().Get("sort")

	w.Header().Set("Content-Type", "text/html")
	_, _ = w.Write([]byte(h.renderLibraryFragment(query, genre, sort)))
}

func (h *Handler) renderLibraryFragment(query, genre, sort string) string {
	sqlQuery := "SELECT id, title, artist, album, genre, track_number, track_total, cover_art, duration, location, status, is_favourite FROM songs WHERE status = 'library'"
	var args []interface{}

	if query != "" {
		likeQuery := "%" + query + "%"
		sqlQuery += " AND (title LIKE ? OR artist LIKE ? OR album LIKE ? OR genre LIKE ?)"
		args = append(args, likeQuery, likeQuery, likeQuery, likeQuery)
	}

	if genre != "" {
		sqlQuery += " AND genre = ?"
		args = append(args, genre)
	}

	switch sort {
	case "title":
		sqlQuery += " ORDER BY title"
	case "artist":
		sqlQuery += " ORDER BY artist, title"
	case "album":
		sqlQuery += " ORDER BY album, title"
	case "genre":
		sqlQuery += " ORDER BY genre, artist, title"
	case "duration":
		sqlQuery += " ORDER BY duration DESC"
	default:
		sqlQuery += " ORDER BY artist, title"
	}

	rows, err := h.db.Query(sqlQuery, args...)
	if err != nil {
		return emptySongsTemplate
	}
	defer func() {
		_ = rows.Close()
	}()

	playlists := h.getPlaylistOptions()

	var rowsHTML string
	count := 0
	for rows.Next() {
		var song SongResponse
		var fav int
		err := rows.Scan(&song.ID, &song.Title, &song.Artist, &song.Album, &song.Genre, &song.TrackNumber, &song.TrackTotal, &song.CoverArt, &song.Duration, &song.Location, &song.Status, &fav)
		if err != nil {
			continue
		}
		song.DurationFmt = formatDuration(song.Duration)
		song.IsFavourite = fav == 1
		favClass := ""
		if song.IsFavourite {
			favClass = "favourited"
		}
		playlistOptions := ""
		for _, p := range playlists {
			playlistOptions += fmt.Sprintf(`<option value="%d">%s</option>`, p.ID, p.Name)
		}
		rowsHTML += fmt.Sprintf(librarySongRowTemplate, favClass, song.ID, song.Title, song.Artist, song.Album, song.Genre, song.TrackNumber, song.DurationFmt, song.ID, song.ID, playlistOptions, song.ID, song.ID)
		count++
	}

	if count == 0 {
		return emptyLibraryTemplate
	}

	return fmt.Sprintf(libraryTableTemplate, rowsHTML)
}

func (h *Handler) AddToPlaylistFromLibraryHTMX(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, "Invalid form data")
		return
	}

	playlistID, err := strconv.Atoi(r.FormValue("playlist_id"))
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, "Select a playlist")
		return
	}

	songID, err := strconv.Atoi(r.FormValue("song_id"))
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, "Invalid song ID")
		return
	}

	var maxPos int
	_ = h.db.QueryRow("SELECT COALESCE(MAX(position), 0) FROM playlist_songs WHERE playlist_id = ?", playlistID).Scan(&maxPos)

	_, err = h.db.Exec("INSERT INTO playlist_songs (playlist_id, song_id, position) VALUES (?, ?, ?)", playlistID, songID, maxPos+1)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, fmt.Sprintf("Failed to add to playlist: %v", err))
		return
	}

	w.Header().Set("Content-Type", "text/html")
	_, _ = fmt.Fprintf(w, messageSuccessTemplate, "Song added to playlist!")
}

// ==================== DELETED ====================

func (h *Handler) DeletedPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	deletedHTML := h.renderDeletedFragment()

	w.Header().Set("Content-Type", "text/html")
	result := deletedPageTemplate
	result = strings.Replace(result, "{{DELETED}}", deletedHTML, 1)
	fmt.Fprint(w, result)
}

func (h *Handler) DeletedSongsFragment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	_, _ = w.Write([]byte(h.renderDeletedFragment()))
}

func (h *Handler) renderDeletedFragment() string {
	rows, err := h.db.Query("SELECT id, title, artist, album, genre, track_number, track_total, cover_art, duration, location, status, is_favourite FROM songs WHERE status = 'deleted' ORDER BY artist, title")
	if err != nil {
		return emptyDeletedTemplate
	}
	defer func() {
		_ = rows.Close()
	}()

	var rowsHTML string
	count := 0
	for rows.Next() {
		var song SongResponse
		var fav int
		err := rows.Scan(&song.ID, &song.Title, &song.Artist, &song.Album, &song.Genre, &song.TrackNumber, &song.TrackTotal, &song.CoverArt, &song.Duration, &song.Location, &song.Status, &fav)
		if err != nil {
			continue
		}
		song.DurationFmt = formatDuration(song.Duration)
		rowsHTML += fmt.Sprintf(deletedSongRowTemplate, song.Title, song.Artist, song.Album, song.Genre, song.DurationFmt, song.ID)
		count++
	}

	if count == 0 {
		return emptyDeletedTemplate
	}

	return fmt.Sprintf(deletedTableTemplate, rowsHTML)
}

func (h *Handler) AddToLibraryHTMX(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := r.URL.Path
	prefix := "/admin/library/"
	suffix := "/add"
	if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, suffix) {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	idStr := path[len(prefix) : len(path)-len(suffix)]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, "Invalid song ID")
		return
	}

	var title string
	err = h.db.QueryRow("SELECT title FROM songs WHERE id = ?", id).Scan(&title)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, "Song not found")
		return
	}

	_, err = h.db.Exec("UPDATE songs SET status = 'library', added_to_library_at = CURRENT_TIMESTAMP WHERE id = ?", id)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, fmt.Sprintf("Failed to add to library: %v", err))
		return
	}

	w.Header().Set("Content-Type", "text/html")
	_, _ = fmt.Fprintf(w, messageSuccessTemplate, fmt.Sprintf("'%s' added to library!", title))
}

func (h *Handler) RemoveFromLibraryHTMX(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := r.URL.Path
	prefix := "/admin/library/"
	suffix := "/remove"
	if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, suffix) {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	idStr := path[len(prefix) : len(path)-len(suffix)]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, "Invalid song ID")
		return
	}

	var title string
	err = h.db.QueryRow("SELECT title FROM songs WHERE id = ?", id).Scan(&title)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, "Song not found")
		return
	}

	_, err = h.db.Exec("UPDATE songs SET status = 'deleted' WHERE id = ?", id)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, fmt.Sprintf("Failed to remove from library: %v", err))
		return
	}

	w.Header().Set("Content-Type", "text/html")
	_, _ = fmt.Fprintf(w, messageSuccessTemplate, fmt.Sprintf("'%s' removed from library!", title))
}

// ==================== HISTORY ====================

func (h *Handler) HistoryPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	historyHTML := h.renderHistoryFragment()

	w.Header().Set("Content-Type", "text/html")
	result := historyPageTemplate
	result = strings.Replace(result, "{{HISTORY}}", historyHTML, 1)
	fmt.Fprint(w, result)
}

func (h *Handler) HistoryFragment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	_, _ = w.Write([]byte(h.renderHistoryFragment()))
}

func (h *Handler) renderHistoryFragment() string {
	rows, err := h.db.Query(`
		SELECT h.id, h.played_at, h.duration_played, s.id, s.title, s.artist, s.album, s.genre, s.duration
		FROM history h
		JOIN songs s ON h.song_id = s.id
		ORDER BY h.played_at DESC
		LIMIT 100
	`)
	if err != nil {
		return emptyHistoryTemplate
	}
	defer func() {
		_ = rows.Close()
	}()

	var rowsHTML string
	count := 0
	for rows.Next() {
		var id int
		var playedAt time.Time
		var durationPlayed int
		var songID int
		var title, artist, album, genre string
		var duration int

		err := rows.Scan(&id, &playedAt, &durationPlayed, &songID, &title, &artist, &album, &genre, &duration)
		if err != nil {
			continue
		}

		playedAtFormatted := playedAt.Format("2006-01-02 15:04")
		durationPlayedFmt := formatDuration(durationPlayed)
		durationFmt := formatDuration(duration)

		rowsHTML += fmt.Sprintf(historyRowTemplate, playedAtFormatted, title, artist, album, genre, durationFmt, durationPlayedFmt)
		count++
	}

	if count == 0 {
		return emptyHistoryTemplate
	}

	return fmt.Sprintf(historyTableTemplate, rowsHTML)
}

// ==================== EDIT METADATA ====================

func (h *Handler) EditSongPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := r.URL.Path
	prefix := "/admin/songs/"
	suffix := "/edit"
	if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, suffix) {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	idStr := path[len(prefix) : len(path)-len(suffix)]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid song ID", http.StatusBadRequest)
		return
	}

	var song SongResponse
	var fav int
	err = h.db.QueryRow("SELECT id, title, artist, album, genre, track_number, track_total, cover_art, duration, location, status, is_favourite FROM songs WHERE id = ?", id).
		Scan(&song.ID, &song.Title, &song.Artist, &song.Album, &song.Genre, &song.TrackNumber, &song.TrackTotal, &song.CoverArt, &song.Duration, &song.Location, &song.Status, &fav)
	if err != nil {
		http.Error(w, "Song not found", http.StatusNotFound)
		return
	}
	song.IsFavourite = fav == 1

	w.Header().Set("Content-Type", "text/html")
	result := editSongPageTemplate
	result = strings.Replace(result, "{{SONG_ID}}", idStr, 1)
	result = strings.Replace(result, "{{TITLE}}", song.Title, 1)
	result = strings.Replace(result, "{{ARTIST}}", song.Artist, 1)
	result = strings.Replace(result, "{{ALBUM}}", song.Album, 1)
	result = strings.Replace(result, "{{GENRE}}", song.Genre, 1)
	result = strings.Replace(result, "{{TRACK_NUMBER}}", strconv.Itoa(song.TrackNumber), 1)
	result = strings.Replace(result, "{{TRACK_TOTAL}}", strconv.Itoa(song.TrackTotal), 1)
	fmt.Fprint(w, result)
}

func (h *Handler) UpdateSongHTMX(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := r.URL.Path
	prefix := "/admin/songs/"
	suffix := "/edit"
	if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, suffix) {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	idStr := path[len(prefix) : len(path)-len(suffix)]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, "Invalid song ID")
		return
	}

	if err := r.ParseForm(); err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, "Invalid form data")
		return
	}

	title := r.FormValue("title")
	artist := r.FormValue("artist")
	album := r.FormValue("album")
	genre := r.FormValue("genre")
	trackNumber, _ := strconv.Atoi(r.FormValue("track_number"))
	trackTotal, _ := strconv.Atoi(r.FormValue("track_total"))

	_, err = h.db.Exec("UPDATE songs SET title = ?, artist = ?, album = ?, genre = ?, track_number = ?, track_total = ? WHERE id = ?",
		title, artist, album, genre, trackNumber, trackTotal, id)
	if err != nil {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprintf(w, messageErrorTemplate, fmt.Sprintf("Failed to update song: %v", err))
		return
	}

	w.Header().Set("Content-Type", "text/html")
	_, _ = fmt.Fprintf(w, messageSuccessTemplate, "Song metadata updated!")
}

// ==================== ANALYTICS ====================

func (h *Handler) AnalyticsPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, analyticsPageTemplate)
}

func (h *Handler) AnalyticsDataEndpoint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	period := r.URL.Query().Get("period")
	if period == "" {
		period = "hour"
	}

	type DataPoint struct {
		Time     string `json:"time"`
		Count    int    `json:"count"`
		SongName string `json:"song_name,omitempty"`
	}

	var data []DataPoint

	switch period {
	case "hour":
		rows, err := h.db.Query(`
			SELECT recorded_at, listener_count, current_song_id
			FROM listener_snapshots
			WHERE recorded_at >= datetime('now', '-1 hour')
			ORDER BY recorded_at ASC
		`)
		if err == nil {
			defer func() { _ = rows.Close() }()
			for rows.Next() {
				var dp DataPoint
				var recordedAt time.Time
				var songID int
				if err := rows.Scan(&recordedAt, &dp.Count, &songID); err == nil {
					dp.Time = recordedAt.Format("15:04")
					if songID > 0 {
						var name string
						_ = h.db.QueryRow("SELECT title || ' - ' || artist FROM songs WHERE id = ?", songID).Scan(&name)
						dp.SongName = name
					}
					data = append(data, dp)
				}
			}
		}
	case "day":
		rows, err := h.db.Query(`
			SELECT strftime('%H:%M', recorded_at) as time_bucket, AVG(listener_count), MAX(current_song_id)
			FROM listener_snapshots
			WHERE recorded_at >= datetime('now', '-1 day')
			GROUP BY strftime('%H', recorded_at), strftime('%M', recorded_at) / 15
			ORDER BY time_bucket ASC
		`)
		if err == nil {
			defer func() { _ = rows.Close() }()
			for rows.Next() {
				var dp DataPoint
				var songID int
				if err := rows.Scan(&dp.Time, &dp.Count, &songID); err == nil {
					if songID > 0 {
						var name string
						_ = h.db.QueryRow("SELECT title || ' - ' || artist FROM songs WHERE id = ?", songID).Scan(&name)
						dp.SongName = name
					}
					data = append(data, dp)
				}
			}
		}
	case "week":
		rows, err := h.db.Query(`
			SELECT strftime('%Y-%m-%d %H:00', recorded_at) as time_bucket, AVG(listener_count), MAX(current_song_id)
			FROM listener_snapshots
			WHERE recorded_at >= datetime('now', '-7 days')
			GROUP BY strftime('%Y-%m-%d %H', recorded_at)
			ORDER BY time_bucket ASC
		`)
		if err == nil {
			defer func() { _ = rows.Close() }()
			for rows.Next() {
				var dp DataPoint
				var songID int
				if err := rows.Scan(&dp.Time, &dp.Count, &songID); err == nil {
					if songID > 0 {
						var name string
						_ = h.db.QueryRow("SELECT title || ' - ' || artist FROM songs WHERE id = ?", songID).Scan(&name)
						dp.SongName = name
					}
					data = append(data, dp)
				}
			}
		}
	case "month":
		rows, err := h.db.Query(`
			SELECT strftime('%Y-%m-%d', recorded_at) as time_bucket, AVG(listener_count), MAX(current_song_id)
			FROM listener_snapshots
			WHERE recorded_at >= datetime('now', '-30 days')
			GROUP BY strftime('%Y-%m-%d', recorded_at)
			ORDER BY time_bucket ASC
		`)
		if err == nil {
			defer func() { _ = rows.Close() }()
			for rows.Next() {
				var dp DataPoint
				var songID int
				if err := rows.Scan(&dp.Time, &dp.Count, &songID); err == nil {
					if songID > 0 {
						var name string
						_ = h.db.QueryRow("SELECT title || ' - ' || artist FROM songs WHERE id = ?", songID).Scan(&name)
						dp.SongName = name
					}
					data = append(data, dp)
				}
			}
		}
	case "year":
		rows, err := h.db.Query(`
			SELECT strftime('%Y-%m', recorded_at) as time_bucket, AVG(listener_count), MAX(current_song_id)
			FROM listener_snapshots
			WHERE recorded_at >= datetime('now', '-1 year')
			GROUP BY strftime('%Y-%m', recorded_at)
			ORDER BY time_bucket ASC
		`)
		if err == nil {
			defer func() { _ = rows.Close() }()
			for rows.Next() {
				var dp DataPoint
				var songID int
				if err := rows.Scan(&dp.Time, &dp.Count, &songID); err == nil {
					if songID > 0 {
						var name string
						_ = h.db.QueryRow("SELECT title || ' - ' || artist FROM songs WHERE id = ?", songID).Scan(&name)
						dp.SongName = name
					}
					data = append(data, dp)
				}
			}
		}
	}

	if data == nil {
		data = []DataPoint{}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
	}
}

func (h *Handler) CurrentListenersHTMX(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	_, _ = w.Write([]byte(h.renderCurrentListenersFragment()))
}

func (h *Handler) renderCurrentListenersFragment() string {
	count := 0
	if h.broadcaster != nil {
		count = h.broadcaster.ClientCount()
	}
	return fmt.Sprintf(`<span class="listener-count">%d listeners online</span>`, count)
}

func (h *Handler) SaveListenerSnapshot(db *sql.DB, listenerCount int, currentSongID int) {
	_, _ = db.Exec("INSERT INTO listener_snapshots (listener_count, current_song_id) VALUES (?, ?)", listenerCount, currentSongID)
}

// ==================== RENDER HELPERS ====================

type playlistOption struct {
	ID   int
	Name string
}

func (h *Handler) getPlaylistOptions() []playlistOption {
	rows, err := h.db.Query("SELECT id, name FROM playlists ORDER BY name")
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()

	var opts []playlistOption
	for rows.Next() {
		var p playlistOption
		if err := rows.Scan(&p.ID, &p.Name); err == nil {
			opts = append(opts, p)
		}
	}
	return opts
}

func (h *Handler) renderSongsFragment() string {
	rows, err := h.db.Query("SELECT id, title, artist, album, genre, track_number, track_total, cover_art, duration, location, status, is_favourite FROM songs WHERE status = 'library' ORDER BY artist, title")
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
		var fav int
		err := rows.Scan(&song.ID, &song.Title, &song.Artist, &song.Album, &song.Genre, &song.TrackNumber, &song.TrackTotal, &song.CoverArt, &song.Duration, &song.Location, &song.Status, &fav)
		if err != nil {
			continue
		}
		song.DurationFmt = formatDuration(song.Duration)
		song.IsFavourite = fav == 1
		rowsHTML += fmt.Sprintf(songRowTemplate, song.Title, song.Artist, song.DurationFmt, song.ID, song.ID)
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
		rowsHTML += fmt.Sprintf(queueRowTemplate, item.ID, item.Position, item.Position, item.Song.TrackNumber, item.Song.Title, item.Song.Artist, item.Song.Album, item.Song.Duration, item.ID)
	}

	return fmt.Sprintf(queueTableTemplate, rowsHTML)
}

func (h *Handler) renderNowPlayingFragment() string {
	var songID int
	err := h.db.QueryRow("SELECT value FROM state WHERE key = 'current_song'").Scan(&songID)
	if err != nil {
		if h.playback != nil && h.playback.IsPaused() {
			return fmt.Sprintf(nowPlayingEmptyTemplate, "paused", "Paused", "play", "▶ Play")
		}
		return fmt.Sprintf(nowPlayingEmptyTemplate, "playing", "Waiting for queue", "stop", "⏸ Pause")
	}

	var song SongResponse
	var fav int
	err = h.db.QueryRow("SELECT id, title, artist, album, genre, track_number, track_total, cover_art, duration, location, status, is_favourite FROM songs WHERE id = ?", songID).
		Scan(&song.ID, &song.Title, &song.Artist, &song.Album, &song.Genre, &song.TrackNumber, &song.TrackTotal, &song.CoverArt, &song.Duration, &song.Location, &song.Status, &fav)
	if err != nil {
		if h.playback != nil && h.playback.IsPaused() {
			return fmt.Sprintf(nowPlayingEmptyTemplate, "paused", "Paused", "play", "▶ Play")
		}
		return fmt.Sprintf(nowPlayingEmptyTemplate, "playing", "Waiting for queue", "stop", "⏸ Pause")
	}
	song.IsFavourite = fav == 1

	coverHTML := ""
	if song.CoverArt != "" {
		coverHTML = fmt.Sprintf(`<img src="/admin/cover/%d" alt="Cover" style="width:100px;height:100px;border-radius:5px;margin-bottom:10px;">`, song.ID)
	}

	detailsText := ""
	if song.Album != "" {
		detailsText += fmt.Sprintf("Album: %s | ", song.Album)
	}
	if song.Genre != "" {
		detailsText += fmt.Sprintf("Genre: %s | ", song.Genre)
	}
	if song.TrackNumber > 0 {
		detailsText += fmt.Sprintf("Track: %d", song.TrackNumber)
		if song.TrackTotal > 0 {
			detailsText += fmt.Sprintf("/%d", song.TrackTotal)
		}
	}
	detailsText = strings.TrimRight(detailsText, " | ")

	durationFmt := formatDuration(song.Duration)

	if h.playback != nil && h.playback.IsPaused() {
		return fmt.Sprintf(nowPlayingTemplate, coverHTML, song.Title, song.Artist, detailsText, durationFmt, "paused", "Paused", "play", "▶ Play")
	}
	return fmt.Sprintf(nowPlayingTemplate, coverHTML, song.Title, song.Artist, detailsText, durationFmt, "playing", "Playing", "stop", "⏸ Pause")
}

func (h *Handler) renderPlaylistsFragment() string {
	rows, err := h.db.Query("SELECT id, name, created_at FROM playlists ORDER BY name")
	if err != nil {
		return emptyPlaylistsTemplate
	}
	defer func() { _ = rows.Close() }()

	type playlistRow struct {
		ID        int
		Name      string
		CreatedAt string
		SongCount int
	}

	var playlists []playlistRow
	for rows.Next() {
		var p playlistRow
		if err := rows.Scan(&p.ID, &p.Name, &p.CreatedAt); err == nil {
			_ = h.db.QueryRow("SELECT COUNT(*) FROM playlist_songs WHERE playlist_id = ?", p.ID).Scan(&p.SongCount)
			playlists = append(playlists, p)
		}
	}

	if len(playlists) == 0 {
		return emptyPlaylistsTemplate
	}

	var rowsHTML string
	for _, p := range playlists {
		rowsHTML += fmt.Sprintf(playlistRowTemplate, p.ID, p.Name, p.SongCount, p.ID)
	}

	return fmt.Sprintf(playlistsTableTemplate, rowsHTML)
}

func (h *Handler) renderPlaylistSongsFragment(playlistID int) string {
	rows, err := h.db.Query(`
		SELECT ps.position, s.id, s.title, s.artist, s.album, s.genre, s.track_number, s.duration
		FROM playlist_songs ps
		JOIN songs s ON ps.song_id = s.id
		WHERE ps.playlist_id = ?
		ORDER BY ps.position
	`, playlistID)
	if err != nil {
		return emptySongsTemplate
	}
	defer func() { _ = rows.Close() }()

	var rowsHTML string
	count := 0
	for rows.Next() {
		var position, songID, trackNumber, duration int
		var title, artist, album, genre string
		if err := rows.Scan(&position, &songID, &title, &artist, &album, &genre, &trackNumber, &duration); err != nil {
			continue
		}
		durationFmt := formatDuration(duration)
		rowsHTML += fmt.Sprintf(playlistSongRowTemplate, songID, playlistID, trackNumber, title, artist, album, genre, durationFmt, songID, playlistID, songID)
		count++
	}

	if count == 0 {
		return emptySongsTemplate
	}

	return fmt.Sprintf(playlistSongsTableTemplate, rowsHTML)
}

func (h *Handler) renderAlbumsFragment() string {
	albums, err := h.crawler.GetAllAlbums()
	if err != nil {
		return emptyAlbumsTemplate
	}

	if len(albums) == 0 {
		return emptyAlbumsTemplate
	}

	var gridHTML string
	for _, album := range albums {
		encodedName := url.PathEscape(album)
		gridHTML += fmt.Sprintf(`<a href="/admin/library/album/%s" class="album-card"><div class="album-name">%s</div></a>`, encodedName, album)
	}

	return fmt.Sprintf(`<div class="grid">%s</div>`, gridHTML)
}

func (h *Handler) renderAlbumSongsFragment(songs []crawler.SongInfo, albumName string) string {
	if len(songs) == 0 {
		return emptySongsTemplate
	}

	var rowsHTML string
	for _, s := range songs {
		rowsHTML += fmt.Sprintf(albumSongRowTemplate, s.TrackNumber, s.Title, s.Artist, s.Genre, s.DurationFormatted, s.ID)
	}

	return fmt.Sprintf(albumSongsTableTemplate, rowsHTML)
}

func (h *Handler) renderArtistsFragment() string {
	artists, err := h.crawler.GetAllArtists()
	if err != nil {
		return emptyArtistsTemplate
	}

	if len(artists) == 0 {
		return emptyArtistsTemplate
	}

	var gridHTML string
	for _, artist := range artists {
		encodedName := url.PathEscape(artist)
		gridHTML += fmt.Sprintf(`<a href="/admin/library/artist/%s" class="artist-card"><div class="artist-name">%s</div></a>`, encodedName, artist)
	}

	return fmt.Sprintf(`<div class="grid">%s</div>`, gridHTML)
}

func (h *Handler) renderArtistSongsFragment(songs []crawler.SongInfo, artistName string) string {
	if len(songs) == 0 {
		return emptySongsTemplate
	}

	var rowsHTML string
	for _, s := range songs {
		rowsHTML += fmt.Sprintf(artistSongRowTemplate, s.Album, s.TrackNumber, s.Title, s.Genre, s.DurationFormatted, s.ID)
	}

	return fmt.Sprintf(artistSongsTableTemplate, rowsHTML)
}
