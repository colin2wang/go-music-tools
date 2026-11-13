package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"go-music-tools/internal/config"
	"go-music-tools/internal/httpclient"
	"go-music-tools/internal/model"
)

var shareIDRe = regexp.MustCompile(`id=(\d+)`)
var pureIDRe = regexp.MustCompile(`^\d+$`)
var looseIDRe = regexp.MustCompile(`\d{4,}`)

// ExtractResourceID 支持纯数字 ID 与 music.163.com / 163cn.tv 分享链接。
func ExtractResourceID(raw string) string {
	raw = strings.TrimSpace(raw)
	if pureIDRe.MatchString(raw) {
		return raw
	}
	if m := shareIDRe.FindStringSubmatch(raw); len(m) > 1 {
		return m[1]
	}
	if all := looseIDRe.FindAllString(raw, -1); len(all) > 0 {
		return all[len(all)-1]
	}
	return raw
}

type NetEaseClient struct {
	http    *httpclient.Client
	baseURL string
	retries int
	logger  *slog.Logger
	cache   SongCache
}

// SongCache 由 internal/store.Store 实现，避免 api 包直接依赖具体存储。
type SongCache interface {
	GetNetEaseSong(songID, level string) (*model.SongMeta, bool)
	UpsertNetEaseSong(meta model.SongMeta, level string) error
	DeleteNetEaseSong(songID, level string)
}

func NewNetEase(cfg config.Config, hc *httpclient.Client, logger *slog.Logger, cache SongCache) *NetEaseClient {
	base := strings.TrimRight(cfg.API.BaseURL, "/")
	hc.SetOrigin(base)
	return &NetEaseClient{
		http: hc, baseURL: base,
		retries: cfg.Network.MaxRetries, logger: logger, cache: cache,
	}
}

func (c *NetEaseClient) endpoint(path string) string { return c.baseURL + path }

func (c *NetEaseClient) headers() map[string]string {
	return map[string]string{
		"x-requested-with": "XMLHttpRequest",
		"origin":           c.baseURL,
		"referer":          c.baseURL + "/",
	}
}

func jsonString(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	return strings.Trim(string(raw), `"`)
}

func artistsString(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var list []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(raw, &list); err == nil {
		names := make([]string, 0, len(list))
		for _, item := range list {
			if item.Name != "" {
				names = append(names, item.Name)
			}
		}
		return strings.Join(names, ", ")
	}
	var plain []string
	if err := json.Unmarshal(raw, &plain); err == nil {
		return strings.Join(plain, ", ")
	}
	return ""
}

func creatorString(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return "Unknown Creator"
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		if s == "" {
			return "Unknown Creator"
		}
		return s
	}
	var obj struct {
		Nickname string `json:"nickname"`
	}
	if err := json.Unmarshal(raw, &obj); err == nil && obj.Nickname != "" {
		return obj.Nickname
	}
	return "Unknown Creator"
}

