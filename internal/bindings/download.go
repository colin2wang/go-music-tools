package bindings

import (
	"context"
	"fmt"
	"strings"

	"go-music-tools/internal/api"
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

	// 重试只针对失败子项：捕获原始请求，在执行时重新构造业务服务以应用最新配置。
	retry := func(failed []task.ItemResult) (task.TaskFunc, error) {
		if len(failed) == 0 {
			return nil, fmt.Errorf("没有可重试的失败曲目")
		}
		return func(ctx context.Context, r *task.Reporter) error {
			svc := s.musicService()
			if kind == "song" {
				return svc.DownloadSong(ctx, resourceID, quality, r)
			}
			return svc.RetryFailedSongs(ctx, kind, resourceID, indexes, quality, failed, r)
		}, nil
	}

	svc := s.musicService()
	return s.start("download:"+kind, 1, func(ctx context.Context, r *task.Reporter) error {
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
	}, task.WithRetry(retry))
}

func (s *Service) ParseIndexes(raw string) []int { return music.ParseIndexes(raw) }

func (s *Service) ExtractResourceID(raw string) string { return api.ExtractResourceID(raw) }

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
