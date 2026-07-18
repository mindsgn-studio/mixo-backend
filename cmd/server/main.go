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
	"github.com/mindsgn-studio/mixo-backend/internal/playback"
	"github.com/mindsgn-studio/mixo-backend/internal/queue"
	"github.com/mindsgn-studio/mixo-backend/internal/stream"
	"github.com/mindsgn-studio/mixo-backend/internal/worker"
)

const version = "0.3.0"

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

	// Initialize admin handler
	adminHandler := admin.New(db.DB, queueManager, cfg)
	adminHandler.SetPlayback(playbackEngine)
	adminHandler.SetCrawler(crawlerWorker)

	// Setup HTTP server
	mux := http.NewServeMux()

	// CORS middleware
	corsMux := http.NewServeMux()
	corsMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		mux.ServeHTTP(w, r)
	})

	// Stream endpoint
	streamHandler := stream.NewHandler(broadcaster)
	mux.Handle("/stream", streamHandler)

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
