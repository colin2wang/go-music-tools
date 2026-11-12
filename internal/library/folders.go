package library

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"go-music-tools/internal/model"
	"go-music-tools/internal/task"
)

type FixKind string

const (
	FixLyrics   FixKind = "lyrics"
	FixCover    FixKind = "cover"
	FixArtists  FixKind = "artists"
	FixMetadata FixKind = "metadata"
)

type FixRequest struct {
	Directory       string
	Kind            FixKind
	OnlyMissing     bool
	Force           bool
	Recursive       bool
	ContinueOnError bool
}

type Options struct {
	ContinueOnError bool
	Reporter        *task.Reporter
}

// ListFolders 返回一级子目录，跳过隐藏目录并按名称大小写无关排序。
func ListFolders(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			out = append(out, filepath.Join(dir, entry.Name()))
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(filepath.Base(out[i])) < strings.ToLower(filepath.Base(out[j]))
	})
	return out, nil
}

// ProcessFolders 逐目录执行修补函数，记录耗时与结果；出错默认不中断整批。
func ProcessFolders(ctx context.Context, dir string,
	fn func(context.Context, string) (string, error), opts Options) ([]model.FolderResult, error) {

	folders, err := ListFolders(dir)
	if err != nil {
		return nil, err
	}
	if opts.Reporter != nil {
		opts.Reporter.SetTotal(len(folders))
	}
	results := make([]model.FolderResult, 0, len(folders))
	for _, folder := range folders {
		if err := ctx.Err(); err != nil {
			return results, err
		}
		started := time.Now()
		if opts.Reporter != nil {
			opts.Reporter.Stage("处理目录：" + filepath.Base(folder))
		}
		message, runErr := fn(ctx, folder)
		result := model.FolderResult{
			Folder: folder, Success: runErr == nil,
			ElapsedMS: time.Since(started).Milliseconds(),
		}
		if runErr != nil {
			result.Message = runErr.Error()
		} else {
			result.Message = message
		}
		results = append(results, result)

		if opts.Reporter != nil {
			status := "success"
			if runErr != nil {
				status = "failed"
			}
			opts.Reporter.ItemWithElapsed(filepath.Base(folder), status, result.Message, result.ElapsedMS)
		}
		if runErr != nil && !opts.ContinueOnError {
			return results, runErr
		}
	}
	return results, nil
}
