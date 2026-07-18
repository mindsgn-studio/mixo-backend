package crawler

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dhowden/tag"
	"github.com/mindsgn-studio/mixo-backend/internal/playback"
)

type Crawler struct {
	db      *sql.DB
	songDir string
}

func New(db *sql.DB, songDir string) *Crawler {
	return &Crawler{db: db, songDir: songDir}
}

func (c *Crawler) ScanDirectories(dirs []string) error {
	for _, dir := range dirs {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		if err := c.scanDirectory(dir); err != nil {
			log.Printf("Error scanning directory %s: %v", dir, err)
		}
	}

	if err := c.removeMissingFiles(); err != nil {
		log.Printf("Error removing missing files: %v", err)
	}

	return nil
}

func (c *Crawler) scanDirectory(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() && path != dir && strings.HasPrefix(info.Name(), ".") {
			return filepath.SkipDir
		}
		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".mp3" {
			return nil
		}

		return c.processFile(path)
	})
}

func (c *Crawler) processFile(filePath string) error {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	var exists bool
	err = c.db.QueryRow("SELECT EXISTS(SELECT 1 FROM songs WHERE location = ? OR source_location = ?)", absPath, absPath).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check song existence: %w", err)
	}
	if exists {
		return nil
	}

	file, err := os.Open(absPath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("Warning: failed to close file: %v", err)
		}
	}()

	metadata, err := tag.ReadFrom(file)
	if err != nil {
		log.Printf("Warning: failed to read metadata from %s: %v", absPath, err)
	}

	title := ""
	artist := ""
	album := ""
	coverArt := ""

	if metadata != nil {
		title = metadata.Title()
		artist = metadata.Artist()
		album = metadata.Album()

		// Extract cover art
		picture := metadata.Picture()
		if picture != nil {
			coverPath := filepath.Join(c.songDir, ".covers", fmt.Sprintf("%d.jpg", time.Now().UnixNano()))
			if err := os.MkdirAll(filepath.Dir(coverPath), 0755); err == nil {
				if err := os.WriteFile(coverPath, picture.Data, 0644); err == nil {
					coverArt = coverPath
				}
			}
		}
	}

	if title == "" {
		title = filepath.Base(absPath)
		title = strings.TrimSuffix(title, filepath.Ext(title))
	}
	if artist == "" {
		artist = "Unknown Artist"
	}

	duration, err := c.getDuration(absPath)
	if err != nil {
		log.Printf("Warning: failed to get duration for %s: %v", absPath, err)
		duration = 0
	}

	stationPath, err := playback.NormalizeToStationMP3(absPath, playback.StationCacheDir(c.songDir))
	if err != nil {
		return fmt.Errorf("failed to normalize audio: %w", err)
	}

	_, err = c.db.Exec(
		"INSERT INTO songs (title, artist, album, cover_art, duration, location, source_location, normalized) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		title, artist, album, coverArt, duration, stationPath, absPath, 1,
	)
	if err != nil {
		return fmt.Errorf("failed to insert song: %w", err)
	}

	log.Printf("Added song: %s - %s (%ds) from %s", artist, title, duration, absPath)
	return nil
}

func (c *Crawler) getDuration(filePath string) (int, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", filePath)
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

func (c *Crawler) removeMissingFiles() error {
	rows, err := c.db.Query("SELECT id, location, source_location FROM songs")
	if err != nil {
		return fmt.Errorf("failed to query songs: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Warning: failed to close rows: %v", err)
		}
	}()

	var idsToRemove []int
	for rows.Next() {
		var id int
		var location string
		var sourceLocation string
		if err := rows.Scan(&id, &location, &sourceLocation); err != nil {
			continue
		}

		checkPath := location
		if sourceLocation != "" {
			checkPath = sourceLocation
		}
		if _, err := os.Stat(checkPath); os.IsNotExist(err) {
			idsToRemove = append(idsToRemove, id)
			log.Printf("File missing, marking for removal: %s", checkPath)
		}
	}

	for _, id := range idsToRemove {
		_, err := c.db.Exec("DELETE FROM songs WHERE id = ?", id)
		if err != nil {
			log.Printf("Warning: failed to remove song %d: %v", id, err)
		}
	}

	if len(idsToRemove) > 0 {
		log.Printf("Removed %d songs with missing files", len(idsToRemove))
	}

	return nil
}

func (c *Crawler) GetTotalDuration(dirs []string) (int, error) {
	var totalDuration int

	rows, err := c.db.Query("SELECT COALESCE(SUM(duration), 0) FROM songs")
	if err != nil {
		return 0, fmt.Errorf("failed to query total duration: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Warning: failed to close rows: %v", err)
		}
	}()

	if rows.Next() {
		if err := rows.Scan(&totalDuration); err != nil {
			return 0, fmt.Errorf("failed to scan total duration: %w", err)
		}
	}

	return totalDuration, nil
}

func (c *Crawler) GetRandomSongs(count int) ([]int, error) {
	rows, err := c.db.Query("SELECT id FROM songs ORDER BY RANDOM() LIMIT ?", count)
	if err != nil {
		return nil, fmt.Errorf("failed to query random songs: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Warning: failed to close rows: %v", err)
		}
	}()

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			continue
		}
		ids = append(ids, id)
	}

	return ids, nil
}

func (c *Crawler) GetSongCount() (int, error) {
	var count int
	err := c.db.QueryRow("SELECT COUNT(*) FROM songs").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get song count: %w", err)
	}
	return count, nil
}

func (c *Crawler) GetTotalDurationFormatted() (string, error) {
	totalSeconds, err := c.GetTotalDuration(nil)
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

func (c *Crawler) GetAllSongs() ([]SongInfo, error) {
	rows, err := c.db.Query("SELECT id, title, artist, album, duration, location FROM songs ORDER BY artist, title")
	if err != nil {
		return nil, fmt.Errorf("failed to query songs: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Warning: failed to close rows: %v", err)
		}
	}()

	var songs []SongInfo
	for rows.Next() {
		var s SongInfo
		if err := rows.Scan(&s.ID, &s.Title, &s.Artist, &s.Album, &s.Duration, &s.Location); err != nil {
			continue
		}
		s.DurationFormatted = formatDuration(s.Duration)
		songs = append(songs, s)
	}

	return songs, nil
}

type SongInfo struct {
	ID                int
	Title             string
	Artist            string
	Album             string
	Duration          int
	DurationFormatted string
	Location          string
}

func formatDuration(seconds int) string {
	h := seconds / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60

	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

func (c *Crawler) WriteToFile(w io.Writer, data []byte) error {
	_, err := w.Write(data)
	return err
}
