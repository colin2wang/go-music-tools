package oomusic

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"go-music-tools/internal/api"
	"go-music-tools/internal/config"
	"go-music-tools/internal/httpclient"
	"go-music-tools/internal/model"
	"go-music-tools/internal/tag"
	"go-music-tools/internal/task"
)

var illegalRe = regexp.MustCompile(`[<>:"/\\|?*]`)

func SafePath(name string) string {
	return strings.Trim(strings.TrimSpace(illegalRe.ReplaceAllString(name, "_")), ". ")
}

type Service struct {
	Cfg    config.Config
	Root   string
	HTTP   *httpclient.Client
	API    *api.OOMusicClient
	Logger *slog.Logger
	Cache  DetailCache

	mu          sync.Mutex
	searchCache map[string][]model.OOSong
}

// DetailCache 由 internal/store.Store 实现，避免本包直接依赖具体存储。
type DetailCache interface {
	GetOODetail(keyword, songMid string) (map[string]any, bool)
	UpsertOODetail(keyword, songMid string, detail map[string]any, raw string) error
	DeleteOODetail(keyword, songMid string)
}

func New(cfg config.Config, root string, hc *httpclient.Client, logger *slog.Logger, cache DetailCache) *Service {
	return &Service{
		Cfg: cfg, Root: root, HTTP: hc, Cache: cache,
		API: api.NewOOMusic(cfg, hc, logger), Logger: logger,
		searchCache: make(map[string][]model.OOSong),
	}
}

func (s *Service) resolve(path string) string {
	if path == "" {
		return s.Root
	}
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(s.Root, path)
}

func (s *Service) DownloadDir() string {
	return s.resolve(s.Cfg.Advanced.OOMusicDownloadDir)
}

// Search 复用进程内缓存，避免重复触发代理限流（原版亦有此缓存，但不持久化）。
func (s *Service) Search(ctx context.Context, keyword string) ([]model.OOSong, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return []model.OOSong{}, nil
	}
	s.mu.Lock()
	cached, ok := s.searchCache[keyword]
	s.mu.Unlock()
	if ok {
		if s.Logger != nil {
			s.Logger.Info("OO Music 搜索命中缓存", "keyword", keyword, "count", len(cached))
		}
		return cached, nil
	}
	songs, err := s.API.Search(ctx, keyword)
	if err != nil {
		return nil, err
	}
	// 空结果不入缓存：避免代理偶发返回空列表后被永久缓存，导致后续搜索始终为 0。
	if len(songs) > 0 {
		s.mu.Lock()
		s.searchCache[keyword] = songs
		s.mu.Unlock()
	}
	if s.Logger != nil {
		s.Logger.Info("OO Music 搜索完成", "keyword", keyword, "count", len(songs))
	}
	return songs, nil
}

func (s *Service) FetchDetail(ctx context.Context, keyword, songMid string) (map[string]any, error) {
	if s.Cache != nil {
		if cached, ok := s.Cache.GetOODetail(keyword, songMid); ok {
			return cached, nil
		}
	}
	detail, raw, err := s.API.GetSongDetail(ctx, keyword, songMid)
	if err != nil {
		return nil, err
	}
	// 空详情不入缓存；成功详情随播放地址 TTL 一起过期，过期后整体重新拉取。
	if s.Cache != nil && len(detail) > 0 {
		_ = s.Cache.UpsertOODetail(keyword, songMid, detail, raw)
	}
	return detail, nil
}

// InvalidateDetail 使指定歌曲的 OO Music 详情缓存失效（播放地址过期重取时调用）。
func (s *Service) InvalidateDetail(keyword, songMid string) {
	if s.Cache != nil {
		s.Cache.DeleteOODetail(keyword, songMid)
	}
}

func (s *Service) Detail(ctx context.Context, keyword, songMid string) (*model.OODetail, error) {
	detail, err := s.FetchDetail(ctx, keyword, songMid)
	if err != nil {
		return nil, err
	}
	info := api.OOParseSongInfo(detail)
	return &info, nil
}

func (s *Service) Qualities(ctx context.Context, keyword, songMid string) ([]model.QualityOption, error) {
	detail, err := s.FetchDetail(ctx, keyword, songMid)
	if err != nil {
		return nil, err
	}
	return api.OOListQualityOptions(detail), nil
}

