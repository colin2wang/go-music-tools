package library

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"

	"go-music-tools/internal/config"
	"go-music-tools/internal/httpclient"
	"go-music-tools/internal/model"
	"go-music-tools/internal/music"
	"go-music-tools/internal/task"
)

// Service 组合目录调度与四条修补流程，复用主下载服务的 API 与元数据能力。
type Service struct {
	Cfg    config.Config
	Root   string
	Music  *music.Service
	Logger *slog.Logger
}

func New(cfg config.Config, root string, hc *httpclient.Client, logger *slog.Logger, store music.Store) *Service {
	return &Service{
		Cfg: cfg, Root: root, Logger: logger,
		Music: music.New(cfg, root, hc, logger, store),
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

func (s *Service) Run(ctx context.Context, req FixRequest, r *task.Reporter) ([]model.FolderResult, error) {
	dir := s.resolve(req.Directory)
	if stat, err := filepath.Abs(dir); err == nil {
		dir = stat
	}
	var fn func(context.Context, string) (string, error)
	switch req.Kind {
	case FixLyrics:
		fn = func(ctx context.Context, folder string) (string, error) {
			return s.FixLyrics(ctx, folder, req.Recursive)
		}
	case FixCover:
		fn = func(ctx context.Context, folder string) (string, error) {
			return s.FixCover(ctx, folder, req.Force)
		}
	case FixArtists:
		fn = func(ctx context.Context, folder string) (string, error) {
			return s.FixArtists(ctx, folder, req.Recursive)
		}
	case FixMetadata:
		fn = func(ctx context.Context, folder string) (string, error) {
			return s.FixMetadata(ctx, folder, req.OnlyMissing, req.Recursive)
		}
	default:
		return nil, fmt.Errorf("未知的修补类型: %q", req.Kind)
	}

	if s.Logger != nil {
		s.Logger.Info("开始批量修补", "kind", req.Kind, "directory", dir)
	}
	return ProcessFolders(ctx, dir, fn, Options{
		ContinueOnError: req.ContinueOnError, Reporter: r,
	})
}