// ParseSong 对应 POST /Song_V1。源 API 的 data 没有 duration，size 是字符串。
func (c *NetEaseClient) ParseSong(ctx context.Context, songID, level string) (*model.SongMeta, error) {
	if c.cache != nil {
		if cached, ok := c.cache.GetNetEaseSong(songID, level); ok {
			return cached, nil
		}
	}
	form := url.Values{"url": {songID}, "level": {level}, "type": {"json"}}
	headers := c.headers()
	headers["content-type"] = "application/x-www-form-urlencoded; charset=UTF-8"

	resp, err := c.http.RequestWithRetry(ctx, http.MethodPost, c.endpoint("/Song_V1"),
		func() io.Reader { return strings.NewReader(form.Encode()) }, headers, c.retries)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("/Song_V1 返回 HTTP %d", resp.StatusCode)
	}
	var payload struct {
		Status  int    `json:"status"`
		Success bool   `json:"success"`
		Message string `json:"message"`
		Data    struct {
			AlName string          `json:"al_name"`
			ArName string          `json:"ar_name"`
			ID     json.RawMessage `json:"id"`
			Level  string          `json:"level"`
			Lyric  string          `json:"lyric"`
			Name   string          `json:"name"`
			Pic    string          `json:"pic"`
			Size   json.RawMessage `json:"size"`
			TLyric string          `json:"tlyric"`
			URL    string          `json:"url"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp.Body, &payload); err != nil {
		return nil, fmt.Errorf("解析歌曲响应: %w", err)
	}
	if payload.Data.URL == "" {
		return nil, fmt.Errorf("歌曲 %s 未返回下载地址（%s）", songID, payload.Message)
	}
	meta := &model.SongMeta{
		Name: payload.Data.Name, Artists: payload.Data.ArName, Album: payload.Data.AlName,
		ID: jsonString(payload.Data.ID), URL: payload.Data.URL, Level: payload.Data.Level,
		Size: jsonString(payload.Data.Size), Pic: payload.Data.Pic, Lyric: payload.Data.Lyric,
		UsedQuality: payload.Data.Level,
	}
	// 成功且标题、地址非空才入缓存；缓存随地址 TTL 一起过期，过期后重新请求。
	if c.cache != nil {
		_ = c.cache.UpsertNetEaseSong(*meta, level)
	}
	return meta, nil
}

// InvalidateSong 使网易云主通道的整曲缓存失效（播放地址过期重取时调用）。
func (c *NetEaseClient) InvalidateSong(songID, level string) {
	if c.cache != nil {
		c.cache.DeleteNetEaseSong(songID, level)
	}
}

type albumTrack struct {
	Album   string          `json:"album"`
	Artists json.RawMessage `json:"artists"`
	ID      json.RawMessage `json:"id"`
	Name    string          `json:"name"`
	PicURL  string          `json:"picUrl"`
}

type playlistTrack struct {
	Album   json.RawMessage `json:"album"`
	Artists json.RawMessage `json:"artists"`
	ID      json.RawMessage `json:"id"`
	Name    string          `json:"name"`
	PicURL  string          `json:"picUrl"`
}

func (c *NetEaseClient) ParseAlbum(ctx context.Context, albumID string) (*model.AlbumMeta, error) {
	endpoint := c.endpoint("/Album?id=" + url.QueryEscape(albumID))
	resp, err := c.http.RequestWithRetry(ctx, http.MethodGet, endpoint, nil, c.headers(), c.retries)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("/Album 返回 HTTP %d", resp.StatusCode)
	}
	var payload struct {
		Data struct {
			Album struct {
				Artist      string          `json:"artist"`
				CoverImgURL string          `json:"coverImgUrl"`
				Description string          `json:"description"`
				ID          json.RawMessage `json:"id"`
				Name        string          `json:"name"`
				PublishTime int64           `json:"publishTime"`
				Songs       []albumTrack    `json:"songs"`
			} `json:"album"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp.Body, &payload); err != nil {
		return nil, fmt.Errorf("解析专辑响应: %w", err)
	}
	album := payload.Data.Album
	if album.Name == "" {
		return nil, fmt.Errorf("专辑 %s 响应中缺少专辑信息", albumID)
	}

	publishDate := ""
	if album.PublishTime > 0 {
		publishDate = time.UnixMilli(album.PublishTime).Format("2006-01-02")
	}
	songs := make([]model.SongDetail, 0, len(album.Songs))
	for i, track := range album.Songs {
		songs = append(songs, model.SongDetail{
			Index: i + 1, ID: jsonString(track.ID), Name: track.Name,
			Artists: artistsString(track.Artists), Album: track.Album, Pic: track.PicURL,
		})
	}
	return &model.AlbumMeta{
		AlbumID: jsonString(album.ID), Name: album.Name, Artist: album.Artist,
		PublishDate: publishDate, Description: album.Description, CoverURL: album.CoverImgURL,
		SongCount: len(songs), Songs: songs, Raw: append(json.RawMessage(nil), resp.Body...),
	}, nil
}

func (c *NetEaseClient) ParsePlaylist(ctx context.Context, playlistID string) (*model.PlaylistMeta, error) {
	endpoint := c.endpoint("/Playlist?id=" + url.QueryEscape(playlistID))
	resp, err := c.http.RequestWithRetry(ctx, http.MethodGet, endpoint, nil, c.headers(), c.retries)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("/Playlist 返回 HTTP %d", resp.StatusCode)
	}
	var payload struct {
		Data struct {
			Playlist struct {
				CoverImgURL string          `json:"coverImgUrl"`
				Creator     json.RawMessage `json:"creator"`
				Description string          `json:"description"`
				ID          json.RawMessage `json:"id"`
				Name        string          `json:"name"`
				TrackCount  int             `json:"trackCount"`
				Tracks      []playlistTrack `json:"tracks"`
			} `json:"playlist"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp.Body, &payload); err != nil {
		return nil, fmt.Errorf("解析歌单响应: %w", err)
	}
	pl := payload.Data.Playlist
	if pl.Name == "" {
		return nil, fmt.Errorf("歌单 %s 响应中缺少歌单信息", playlistID)
	}
	songs := make([]model.SongDetail, 0, len(pl.Tracks))
	for i, track := range pl.Tracks {
		var albumName string
		_ = json.Unmarshal(track.Album, &albumName)
		if albumName == "" {
			var obj struct {
				Name string `json:"name"`
			}
			if err := json.Unmarshal(track.Album, &obj); err == nil {
				albumName = obj.Name
			}
		}
		songs = append(songs, model.SongDetail{
			Index: i + 1, ID: jsonString(track.ID), Name: track.Name,
			Artists: artistsString(track.Artists), Album: albumName, Pic: track.PicURL,
		})
	}
	count := pl.TrackCount
	if count == 0 {
		count = len(songs)
	}
	return &model.PlaylistMeta{
		PlaylistID: jsonString(pl.ID), Name: pl.Name, Creator: creatorString(pl.Creator),
		Description: pl.Description, CoverURL: pl.CoverImgURL, SongCount: count, Songs: songs,
	}, nil
}