// Download 下载单曲并写入封面、LRC 与标签；已存在文件直接跳过。
// 播放地址失效（403/404/410）时使详情缓存失效、重新拉取地址并重试一次。
func (s *Service) Download(ctx context.Context, keyword, songMid, quality string) (string, bool, error) {
	detail, err := s.FetchDetail(ctx, keyword, songMid)
	if err != nil {
		return "", false, err
	}
	if quality == "" {
		quality = s.Cfg.Advanced.OOMusicDefaultQuality
	}
	info := api.OOParseSongInfo(detail)

	dir := s.DownloadDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", false, err
	}
	base := SafePath(info.Artist) + " - " + SafePath(info.Title)
	if base == " - " {
		base = "OO Music - " + SafePath(songMid)
	}

	var dest, ext, actualQuality, kbps string
	for attempt := 0; ; attempt++ {
		playURL, aq, e, kb, err := api.OOGetPlayURL(detail, quality)
		if err != nil {
			return "", false, err
		}
		actualQuality, ext, kbps = aq, e, kb
		if s.Logger != nil {
			s.Logger.Info("OO Music 音频资源 URL", "song_mid", songMid, "quality", actualQuality, "url", playURL)
		}
		dest = filepath.Join(dir, base+ext)
		if attempt == 0 && s.Cfg.Download.SkipExisting {
			if stat, err := os.Stat(dest); err == nil && stat.Size() > 0 {
				if s.Logger != nil {
					s.Logger.Info("OO Music 文件已存在，跳过", "file", base+ext)
				}
				return dest, true, nil
			}
		}
		if err := httpclient.RandomDelay(ctx, s.Logger, s.Cfg.Network.APIDelayMin, s.Cfg.Network.RandomDelayMax); err != nil {
			return "", false, err
		}
		_, dlErr := s.HTTP.Download(ctx, playURL, dest, nil, httpclient.ThrottledProgress(s.Logger, base+ext))
		if dlErr == nil {
			break
		}
		// 地址过期：失效详情缓存、重新拉取地址后重试一次；其余错误直接返回。
		if attempt > 0 || !httpclient.IsURLExpired(dlErr) {
			return "", false, fmt.Errorf("下载 OO Music 音频失败: %w", dlErr)
		}
		if s.Logger != nil {
			s.Logger.Info("OO Music 播放地址已过期，重新获取后重试", "song", base)
		}
		s.InvalidateDetail(keyword, songMid)
		detail, err = s.FetchDetail(ctx, keyword, songMid)
		if err != nil {
			return "", false, err
		}
	}
	if s.Logger != nil {
		s.Logger.Info("OO Music 音频下载完成", "file", base+ext, "quality", actualQuality, "kbps", kbps)
	}

	stem := filepath.Join(dir, base)
	var cover []byte
	coverMIME := ""
	if info.CoverURL != "" {
		if s.Logger != nil {
			s.Logger.Info("OO Music 封面资源 URL", "song_mid", songMid, "url", info.CoverURL)
		}
		data, contentType, err := s.HTTP.DownloadBytes(ctx, info.CoverURL, nil)
		if err != nil {
			s.warn("下载封面失败", err)
		} else if len(data) > 0 {
			cover = data
			coverMIME = tag.MimeFromData(data)
			if strings.Contains(contentType, "png") {
				coverMIME = "image/png"
			}
			coverPath := stem + ".jpg"
			if coverMIME == "image/png" {
				coverPath = stem + ".png"
			}
			if err := os.WriteFile(coverPath, data, 0o644); err != nil {
				s.warn("保存封面失败", err)
			}
		}
	}
	if info.LyricLRC != "" {
		if err := os.WriteFile(stem+".lrc", []byte(info.LyricLRC), 0o644); err != nil {
			s.warn("保存歌词失败", err)
		}
	}
	if err := WriteTags(dest, info, cover, coverMIME); err != nil {
		s.warn("写入 OO Music 标签失败", err)
	}
	return dest, false, nil
}

