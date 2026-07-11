package stream

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
)

type ListenHandler struct {
	db *sql.DB
}

func NewListenHandler(db *sql.DB) *ListenHandler {
	return &ListenHandler{db: db}
}

func (h *ListenHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(listenPageTemplate))
}

func (h *ListenHandler) NowPlayingFragment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	_, _ = w.Write([]byte(h.renderNowPlaying()))
}

func (h *ListenHandler) CoverHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.URL.Path[len("/cover/"):]
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

func (h *ListenHandler) renderNowPlaying() string {
	var songID int
	err := h.db.QueryRow("SELECT value FROM state WHERE key = 'current_song'").Scan(&songID)
	if err != nil {
		return fmt.Sprintf(nowPlayingEmptyListenTemplate, "No song currently playing", "Waiting for queue")
	}

	var title, artist, album, coverArt string
	var duration int
	err = h.db.QueryRow("SELECT title, artist, album, cover_art, duration FROM songs WHERE id = ?", songID).
		Scan(&title, &artist, &album, &coverArt, &duration)
	if err != nil {
		return fmt.Sprintf(nowPlayingEmptyListenTemplate, "No song currently playing", "Waiting for queue")
	}

	coverHTML := `<div class="cover-placeholder">🎵</div>`
	if coverArt != "" {
		coverHTML = fmt.Sprintf(`<img src="/cover/%d" alt="%s" class="cover-art">`, songID, title)
	}

	albumLine := ""
	if album != "" {
		albumLine = fmt.Sprintf(`<div class="album">%s</div>`, album)
	}

	durationStr := formatDuration(duration)

	return fmt.Sprintf(nowPlayingListenTemplate, coverHTML, title, artist, albumLine, durationStr)
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

const listenPageTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Hackday Radio</title>
    <script src="https://unpkg.com/htmx.org@1.9.10"></script>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
            background: linear-gradient(135deg, #1a1a2e 0%%, #16213e 50%%, #0f3460 100%%);
            color: #fff;
            min-height: 100vh;
            display: flex;
            flex-direction: column;
            align-items: center;
            justify-content: center;
            padding: 20px;
        }
        .player {
            background: rgba(255, 255, 255, 0.1);
            backdrop-filter: blur(10px);
            border-radius: 20px;
            padding: 40px;
            max-width: 400px;
            width: 100%%;
            text-align: center;
            box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
        }
        .cover-art {
            width: 250px;
            height: 250px;
            border-radius: 15px;
            object-fit: cover;
            margin-bottom: 25px;
            box-shadow: 0 4px 20px rgba(0, 0, 0, 0.4);
        }
        .cover-placeholder {
            width: 250px;
            height: 250px;
            border-radius: 15px;
            background: rgba(255, 255, 255, 0.05);
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 80px;
            margin: 0 auto 25px;
            border: 2px dashed rgba(255, 255, 255, 0.2);
        }
        .song-info {
            margin-bottom: 25px;
        }
        .title {
            font-size: 24px;
            font-weight: 700;
            margin-bottom: 8px;
            line-height: 1.3;
        }
        .artist {
            font-size: 18px;
            color: rgba(255, 255, 255, 0.7);
            margin-bottom: 5px;
        }
        .album {
            font-size: 14px;
            color: rgba(255, 255, 255, 0.5);
            font-style: italic;
        }
        .duration {
            font-size: 13px;
            color: rgba(255, 255, 255, 0.4);
            margin-top: 8px;
        }
        .status-badge {
            display: inline-block;
            padding: 4px 12px;
            border-radius: 20px;
            font-size: 12px;
            font-weight: 600;
            text-transform: uppercase;
            letter-spacing: 1px;
            margin-bottom: 20px;
        }
        .status-badge.live {
            background: #e91e63;
            animation: pulse 2s infinite;
        }
        @keyframes pulse {
            0%%, 100%% { opacity: 1; }
            50%% { opacity: 0.7; }
        }
        .audio-player {
            width: 100%%;
            margin-top: 20px;
        }
        audio {
            width: 100%%;
            border-radius: 10px;
        }
        .controls {
            margin-top: 20px;
        }
        .play-btn {
            background: #e91e63;
            color: white;
            border: none;
            padding: 15px 40px;
            border-radius: 30px;
            font-size: 16px;
            font-weight: 600;
            cursor: pointer;
            transition: all 0.3s ease;
        }
        .play-btn:hover {
            background: #c2185b;
            transform: scale(1.05);
        }
        .empty-state {
            color: rgba(255, 255, 255, 0.5);
            font-style: italic;
        }
        .admin-link {
            margin-top: 30px;
            font-size: 14px;
        }
        .admin-link a {
            color: rgba(255, 255, 255, 0.5);
            text-decoration: none;
            transition: color 0.3s;
        }
        .admin-link a:hover {
            color: rgba(255, 255, 255, 0.8);
        }
    </style>
</head>
<body>
    <div class="player">
        <div class="status-badge live">● LIVE</div>

        <div id="now-playing" hx-get="/listen/now-playing" hx-trigger="load, every 5s" hx-swap="innerHTML">
            <div class="cover-placeholder">🎵</div>
            <div class="song-info">
                <div class="empty-state">Loading...</div>
            </div>
        </div>

        <div class="audio-player">
            <audio id="player" controls autoplay>
                <source src="/stream" type="audio/mpeg">
                Your browser does not support the audio element.
            </audio>
        </div>

        <div class="admin-link">
            <a href="/admin">Admin Panel</a>
        </div>
    </div>
</body>
</html>`

const nowPlayingListenTemplate = `<div class="cover-art-container">
    %s
</div>
<div class="song-info">
    <div class="title">%s</div>
    <div class="artist">%s</div>
    %s
    <div class="duration">⏱ %s</div>
</div>`

const nowPlayingEmptyListenTemplate = `<div class="cover-placeholder">🎵</div>
<div class="song-info">
    <div class="title">%s</div>
    <div class="artist empty-state">%s</div>
</div>`
