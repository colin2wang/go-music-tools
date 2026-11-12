package bindings

import (
	"context"
	"fmt"
	"strings"

	"go-music-tools/internal/api"
	"go-music-tools/internal/model"
	"go-music-tools/internal/music"
	"go-music-tools/internal/task"
)

func (s *Service) musicService() *music.Service {
	return music.New(s.cfg(), s.deps.Root, s.httpClient(), s.deps.Logger, s.deps.Cache)
}

// Download 启动一个异步下载任务，立即返回 task_id，进度通过 task:progress 事件推送。
func (s *Service) Download(req DownloadRequest) (string, error) {
	cfg := s.cfg()
	kind := strings.ToLower(strings.TrimSpace(req.Kind))
	if kind != "song" && kind != "album" && kind != "playlist" {
		return "", fmt.Errorf("不支持的下载方式: %q", req.Kind)
	}
	resourceID := api.ExtractResourceID(strings.TrimSpace(req.ResourceID))
	if resourceID == "" {
		return "", fmt.Errorf("请填写歌曲、专辑或歌单 ID")
	}
	quality := strings.TrimSpace(req.Quality)
	if quality == "" {
		quality = cfg.Download.DefaultQuality
	}
	if !contains(cfgQualityLevels(), quality) {
		return "", fmt.Errorf("不支持的音质档位: %s", quality)
	}
	indexes := append([]int(nil), req.Indexes...)

	_ = s.SaveLastDownloadConfig(LastDownloadConfig{
		Kind: kind, ResourceID: resourceID, Quality: quality, Indexes: indexes,
	})

	svc := s.musicService()
	return s.Start("download:"+kind, 1, func(ctx context.Context, r *task.Reporter) error {
		s.deps.Logger.Info("下载任务开始", "kind", kind, "resource_id", resourceID, "quality", quality, "indexes", len(indexes))
		var err error
		switch kind {
		case "song":
			err = svc.DownloadSong(ctx, resourceID, quality, r)
		case "album":
			err = svc.DownloadAlbum(ctx, resourceID, indexes, quality, r)
		case "playlist":
			err = svc.DownloadPlaylist(ctx, resourceID, indexes, quality, r)
		}
		if err != nil {
			s.deps.Logger.Error("下载任务失败", "kind", kind, "error", err)
		}
		return err
	})
}

// DownloadSongs 把搜索页多选的歌曲合并为一个任务，避免多个主任务互相冲突。
func (s *Service) DownloadSongs(ids []string, quality string) (string, error) {
	cfg := s.cfg()
	if len(ids) == 0 {
		return "", fmt.Errorf("请先选择要下载的歌曲")
	}
	if strings.TrimSpace(quality) == "" {
		quality = cfg.Download.DefaultQuality
	}
	normalized := make([]string, 0, len(ids))
	for _, id := range ids {
		clean := api.ExtractResourceID(strings.TrimSpace(id))
		if clean != "" {
			normalized = append(normalized, clean)
		}
	}
	svc := s.musicService()
	return s.Start("download:songs", len(normalized), func(ctx context.Context, r *task.Reporter) error {
		s.deps.Logger.Info("批量单曲下载开始", "count", len(normalized), "quality", quality)
		for _, id := range normalized {
			if err := ctx.Err(); err != nil {
				return err
			}
			if r != nil {
				r.Stage("下载歌曲 " + id)
			}
			err := svc.DownloadSong(ctx, id, quality, nil)
			status, message := "success", id
			if err != nil {
				status, message = "failed", err.Error()
				s.deps.Logger.Error("单曲下载失败", "song_id", id, "error", err)
			}
			if r != nil {
				r.Item(id, status, message)
			}
		}
		return nil
	})
}

func (s *Service) ParseIndexes(raw string) []int { return music.ParseIndexes(raw) }

func (s *Service) ExtractResourceID(raw string) string { return api.ExtractResourceID(raw) }

func (s *Service) SearchSongs(keyword string) ([]model.SearchSong, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return []model.SearchSong{}, nil
	}
	songs, err := s.musicService().Search(context.Background(), keyword)
	if err != nil {
		return nil, err
	}
	s.deps.Logger.Info("搜索完成", "keyword", keyword, "count", len(songs))
	return songs, nil
}

func cfgQualityLevels() []string {
	return []string{"standard", "exhigh", "lossless", "hires", "sky", "jyeffect", "jymaster"}
}

func contains(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}

func intsToString(values []int) string {
	parts := make([]string, 0, len(values))
	for _, v := range values {
		parts = append(parts, fmt.Sprintf("%d", v))
	}
	return strings.Join(parts, " ")
}