// BatchDownload 遍历搜索结果逐首下载，取消信号贯穿循环，单首失败不中断。
func (s *Service) BatchDownload(ctx context.Context, keyword, quality string, r *task.Reporter) error {
	songs, err := s.Search(ctx, keyword)
	if err != nil {
		return err
	}
	if len(songs) == 0 {
		return fmt.Errorf("关键词 %q 没有搜索结果", keyword)
	}
	if r != nil {
		r.SetTotal(len(songs))
	}
	for _, song := range songs {
		if err := ctx.Err(); err != nil {
			return err
		}
		if r != nil {
			r.Stage(fmt.Sprintf("批量下载：%s - %s", song.SingerName, song.SongTitle))
		}
		path, skipped, err := s.Download(ctx, song.SingerName, song.SongMid, quality)
		name := song.SingerName + " - " + song.SongTitle
		if err != nil {
			if s.Logger != nil {
				s.Logger.Error("OO Music 单曲下载失败", "song", name, "error", err)
			}
			if r != nil {
				r.Item(name, "failed", err.Error())
			}
			continue
		}
		status := "success"
		if skipped {
			status = "skipped"
		}
		if r != nil {
			r.Item(name, status, path)
		}
	}
	return ctx.Err()
}

// RetryFailed 只重试批量下载里失败的曲目：重新搜索关键词定位 song_mid，
// 按「歌手 - 歌名」匹配失败子项后逐首复用 Download。
func (s *Service) RetryFailed(ctx context.Context, keyword, quality string,
	failed []task.ItemResult, r *task.Reporter) error {

	if len(failed) == 0 {
		return nil
	}
	songs, err := s.Search(ctx, keyword)
	if err != nil {
		return err
	}
	byName := make(map[string]model.OOSong, len(songs))
	for _, song := range songs {
		byName[song.SingerName+" - "+song.SongTitle] = song
	}
	selected := make([]model.OOSong, 0, len(failed))
	for _, item := range failed {
		if song, ok := byName[item.Name]; ok {
			selected = append(selected, song)
		}
	}
	if len(selected) == 0 {
		return fmt.Errorf("未能在当前搜索结果中定位失败曲目，请重新搜索后下载")
	}
	if r != nil {
		r.SetTotal(len(selected))
	}
	for _, song := range selected {
		if err := ctx.Err(); err != nil {
			return err
		}
		name := song.SingerName + " - " + song.SongTitle
		if r != nil {
			r.Stage("重试下载：" + name)
		}
		path, skipped, err := s.Download(ctx, song.SingerName, song.SongMid, quality)
		if err != nil {
			if s.Logger != nil {
				s.Logger.Error("重试 OO Music 单曲失败", "song", name, "error", err)
			}
			if r != nil {
				r.Item(name, "failed", err.Error())
			}
			continue
		}
		status := "success"
		if skipped {
			status = "skipped"
		}
		if r != nil {
			r.Item(name, status, path)
		}
	}
	return ctx.Err()
}

// SearchCacheEntry 是搜索缓存的单条检视信息。
type SearchCacheEntry struct {
	Keyword string `json:"keyword"`
	Count   int    `json:"count"`
}

// ListSearchCache 返回全部搜索缓存条目，按关键词排序。
func (s *Service) ListSearchCache() []SearchCacheEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]SearchCacheEntry, 0, len(s.searchCache))
	for k, v := range s.searchCache {
		out = append(out, SearchCacheEntry{Keyword: k, Count: len(v)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Keyword < out[j].Keyword })
	return out
}

// GetSearchCacheSongs 返回指定关键词缓存的搜索结果列表。
func (s *Service) GetSearchCacheSongs(keyword string) []model.OOSong {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.searchCache[keyword]
}

// ClearSearchCache 清空全部搜索缓存，或仅删除指定关键词；返回删除的条目数。
func (s *Service) ClearSearchCache(keyword string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if keyword == "" {
		n := len(s.searchCache)
		s.searchCache = make(map[string][]model.OOSong)
		return n
	}
	if _, ok := s.searchCache[keyword]; !ok {
		return 0
	}
	delete(s.searchCache, keyword)
	return 1
}

func (s *Service) warn(message string, err error) {
	if s.Logger != nil {
		s.Logger.Warn(message, "error", err)
	}
}
