package db

import (
	"database/sql"
	"errors"
	"strconv"
	"time"
)

const (
	keyLastDownloadConfig = "last_download_config"
	keyMetadataTimestamp  = "last_get_metadata_timestamp"
)

func (d *DB) GetPref(key string) (string, error) {
	var value string
	err := d.conn.QueryRow(`SELECT value FROM preferences WHERE name = ?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return value, err
}

func (d *DB) SetPref(key, value string) error {
	_, err := d.conn.Exec(
		`INSERT INTO preferences (name, value, updated_at) VALUES (?, ?, ?)
		 ON CONFLICT(name) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
		key, value, time.Now().Unix(),
	)
	return err
}

// GetLastDownloadConfig 返回界面上次使用的下载参数 JSON。
func (d *DB) GetLastDownloadConfig() (string, error) {
	return d.GetPref(keyLastDownloadConfig)
}

func (d *DB) SetLastDownloadConfig(raw string) error {
	return d.SetPref(keyLastDownloadConfig, raw)
}

// GetMetadataTimestamp 用于跨任务、跨进程的元数据请求限流。
func (d *DB) GetMetadataTimestamp() (float64, error) {
	raw, err := d.GetPref(keyMetadataTimestamp)
	if err != nil || raw == "" {
		return 0, err
	}
	return strconv.ParseFloat(raw, 64)
}

func (d *DB) SetMetadataTimestamp(ts float64) error {
	return d.SetPref(keyMetadataTimestamp, strconv.FormatFloat(ts, 'f', 6, 64))
}
