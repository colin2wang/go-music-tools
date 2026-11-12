package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"go-music-tools/internal/config"
	"go-music-tools/internal/httpclient"
	"go-music-tools/internal/model"
)

var OOQualityKeys = map[string]string{
	"SQ": "sq", "HQ": "hq", "standard": "standard", "low": "fq",
}

var OOQualityPreference = []string{"SQ", "HQ", "standard", "low"}

const OOOrigin = "http://qjjlb.quanjian.com.cn"

type OOMusicClient struct {
	http    *httpclient.Client
	baseURL string
	retries int
	logger  *slog.Logger
}

func NewOOMusic(cfg config.Config, hc *httpclient.Client, logger *slog.Logger) *OOMusicClient {
	hc.SetOrigin(OOOrigin)
	return &OOMusicClient{
		http: hc, baseURL: cfg.Advanced.OOMusicBaseURL,
		retries: cfg.Network.MaxRetries, logger: logger,
	}
}

func (c *OOMusicClient) headers() map[string]string {
	return map[string]string{
		"origin":         OOOrigin,
		"referer":        OOOrigin + "/",
		"accept":         "*/*",
		"sec-fetch-site": "cross-site",
	}
}

func (c *OOMusicClient) get(ctx context.Context, params url.Values) ([]byte, error) {
	endpoint := c.baseURL + "?" + params.Encode()
	resp, err := c.http.RequestWithRetry(ctx, http.MethodGet, endpoint, nil, c.headers(), c.retries)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OO Music 接口返回 HTTP %d", resp.StatusCode)
	}
	return resp.Body, nil
}

func AnyToString(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case json.Number:
		return string(x)
	case bool:
		return strconv.FormatBool(x)
	default:
		return fmt.Sprint(x)
	}
}

func firstNonEmpty(values ...any) string {
	for _, v := range values {
		if s := strings.TrimSpace(AnyToString(v)); s != "" && s != "0" {
			return s
		}
	}
	return ""
}

func (c *OOMusicClient) Search(ctx context.Context, keyword string) ([]model.OOSong, error) {
	params := url.Values{"msg": {keyword}, "type": {"json"}}
	body, err := c.get(ctx, params)
	if err != nil {
		return nil, err
	}
	return parseOOSearch(body)
}

func parseOOSearch(body []byte) ([]model.OOSong, error) {
	var list []map[string]any
	if err := json.Unmarshal(body, &list); err != nil {
		var obj map[string]json.RawMessage
		if err2 := json.Unmarshal(body, &obj); err2 != nil {
			return nil, fmt.Errorf("解析 OO Music 搜索响应: %w", err)
		}
		for _, key := range []string{"data", "songs", "list", "result"} {
			if raw, ok := obj[key]; ok {
				if err := json.Unmarshal(raw, &list); err == nil {
					break
				}
			}
		}
	}
	out := make([]model.OOSong, 0, len(list))
	for _, item := range list {
		out = append(out, model.OOSong{
			SongTitle:  firstNonEmpty(item["song_title"], item["song_name"]),
			SingerName: firstNonEmpty(item["singer_name"], item["singer"]),
			SongMid:    firstNonEmpty(item["song_mid"], item["mid"]),
			Pay:        AnyToString(item["pay"]),
		})
	}
	return out, nil
}

// GetSongDetail 保留原始 dict，交由 ParseSongInfo / ListQualityOptions 解析。
func (c *OOMusicClient) GetSongDetail(ctx context.Context, keyword, songMid string) (map[string]any, error) {
	msg := keyword
	if msg == "" {
		msg = songMid
	}
	params := url.Values{"msg": {msg}, "type": {"json"}, "mid": {songMid}}
	body, err := c.get(ctx, params)
	if err != nil {
		return nil, err
	}
	var detail map[string]any
	if err := json.Unmarshal(body, &detail); err != nil {
		var list []map[string]any
		if err2 := json.Unmarshal(body, &list); err2 == nil && len(list) > 0 {
			detail = list[0]
		} else {
			return nil, fmt.Errorf("解析歌曲详情: %w", err)
		}
	}
	if firstNonEmpty(detail["song_mid"], detail["mid"]) == "" {
		return nil, fmt.Errorf("歌曲详情无效（缺少 song_mid）")
	}
	return detail, nil
}

