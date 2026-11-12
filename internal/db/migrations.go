package db

import (
	"database/sql"
	"fmt"
)

var statements = []string{
	`CREATE TABLE IF NOT EXISTS preferences (
		name TEXT PRIMARY KEY,
		value TEXT NOT NULL DEFAULT '',
		updated_at INTEGER NOT NULL
	)`,
	`CREATE TABLE IF NOT EXISTS music_info (
		music_id TEXT PRIMARY KEY,
		artists TEXT NOT NULL DEFAULT '',
		title TEXT NOT NULL DEFAULT '',
		album_name TEXT NOT NULL DEFAULT '',
		cover_url TEXT NOT NULL DEFAULT '',
		lyrics TEXT NOT NULL DEFAULT '',
		duration INTEGER NOT NULL DEFAULT 0,
		updated_at INTEGER NOT NULL
	)`,
	`CREATE TABLE IF NOT EXISTS music_url (
		music_id TEXT NOT NULL,
		quality TEXT NOT NULL,
		url TEXT NOT NULL DEFAULT '',
		updated_at INTEGER NOT NULL,
		PRIMARY KEY (music_id, quality)
	)`,
	`CREATE TABLE IF NOT EXISTS task_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		kind TEXT NOT NULL,
		resource TEXT NOT NULL DEFAULT '',
		quality TEXT NOT NULL DEFAULT '',
		success INTEGER NOT NULL DEFAULT 0,
		skipped INTEGER NOT NULL DEFAULT 0,
		failed INTEGER NOT NULL DEFAULT 0,
		created_at INTEGER NOT NULL
	)`,
	`CREATE INDEX IF NOT EXISTS idx_task_history_created ON task_history(created_at DESC)`,
}

func migrate(conn *sql.DB) error {
	for _, stmt := range statements {
		if _, err := conn.Exec(stmt); err != nil {
			return fmt.Errorf("建表失败: %w", err)
		}
	}
	return nil
}
