package admin

import "net/http"

func RegisterRoutes(h *Handler, mux *http.ServeMux) {
	// ========== HTMX Admin Routes ==========
	// Full admin page
	mux.HandleFunc("/admin", h.AdminPage)

	// Fragments for HTMX
	mux.HandleFunc("/admin/songs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.SongsFragment(w, r)
		} else {
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
	mux.HandleFunc("/admin/queue/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			h.AddToQueueHTMX(w, r)
		case http.MethodDelete:
			h.RemoveFromQueueHTMX(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/admin/songs/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			h.DeleteSongHTMX(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/admin/upload", h.UploadSongHTMX)
	mux.HandleFunc("/admin/rescan", h.RescanHTMX)
	mux.HandleFunc("/admin/play", h.PlayControl)
	mux.HandleFunc("/admin/now-playing", h.NowPlayingFragment)
	mux.HandleFunc("/admin/cover/", h.CoverArtHandler)

	// ========== Music Library Routes ==========
	mux.HandleFunc("/admin/library", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.LibraryPage(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/admin/library/songs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.LibrarySongsFragment(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/admin/library/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			path := r.URL.Path
			if len(path) > len("/admin/library/") {
				rest := path[len("/admin/library/"):]
				if len(rest) > 4 && rest[len(rest)-4:] == "/add" {
					h.AddToLibraryHTMX(w, r)
					return
				}
				if len(rest) > 7 && rest[len(rest)-7:] == "/remove" {
					h.RemoveFromLibraryHTMX(w, r)
					return
				}
			}
			http.Error(w, "Invalid path", http.StatusBadRequest)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// ========== Deleted Songs Routes ==========
	mux.HandleFunc("/admin/deleted", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.DeletedPage(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/admin/deleted/songs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.DeletedSongsFragment(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// ========== History Routes ==========
	mux.HandleFunc("/admin/history", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.HistoryPage(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/admin/history/items", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.HistoryFragment(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// ========== Legacy API Routes (JSON) ==========
	// Songs
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

	// Upload
	mux.HandleFunc("/api/upload", h.UploadSong)

	// Queue
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

	// Now playing
	mux.HandleFunc("/api/now-playing", h.NowPlaying)

	// History
	mux.HandleFunc("/api/history", h.GetHistory)
}
