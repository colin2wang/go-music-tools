package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go-music-tools/internal/config"
	"go-music-tools/internal/httpclient"
	"go-music-tools/internal/model"
)

// MusicInfoCache 由 internal/store.Store 实现，避免 api 包直接依赖具体存储。
type MusicInfoCache interface {
	GetMusicInfo(musicID string) (*model.MusicInfo, bool, error)
	UpsertMusicInfo(info model.MusicInfo) error
	GetMusicURL(musicID, quality string) (string, bool, error)
	UpsertMusicURL(musicID, quality, url string) error
}

type NextMusicClient struct {
	http    *httpclient.Client
	baseURL string
	ip      string
	logger  *slog.Logger
	cache   MusicInfoCache
}

func NewNextMusic(cfg config.Config, hc *httpclient.Client, logger *slog.Logger, cache MusicInfoCache) *NextMusicClient {
	fingerprinted := hc.WithChromeFingerprint()
	fingerprinted.SetOrigin("https://wyapi.toubiec.cn")
	return &NextMusicClient{
		http:    fingerprinted,
		baseURL: strings.TrimRight(cfg.Advanced.NextMusicBaseURL, "/"),
		ip:      cfg.Advanced.NextMusicIP,
		logger:  logger,
		cache:   cache,
	}
}

func codeValue(raw json.RawMessage) int {
	if len(raw) == 0 {
		return 0
	}
	var n int
	if err := json.Unmarshal(raw, &n); err == nil {
		return n
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		n, _ = strconv.Atoi(s)
		return n
	}
	return 0
}

func (c *NextMusicClient) post(ctx context.Context, path string, payload map[string]any) (json.RawMessage, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	headers := map[string]string{
		"content-type":   "application/json",
		"origin":         "https://wyapi.toubiec.cn",
		"referer":        "https://wyapi.toubiec.cn/",
		"sec-fetch-site": "same-site",
	}
	resp, err := c.http.RequestWithRetry(ctx, http.MethodPost, c.baseURL+path,
		func() io.Reader { return strings.NewReader(string(body)) }, headers, 4)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s 返回 HTTP %d", path, resp.StatusCode)
	}
	var out struct {
		Code    json.RawMessage `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(resp.Body, &out); err != nil {
		return nil, fmt.Errorf("解析 %s 响应: %w", path, err)
	}
	if codeValue(out.Code) != 200 || len(out.Data) == 0 || string(out.Data) == "null" {
		return nil, fmt.Errorf("%s 返回异常: code=%s message=%s", path, string(out.Code), out.Message)
	}
	return out.Data, nil
}

func (c *NextMusicClient) requestPayload(id, level string) map[string]any {
	payload := map[string]any{
		"id":        id,
		"timestamp": time.Now().UnixMilli(),
		"ip":        c.ip,
	}
	if level != "" {
		payload["level"] = level
	}
	return payload
}

func (c *NextMusicClient) GetSongInfo(ctx context.Context, id string) (*model.MusicInfo, error) {
	if c.cache != nil {
		if cached, ok, err := c.cache.GetMusicInfo(id); err == nil && ok && cached.Title != "" {
			return cached, nil
		}
	}
	data, err := c.post(ctx, "/api/getSongInfo", c.requestPayload(id, ""))
	if err != nil {
		return nil, err
	}
	var raw struct {
		ID       json.RawMessage `json:"id"`
		Singer   string          `json:"singer"`
		Name     string          `json:"name"`
		Album    string          `json:"album"`
		PicImg   string          `json:"picimg"`
		Duration json.RawMessage `json:"duration"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	musicID := jsonString(raw.ID)
	if musicID == "" {
		musicID = id
	}
	duration, _ := strconv.Atoi(jsonString(raw.Duration))
	info := &model.MusicInfo{
		MusicID: musicID, Artists: raw.Singer, Title: raw.Name,
		AlbumName: raw.Album, CoverURL: raw.PicImg, Duration: duration,
	}
	if c.cache != nil {
		_ = c.cache.UpsertMusicInfo(*info)
	}
	return info, nil
}

func (c *NextMusicClient) GetSongLyric(ctx context.Context, id string) (string, error) {
	if c.cache != nil {
		if cached, ok, err := c.cache.GetMusicInfo(id); err == nil && ok && cached.Lyrics != "" {
			return cached.Lyrics, nil
		}
	}
	data, err := c.post(ctx, "/api/getSongLyric", c.requestPayload(id, ""))
	if err != nil {
		return "", err
	}
	var raw struct {
		LRC string `json:"lrc"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return "", err
	}
	if c.cache != nil && raw.LRC != "" {
		_ = c.cache.UpsertMusicInfo(model.MusicInfo{MusicID: id, Lyrics: raw.LRC})
	}
	return raw.LRC, nil
}

func (c *NextMusicClient) GetSongURL(ctx context.Context, id, level string) (string, string, error) {
	if c.cache != nil {
		if cached, ok, err := c.cache.GetMusicURL(id, level); err == nil && ok {
			return cached, level, nil
		}
	}
	data, err := c.post(ctx, "/api/getSongUrl", c.requestPayload(id, level))
	if err != nil {
		return "", "", err
	}
	var raw struct {
		URL   string          `json:"url"`
		Level string          `json:"level"`
		ID    json.RawMessage `json:"id"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return "", "", err
	}
	if raw.URL == "" {
		return "", "", fmt.Errorf("备用通道未返回歌曲地址")
	}
	actual := raw.Level
	if actual == "" {
		actual = level
	}
	if c.cache != nil {
		_ = c.cache.UpsertMusicURL(id, actual, raw.URL)
	}
	return raw.URL, actual, nil
}

// Resolve 把三个接口的结果拼装成主流程使用的统一 SongMeta。
// 源项目该通道的 data.size 恒为空、lyric 可能缺失，这里保持一致。
func (c *NextMusicClient) Resolve(ctx context.Context, id, level string) (*model.SongMeta, error) {
	info, err := c.GetSongInfo(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("获取歌曲信息失败: %w", err)
	}
	url, actualLevel, err := c.GetSongURL(ctx, id, level)
	if err != nil {
		return nil, fmt.Errorf("获取歌曲地址失败: %w", err)
	}
	lyric, err := c.GetSongLyric(ctx, id)
	if err != nil {
		if c.logger != nil {
			c.logger.Warn("备用通道歌词获取失败，继续下载", "id", id, "error", err)
		}
		lyric = ""
	}
	return &model.SongMeta{
		Name:        info.Title,
		Artists:     strings.ReplaceAll(info.Artists, "/", ", "),
		Album:       info.AlbumName,
		ID:          info.MusicID,
		URL:         url,
		Level:       actualLevel,
		Pic:         info.CoverURL,
		Lyric:       lyric,
		Duration:    info.Duration,
		UsedQuality: actualLevel,
	}, nil
}
