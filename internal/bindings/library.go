package bindings

import (
	"context"
	"fmt"
	"os"
	"strings"

	"go-music-tools/internal/library"
	"go-music-tools/internal/task"
)

var supportedFixKinds = map[string]library.FixKind{
	"lyrics": library.FixLyrics, "cover": library.FixCover,
	"artists": library.FixArtists, "metadata": library.FixMetadata,
}

// RunLibraryFix 对所选目录的一级子目录批量执行修补流程。
func (s *Service) RunLibraryFix(req LibraryFixRequest) (string, error) {
	kind, ok := supportedFixKinds[strings.ToLower(strings.TrimSpace(req.Kind))]
	if !ok {
		return "", fmt.Errorf("未知的修补类型: %q", req.Kind)
	}
	directory := s.resolve(strings.TrimSpace(req.Directory))
	stat, err := os.Stat(directory)
	if err != nil {
		return "", fmt.Errorf("目录不存在或不可访问: %s", directory)
	}
	if !stat.IsDir() {
		return "", fmt.Errorf("请选择目录而不是文件: %s", directory)
	}

	fixReq := library.FixRequest{
		Directory: directory, Kind: kind,
		OnlyMissing: req.OnlyMissing, Force: req.Force,
		Recursive: req.Recursive, ContinueOnError: req.ContinueOnError,
	}
	svc := library.New(s.cfg(), s.deps.Root, s.httpClient(), s.deps.Logger, s.deps.DB)

	return s.Start("library:"+string(kind), 0, func(ctx context.Context, r *task.Reporter) error {
		s.deps.Logger.Info("批量修补任务开始", "kind", kind, "directory", directory)
		results, err := svc.Run(ctx, fixReq, r)
		success := 0
		for _, result := range results {
			if result.Success {
				success++
			}
		}
		s.deps.Logger.Info("批量修补任务结束", "kind", kind,
			"folders", len(results), "success", success, "failed", len(results)-success)
		return err
	})
}
