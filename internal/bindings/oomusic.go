package bindings

import (
	"context"
	"fmt"
	"strings"

	"go-music-tools/internal/model"
	"go-music-tools/internal/oomusic"
	"go-music-tools/internal/task"
)

// ooService 复用同一个实例以保留搜索缓存；配置变更时由 NewService 的回调置空重建。
func (s *Service) ooService() *oomusic.Service {
	s.ooMu.Lock()
	defer s.ooMu.Unlock()
	if s.ooSvc == nil {
		s.ooSvc = oomusic.New(s.cfg(), s.deps.Root, s.httpClient(), s.deps.Logger)
	}
	return s.ooSvc
}

func (s *Service) OOSearch(keyword string) ([]model.OOSong, error) {
	return s.ooService().Search(context.Background(), keyword)
}

func (s *Service) OODetail(keyword, songMid string) (model.OODetail, error) {
	info, err := s.ooService().Detail(context.Background(), keyword, songMid)
	if err != nil {
		return model.OODetail{}, err
	}
	return *info, nil
}

func (s *Service) OOQualities(keyword, songMid string) ([]model.QualityOption, error) {
	return s.ooService().Qualities(context.Background(), keyword, songMid)
}

func (s *Service) OODownload(req OODownloadRequest) (string, error) {
	mid := strings.TrimSpace(req.SongMid)
	if mid == "" {
		return "", fmt.Errorf("请先选择一首歌曲")
	}
	quality := strings.TrimSpace(req.Quality)
	if quality == "" {
		quality = s.cfg().Advanced.OOMusicDefaultQuality
	}
	svc := s.ooService()
	return s.Start("oomusic:single", 1, func(ctx context.Context, r *task.Reporter) error {
		s.deps.Logger.Info("OO Music 单曲下载开始", "song_mid", mid, "quality", quality)
		path, skipped, err := svc.Download(ctx, req.Keyword, mid, quality)
		if err != nil {
			if r != nil {
				r.Item(mid, "failed", err.Error())
			}
			return err
		}
		status := "success"
		if skipped {
			status = "skipped"
		}
		if r != nil {
			r.Item(mid, status, path)
		}
		return nil
	})
}

func (s *Service) OOBatchDownload(keyword, quality string) (string, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return "", fmt.Errorf("请先输入搜索关键词")
	}
	if strings.TrimSpace(quality) == "" {
		quality = s.cfg().Advanced.OOMusicDefaultQuality
	}
	svc := s.ooService()
	return s.Start("oomusic:batch", 0, func(ctx context.Context, r *task.Reporter) error {
		s.deps.Logger.Info("OO Music 批量下载开始", "keyword", keyword, "quality", quality)
		return svc.BatchDownload(ctx, keyword, quality, r)
	})
}
