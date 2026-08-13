package admin

import (
	"net/http"
	"strings"
)

func RegisterRoutes(h *Handler, mux *http.ServeMux) {
	// ========== HTMX Admin Routes ==========
	mux.HandleFunc("/admin", h.AdminPage)

	mux.HandleFunc("/admin/now-playing", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.NowPlayingFragment(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/admin/play", h.PlayControl)

	mux.HandleFunc("/admin/rescan", h.RescanHTMX)

	mux.HandleFunc("/admin/upload", h.UploadSongHTMX)

	mux.HandleFunc("/admin/listeners", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.CurrentListenersHTMX(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/admin/cover/", h.CoverArtHandler)

	// ========== Queue Routes ==========
	mux.HandleFunc("/admin/queue/reorder", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			h.ReorderQueueHTMX(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/admin/queue/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			h.QueueFragment(w, r)
		case http.MethodPost:
			h.AddToQueueHTMX(w, r)
		case http.MethodDelete:
			h.RemoveFromQueueHTMX(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/admin/queue", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.QueueFragment(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// ========== Songs Routes (legacy + new) ==========
	mux.HandleFunc("/admin/songs/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		if strings.HasSuffix(path, "/favourite") {
			if r.Method == http.MethodPost {
				h.ToggleFavouriteHTMX(w, r)
				return
			}
		}

		if strings.HasSuffix(path, "/edit") {
			switch r.Method {
			case http.MethodGet:
				h.EditSongPage(w, r)
			case http.MethodPost:
				h.UpdateSongHTMX(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
			return
		}

		if r.Method == http.MethodDelete {
			h.DeleteSongHTMX(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/admin/songs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.SongsFragment(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// ========== Add to Playlist from Library ==========
	mux.HandleFunc("/admin/add-to-playlist", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			h.AddToPlaylistFromLibraryHTMX(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// ========== Library Routes ==========
	mux.HandleFunc("/admin/library/songs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.LibrarySongsFragment(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/admin/library/album/list", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.AlbumsFragment(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/admin/library/album/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			path := r.URL.Path
			suffix := "/songs"
			if strings.HasSuffix(path, suffix) {
				h.AlbumSongsFragment(w, r)
			} else {
				h.AlbumDetailPage(w, r)
			}
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/admin/library/album", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.AlbumsPage(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/admin/library/artist/list", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.ArtistsFragment(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/admin/library/artist/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			path := r.URL.Path
			suffix := "/songs"
			if strings.HasSuffix(path, suffix) {
				h.ArtistSongsFragment(w, r)
			} else {
				h.ArtistDetailPage(w, r)
			}
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/admin/library/artist", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.ArtistsPage(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/admin/library/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			path := r.URL.Path
			if strings.HasSuffix(path, "/add") {
				h.AddToLibraryHTMX(w, r)
				return
			}
			if strings.HasSuffix(path, "/remove") {
				h.RemoveFromLibraryHTMX(w, r)
				return
			}
			http.Error(w, "Invalid path", http.StatusBadRequest)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/admin/library", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.LibraryPage(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// ========== Deleted Routes ==========
	mux.HandleFunc("/admin/deleted/songs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.DeletedSongsFragment(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/admin/deleted", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.DeletedPage(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// ========== History Routes ==========
	mux.HandleFunc("/admin/history/items", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.HistoryFragment(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/admin/history", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.HistoryPage(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// ========== Analytics Routes ==========
	mux.HandleFunc("/admin/analytics/data", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.AnalyticsDataEndpoint(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/admin/analytics", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.AnalyticsPage(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// ========== Playlist Routes ==========
	mux.HandleFunc("/admin/playlists/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		if strings.HasSuffix(path, "/reorder") {
			if r.Method == http.MethodPost {
				h.ReorderPlaylistHTMX(w, r)
				return
			}
		}

		if strings.HasSuffix(path, "/queue") {
			if r.Method == http.MethodPost {
				h.QueuePlaylistHTMX(w, r)
				return
			}
		}

		if strings.HasSuffix(path, "/songs") {
			if r.Method == http.MethodGet {
				h.PlaylistSongsFragment(w, r)
				return
			}
		}

		if strings.Contains(path, "/add/") {
			if r.Method == http.MethodPost {
				h.AddToPlaylistHTMX(w, r)
				return
			}
		}

		if strings.Contains(path, "/remove/") {
			if r.Method == http.MethodPost {
				h.RemoveFromPlaylistHTMX(w, r)
				return
			}
		}

		remaining := path[len("/admin/playlists/"):]
		if remaining != "" {
			switch r.Method {
			case http.MethodGet:
				h.PlaylistDetailPage(w, r)
			case http.MethodDelete:
				h.DeletePlaylistHTMX(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
			return
		}

		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	})

	mux.HandleFunc("/admin/playlists/create", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			h.CreatePlaylistHTMX(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/admin/playlists/list", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.PlaylistsFragment(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/admin/playlists", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.PlaylistsPage(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// ========== Legacy API Routes (JSON) ==========
	mux.HandleFunc("/api/songs", h.ListSongs)
	mux.HandleFunc("/api/songs/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			h.AddSong(w, r)
		case http.MethodDelete:
			h.DeleteSong(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/upload", h.UploadSong)

	mux.HandleFunc("/api/queue", h.GetQueue)
	mux.HandleFunc("/api/queue/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			h.AddToQueue(w, r)
		case http.MethodDelete:
			h.RemoveFromQueue(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/now-playing", h.NowPlaying)
	mux.HandleFunc("/api/history", h.GetHistory)
}
