package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Port          string
	SongDir       string
	DBPath        string
	StreamTimeout int
	CrawlDirs     string
	QueueHours    int
	CrawlInterval int
}

func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		return nil, fmt.Errorf("failed to load .env file: %w", err)
	}

	port := getEnv("PORT", "8080")
	songDir := getEnv("SONG_DIR", "./songs")
	dbPath := getEnv("DB_PATH", "./radio.db")
	streamTimeout := getEnvAsInt("STREAM_TIMEOUT", "5")
	crawlDirs := getEnv("CRAWL_DIRS", "")
	queueHours := getEnvAsInt("QUEUE_HOURS", "24")
	crawlInterval := getEnvAsInt("CRAWL_INTERVAL", "60")

	return &Config{
		Port:          port,
		SongDir:       songDir,
		DBPath:        dbPath,
		StreamTimeout: streamTimeout,
		CrawlDirs:     crawlDirs,
		QueueHours:    queueHours,
		CrawlInterval: crawlInterval,
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key, defaultValue string) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	intVal, _ := strconv.Atoi(defaultValue)
	return intVal
}
