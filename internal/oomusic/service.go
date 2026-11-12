package oomusic

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
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

	mu          sync.Mutex
	searchCache map[string][]model.OOSong
}

func New(cfg config.Config, root string, hc *httpclient.Client, logger *slog.Logger) *Service {
	return &Service{
		Cfg: cfg, Root: root, HTTP: hc,
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
	s.mu.Lock()
	s.searchCache[keyword] = songs
	s.mu.Unlock()
	if s.Logger != nil {
		s.Logger.Info("OO Music 搜索完成", "keyword", keyword, "count", len(songs))
	}
	return songs, nil
}

func (s *Service) FetchDetail(ctx context.Context, keyword, songMid string) (map[string]any, error) {
	return s.API.GetSongDetail(ctx, keyword, songMid)
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
func (s *Service) Download(ctx context.Context, keyword, songMid, quality string) (string, bool, error) {
	detail, err := s.FetchDetail(ctx, keyword, songMid)
	if err != nil {
		return "", false, err
	}
	if quality == "" {
		quality = s.Cfg.Advanced.OOMusicDefaultQuality
	}
	playURL, actualQuality, ext, kbps, err := api.OOGetPlayURL(detail, quality)
	if err != nil {
		return "", false, err
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
	dest := filepath.Join(dir, base+ext)

	if s.Cfg.Download.SkipExisting {
		if stat, err := os.Stat(dest); err == nil && stat.Size() > 0 {
			if s.Logger != nil {
				s.Logger.Info("OO Music 文件已存在，跳过", "file", base+ext)
			}
			return dest, true, nil
		}
	}

	if err := httpclient.RandomDelay(ctx, s.Cfg.Network.APIDelayMin, s.Cfg.Network.RandomDelayMax); err != nil {
		return "", false, err
	}
	if _, err := s.HTTP.Download(ctx, playURL, dest, nil, nil); err != nil {
		return "", false, fmt.Errorf("下载 OO Music 音频失败: %w", err)
	}
	if s.Logger != nil {
		s.Logger.Info("OO Music 音频下载完成", "file", base+ext, "quality", actualQuality, "kbps", kbps)
	}

	stem := filepath.Join(dir, base)
	var cover []byte
	coverMIME := ""
	if info.CoverURL != "" {
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

func (s *Service) warn(message string, err error) {
	if s.Logger != nil {
		s.Logger.Warn(message, "error", err)
	}
}
