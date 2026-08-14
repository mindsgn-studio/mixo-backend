package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/mindsgn-studio/mixo-backend/internal/admin"
	"github.com/mindsgn-studio/mixo-backend/internal/config"
	"github.com/mindsgn-studio/mixo-backend/internal/crawler"
	"github.com/mindsgn-studio/mixo-backend/internal/database"
	"github.com/mindsgn-studio/mixo-backend/internal/hls"
	"github.com/mindsgn-studio/mixo-backend/internal/playback"
	"github.com/mindsgn-studio/mixo-backend/internal/queue"
	"github.com/mindsgn-studio/mixo-backend/internal/stream"
	"github.com/mindsgn-studio/mixo-backend/internal/worker"
)

const version = "0.5.0"

func main() {
	log.Printf("Starting Radio Server v%s", version)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize database
	db, err := database.New(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("Warning: failed to close database: %v", err)
		}
	}()

	// Initialize queue manager
	queueManager := queue.New(db.DB)

	// Initialize crawler
	crawlerWorker := crawler.New(db.DB, cfg.SongDir)
	crawlDirs := strings.Split(cfg.CrawlDirs, ",")

	// Initialize queue worker (handles initial crawl + periodic rescan)
	queueWorker := worker.New(db.DB, queueManager, crawlerWorker, crawlDirs, cfg.QueueHours, cfg.CrawlInterval)
	queueWorker.Start()
	defer queueWorker.Stop()

	// Initialize playback engine
	playbackEngine := playback.New(db.DB, queueManager)
	playbackEngine.Start()
	defer playbackEngine.Stop()

	// Initialize stream broadcaster
	streamTimeout := time.Duration(cfg.StreamTimeout) * time.Second
	broadcaster := stream.New(playbackEngine.GetChunkChan(), streamTimeout)
	broadcaster.Start()

	// Initialize HLS encoder for the radio stream. It consumes the same audio
	// chunks as the HTTP broadcaster and writes HLS files to the shared web
	// root (default /var/www/html/hls), next to the video streaming server's
	// output.
	enc := hls.New(hls.Config{
		Bin:          cfg.FFmpegBin,
		HLSDir:       cfg.HLSDir,
		StreamID:     cfg.HLSStreamID,
		SegmentTime:  cfg.HLSSegmentTime,
		PlaylistSize: cfg.HLSPlaylistSize,
		Stderr:       os.Stderr,
	})
	if err := enc.Start(); err != nil {
		log.Printf("Warning: HLS encoder disabled: %v", err)
	} else {
		sink := make(chan []byte, 128)
		removeSink := broadcaster.AddSink(sink)
		defer removeSink()
		go func() {
			for chunk := range sink {
				if _, err := enc.Write(chunk); err != nil {
					log.Printf("Warning: HLS encoder write failed: %v", err)
					return
				}
			}
		}()
		log.Printf("HLS encoder enabled: %s/%s/index.m3u8", cfg.HLSDir, cfg.HLSStreamID)
	}
	defer enc.Stop()

	// Initialize admin handler
	adminHandler := admin.New(db.DB, queueManager, cfg)
	adminHandler.SetPlayback(playbackEngine)
	adminHandler.SetCrawler(crawlerWorker)
	adminHandler.SetBroadcaster(broadcaster)

	// Start listener snapshot recorder
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			var currentSongID int
			_ = db.DB.QueryRow("SELECT value FROM state WHERE key = 'current_song'").Scan(&currentSongID)
			adminHandler.SaveListenerSnapshot(db.DB, broadcaster.ClientCount(), currentSongID)
		}
	}()

	// Setup HTTP server
	mux := http.NewServeMux()

	// Protect the admin interface with HTTP Basic Auth (enabled when
	// ADMIN_PASSWORD is set in .env).
	protected := admin.BasicAuth(cfg)(mux)

	// CORS middleware (applied before admin auth so preflight OPTIONS requests
	// never need credentials).
	corsMux := http.NewServeMux()
	corsMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		protected.ServeHTTP(w, r)
	})

	// Stream endpoint
	streamHandler := stream.NewHandler(broadcaster)
	mux.Handle("/stream", streamHandler)

	// HLS files for the radio stream, served from the shared web root.
	mux.Handle("/hls/", hlsCacheControl(http.StripPrefix("/hls/", http.FileServer(http.Dir(cfg.HLSDir)))))

	// Listen page (public-facing player)
	listenHandler := stream.NewListenHandler(db.DB)
	mux.Handle("/listen", listenHandler)
	mux.HandleFunc("/listen/now-playing", listenHandler.NowPlayingFragment)
	mux.HandleFunc("/listen/events", listenHandler.NowPlayingEvents)
	mux.HandleFunc("/cover/", listenHandler.CoverHandler)

	// Admin API endpoints
	admin.RegisterRoutes(adminHandler, mux)

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// Create HTTP server
	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: corsMux,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Server listening on port %s", cfg.Port)
		log.Printf("Stream endpoint: http://localhost:%s/stream", cfg.Port)
		log.Printf("Admin page: http://localhost:%s/admin", cfg.Port)
		log.Printf("Admin API: http://localhost:%s/api", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown error: %v", err)
	}

	log.Println("Server stopped")
}

// hlsCacheControl sends cache-friendly headers for HLS media: playlists are
// never cached, segments are cached briefly, and every response allows CORS.
func hlsCacheControl(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".m3u8") {
			w.Header().Set("Cache-Control", "no-cache")
		} else {
			w.Header().Set("Cache-Control", "max-age=60")
		}
		w.Header().Set("Access-Control-Allow-Origin", "*")
		next.ServeHTTP(w, r)
	})
}
