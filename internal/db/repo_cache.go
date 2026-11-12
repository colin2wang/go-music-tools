package db

import (
	"database/sql"
	"errors"
	"time"

	"go-music-tools/internal/model"
)

// GetMusicInfo 返回 NextMusic V2 缓存的歌曲信息。
func (d *DB) GetMusicInfo(musicID string) (*model.MusicInfo, bool, error) {
	row := d.conn.QueryRow(
		`SELECT music_id, artists, title, album_name, cover_url, lyrics, duration
		 FROM music_info WHERE music_id = ?`, musicID,
	)
	var info model.MusicInfo
	err := row.Scan(&info.MusicID, &info.Artists, &info.Title, &info.AlbumName,
		&info.CoverURL, &info.Lyrics, &info.Duration)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return &info, true, nil
}

// UpsertMusicInfo 只覆盖非空字段，避免歌词接口先写入时把标题清空。
func (d *DB) UpsertMusicInfo(info model.MusicInfo) error {
	_, err := d.conn.Exec(
		`INSERT INTO music_info
			(music_id, artists, title, album_name, cover_url, lyrics, duration, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(music_id) DO UPDATE SET
			artists    = CASE WHEN excluded.artists    <> '' THEN excluded.artists    ELSE music_info.artists    END,
			title      = CASE WHEN excluded.title      <> '' THEN excluded.title      ELSE music_info.title      END,
			album_name = CASE WHEN excluded.album_name <> '' THEN excluded.album_name ELSE music_info.album_name END,
			cover_url  = CASE WHEN excluded.cover_url  <> '' THEN excluded.cover_url  ELSE music_info.cover_url  END,
			lyrics     = CASE WHEN excluded.lyrics     <> '' THEN excluded.lyrics     ELSE music_info.lyrics     END,
			duration   = CASE WHEN excluded.duration   > 0  THEN excluded.duration   ELSE music_info.duration   END,
			updated_at = excluded.updated_at`,
		info.MusicID, info.Artists, info.Title, info.AlbumName, info.CoverURL,
		info.Lyrics, info.Duration, time.Now().Unix(),
	)
	return err
}

// 源项目的 music_url 缓存只写进程内存，重启即失效；这里真正落到 SQLite。
func (d *DB) GetMusicURL(musicID, quality string) (string, bool, error) {
	var url string
	err := d.conn.QueryRow(
		`SELECT url FROM music_url WHERE music_id = ? AND quality = ?`, musicID, quality,
	).Scan(&url)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return url, url != "", nil
}

func (d *DB) UpsertMusicURL(musicID, quality, url string) error {
	_, err := d.conn.Exec(
		`INSERT INTO music_url (music_id, quality, url, updated_at) VALUES (?, ?, ?, ?)
		 ON CONFLICT(music_id, quality) DO UPDATE SET url = excluded.url, updated_at = excluded.updated_at`,
		musicID, quality, url, time.Now().Unix(),
	)
	return err
}

func (d *DB) AddTaskHistory(rec model.TaskRecord) error {
	if rec.CreatedAt == 0 {
		rec.CreatedAt = time.Now().Unix()
	}
	_, err := d.conn.Exec(
		`INSERT INTO task_history (kind, resource, quality, success, skipped, failed, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		rec.Kind, rec.Resource, rec.Quality, rec.Success, rec.Skipped, rec.Failed, rec.CreatedAt,
	)
	return err
}

func (d *DB) ListTaskHistory(limit int) ([]model.TaskRecord, error) {
	if limit < 1 || limit > 200 {
		limit = 20
	}
	rows, err := d.conn.Query(
		`SELECT id, kind, resource, quality, success, skipped, failed, created_at
		 FROM task_history ORDER BY created_at DESC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]model.TaskRecord, 0, limit)
	for rows.Next() {
		var rec model.TaskRecord
		if err := rows.Scan(&rec.ID, &rec.Kind, &rec.Resource, &rec.Quality,
			&rec.Success, &rec.Skipped, &rec.Failed, &rec.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}
