package model

import "encoding/json"

// SongMeta 是 /Song_V1 与 NextMusic V2 拼装后的统一歌曲元数据契约。
// 注意源 API 的 data 中没有 duration，size 是形如 "23.82MB" 的字符串。
type SongMeta struct {
	Name        string `json:"name"`
	Artists     string `json:"ar_name"`
	Album       string `json:"al_name"`
	ID          string `json:"id"`
	URL         string `json:"url"`
	Level       string `json:"level"`
	Size        string `json:"size"`
	Pic         string `json:"pic"`
	Lyric       string `json:"lyric"`
	Duration    int    `json:"duration"`
	UsedQuality string `json:"used_quality"`
}

type SongDetail struct {
	Index   int    `json:"index"`
	ID      string `json:"id"`
	Name    string `json:"name"`
	Artists string `json:"artists"`
	Album   string `json:"album"`
	Pic     string `json:"pic"`
}

type AlbumMeta struct {
	AlbumID     string          `json:"album_id"`
	Name        string          `json:"name"`
	Artist      string          `json:"artist"`
	PublishDate string          `json:"publish_date"`
	Description string          `json:"description"`
	CoverURL    string          `json:"cover_url"`
	SongCount   int             `json:"song_count"`
	Songs       []SongDetail    `json:"songs"`
	Raw         json.RawMessage `json:"-"`
}

type PlaylistMeta struct {
	PlaylistID  string       `json:"playlist_id"`
	Name        string       `json:"name"`
	Creator     string       `json:"creator"`
	Description string       `json:"description"`
	CoverURL    string       `json:"cover_url"`
	SongCount   int          `json:"song_count"`
	Songs       []SongDetail `json:"songs"`
}

type QualityOption struct {
	Quality string `json:"quality"`
	Kbps    string `json:"kbps"`
	Size    string `json:"size"`
	URL     string `json:"url"`
	Ext     string `json:"ext"`
}

type OOSong struct {
	SongTitle  string `json:"song_title"`
	SingerName string `json:"singer_name"`
	SongMid    string `json:"song_mid"`
	Pay        string `json:"pay"`
}

type OODetail struct {
	Title          string          `json:"title"`
	Artist         string          `json:"artist"`
	Album          string          `json:"album"`
	SongMid        string          `json:"song_mid"`
	SongID         string          `json:"song_id"`
	DurationStr    string          `json:"duration_str"`
	VIP            string          `json:"vip"`
	Language       string          `json:"language"`
	Genre          string          `json:"genre"`
	PublicTime     string          `json:"public_time"`
	Quality        string          `json:"quality"`
	Kbps           string          `json:"kbps"`
	SizeStr        string          `json:"size_str"`
	FileExtension  string          `json:"file_extension"`
	CoverURL       string          `json:"cover_url"`
	LyricLRC       string          `json:"lyric_lrc"`
	LyricText      string          `json:"lyric_text"`
	QualityOptions []QualityOption `json:"quality_options"`
}

type MusicInfo struct {
	MusicID   string `json:"music_id"`
	Artists   string `json:"artists"`
	Title     string `json:"title"`
	AlbumName string `json:"album_name"`
	CoverURL  string `json:"cover_url"`
	Lyrics    string `json:"lyrics"`
	Duration  int    `json:"duration"`
}

type TaskRecord struct {
	ID        int64  `json:"id"`
	Kind      string `json:"kind"`
	Resource  string `json:"resource"`
	Quality   string `json:"quality"`
	Success   int    `json:"success"`
	Skipped   int    `json:"skipped"`
	Failed    int    `json:"failed"`
	CreatedAt int64  `json:"created_at"`
}

type FolderResult struct {
	Folder    string `json:"folder"`
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	ElapsedMS int64  `json:"elapsed_ms"`
}