func OOQualitySuffix(quality string) string {
	if suffix, ok := OOQualityKeys[quality]; ok {
		return suffix
	}
	return "hq"
}

// OOInferExtension 按 song_filename_{suffix} 的前缀推断扩展名，修复原版选低音质时扩展名错配。
func OOInferExtension(filename, quality string) string {
	lower := strings.ToLower(filename)
	switch {
	case strings.HasPrefix(lower, "f000"):
		return ".flac"
	case strings.HasPrefix(lower, "c600"), strings.HasPrefix(lower, "c400"), strings.HasPrefix(lower, "c200"):
		return ".m4a"
	}
	if quality == "SQ" {
		return ".flac"
	}
	return ".m4a"
}

func ooString(detail map[string]any, key string) string {
	return AnyToString(detail[key])
}

// OOListQualityOptions 只返回实际存在播放地址的音质档位。
func OOListQualityOptions(detail map[string]any) []model.QualityOption {
	out := make([]model.QualityOption, 0, len(OOQualityPreference))
	for _, quality := range OOQualityPreference {
		suffix := OOQualitySuffix(quality)
		playURL := ooString(detail, "song_play_url_"+suffix)
		if playURL == "" {
			continue
		}
		filename := ooString(detail, "song_filename_"+suffix)
		out = append(out, model.QualityOption{
			Quality: quality,
			Kbps:    ooString(detail, "kbps_"+suffix),
			Size:    firstNonEmpty(detail["song_size_"+suffix+"_str"], detail["song_size_"+suffix]),
			URL:     playURL,
			Ext:     OOInferExtension(filename, quality),
		})
	}
	return out
}

// OOGetPlayURL 按 [请求音质] + 剩余优先级回退，返回实际音质、地址与扩展名。
func OOGetPlayURL(detail map[string]any, quality string) (string, string, string, string, error) {
	order := []string{quality}
	for _, candidate := range OOQualityPreference {
		if candidate != quality {
			order = append(order, candidate)
		}
	}
	for _, q := range order {
		suffix := OOQualitySuffix(q)
		playURL := ooString(detail, "song_play_url_"+suffix)
		if playURL == "" {
			continue
		}
		filename := ooString(detail, "song_filename_"+suffix)
		ext := OOInferExtension(filename, q)
		kbps := ooString(detail, "kbps_"+suffix)
		return playURL, q, ext, kbps, nil
	}
	return "", "", "", "", fmt.Errorf("该歌曲没有可用的播放地址")
}

func durationLabel(detail map[string]any) string {
	raw := firstNonEmpty(detail["song_play_time"], detail["duration"])
	if raw == "" {
		return ""
	}
	seconds, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || seconds <= 0 {
		return raw
	}
	return fmt.Sprintf("%02d:%02d", seconds/60, seconds%60)
}

// OOParseSongInfo 把详情 dict 转成前端展示用的 OODetail。
func OOParseSongInfo(detail map[string]any) model.OODetail {
	options := OOListQualityOptions(detail)
	quality := ""
	kbps := ""
	size := ""
	ext := ".m4a"
	if len(options) > 0 {
		quality = options[0].Quality
		kbps = options[0].Kbps
		size = options[0].Size
		ext = options[0].Ext
	}
	return model.OODetail{
		Title:          firstNonEmpty(detail["song_name"], detail["song_title"]),
		Artist:         ooString(detail, "singer_name"),
		Album:          firstNonEmpty(detail["album_name"], detail["album_title"]),
		SongMid:        firstNonEmpty(detail["song_mid"], detail["mid"]),
		SongID:         ooString(detail, "song_id"),
		DurationStr:    durationLabel(detail),
		VIP:            ooString(detail, "vip"),
		Language:       ooString(detail, "language"),
		Genre:          ooString(detail, "genre"),
		PublicTime:     ooString(detail, "public_time"),
		Quality:        quality,
		Kbps:           kbps,
		SizeStr:        size,
		FileExtension:  ext,
		CoverURL:       ooString(detail, "album_pic"),
		LyricLRC:       ooString(detail, "song_lyric"),
		LyricText:      ooString(detail, "lyric"),
		QualityOptions: options,
	}
}
