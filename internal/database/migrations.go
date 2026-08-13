package database

import (
	"database/sql"
	"fmt"
)

const schema = `
CREATE TABLE IF NOT EXISTS songs (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	title TEXT NOT NULL,
	artist TEXT NOT NULL,
	album TEXT DEFAULT '',
	genre TEXT DEFAULT '',
	track_number INTEGER DEFAULT 0,
	track_total INTEGER DEFAULT 0,
	cover_art TEXT DEFAULT '',
	duration INTEGER NOT NULL,
	location TEXT NOT NULL,
	source_location TEXT DEFAULT '',
	normalized INTEGER NOT NULL DEFAULT 0,
	status TEXT DEFAULT 'deleted',
	is_favourite INTEGER DEFAULT 0,
	added_to_library_at DATETIME,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS queue (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	song_id INTEGER NOT NULL,
	position INTEGER NOT NULL,
	added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (song_id) REFERENCES songs(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS history (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	song_id INTEGER NOT NULL,
	played_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	duration_played INTEGER NOT NULL,
	FOREIGN KEY (song_id) REFERENCES songs(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS state (
	key TEXT PRIMARY KEY,
	value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS playlists (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL UNIQUE,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS playlist_songs (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	playlist_id INTEGER NOT NULL,
	song_id INTEGER NOT NULL,
	position INTEGER NOT NULL,
	added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (playlist_id) REFERENCES playlists(id) ON DELETE CASCADE,
	FOREIGN KEY (song_id) REFERENCES songs(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS listener_snapshots (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	listener_count INTEGER NOT NULL DEFAULT 0,
	current_song_id INTEGER DEFAULT 0,
	recorded_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_queue_position ON queue(position);
CREATE INDEX IF NOT EXISTS idx_history_played_at ON history(played_at);
CREATE INDEX IF NOT EXISTS idx_songs_location ON songs(location);
CREATE INDEX IF NOT EXISTS idx_songs_source_location ON songs(source_location);
CREATE INDEX IF NOT EXISTS idx_songs_is_favourite ON songs(is_favourite);
CREATE INDEX IF NOT EXISTS idx_playlist_songs_playlist_id ON playlist_songs(playlist_id);
CREATE INDEX IF NOT EXISTS idx_listener_snapshots_recorded_at ON listener_snapshots(recorded_at);
`

func RunMigrations(db *sql.DB) error {
	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	alterStmts := []string{
		"ALTER TABLE songs ADD COLUMN album TEXT DEFAULT ''",
		"ALTER TABLE songs ADD COLUMN cover_art TEXT DEFAULT ''",
		"ALTER TABLE songs ADD COLUMN source_location TEXT DEFAULT ''",
		"ALTER TABLE songs ADD COLUMN normalized INTEGER NOT NULL DEFAULT 0",
		"ALTER TABLE songs ADD COLUMN genre TEXT DEFAULT ''",
		"ALTER TABLE songs ADD COLUMN track_number INTEGER DEFAULT 0",
		"ALTER TABLE songs ADD COLUMN track_total INTEGER DEFAULT 0",
		"ALTER TABLE songs ADD COLUMN status TEXT DEFAULT 'deleted'",
		"ALTER TABLE songs ADD COLUMN added_to_library_at DATETIME",
		"ALTER TABLE songs ADD COLUMN is_favourite INTEGER DEFAULT 0",
		"CREATE INDEX IF NOT EXISTS idx_songs_source_location ON songs(source_location)",
		"CREATE INDEX IF NOT EXISTS idx_songs_status ON songs(status)",
		"CREATE INDEX IF NOT EXISTS idx_songs_is_favourite ON songs(is_favourite)",
	}
	for _, stmt := range alterStmts {
		_, _ = db.Exec(stmt)
	}

	return nil
}
