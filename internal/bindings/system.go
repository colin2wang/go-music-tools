package bindings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"go-music-tools/internal/model"
	"go-music-tools/internal/task"
)

func (s *Service) GetAppInfo() AppInfo {
	c := s.cfg()
	return AppInfo{
		Version:     "1.0.0",
		ConfigPath:  s.deps.Config.Path(),
		LogDir:      s.deps.Logs.LogDir(),
		DownloadDir: s.resolve(c.Download.DefaultDir),
		Platform:    runtime.GOOS + "/" + runtime.GOARCH,
	}
}

func (s *Service) GetTaskStatus() task.Progress { return s.tasks.Status() }

func (s *Service) GetTaskHistory(limit int) ([]model.TaskRecord, error) {
	return s.deps.Cache.ListTaskHistory(limit)
}

func (s *Service) OpenDownloadDir() error {
	dir := s.resolve(s.cfg().Download.DefaultDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return s.OpenPath(dir)
}

func (s *Service) GetLastDownloadConfig() (LastDownloadConfig, error) {
	fallback := LastDownloadConfig{Kind: "song", Quality: s.cfg().Download.DefaultQuality, Indexes: []int{}}
	raw, err := s.deps.Cache.GetLastDownloadConfig()
	if err != nil || strings.TrimSpace(raw) == "" {
		return fallback, nil
	}
	var out LastDownloadConfig
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return fallback, nil
	}
	if out.Quality == "" {
		out.Quality = fallback.Quality
	}
	if out.Kind == "" {
		out.Kind = fallback.Kind
	}
	if out.Indexes == nil {
		out.Indexes = []int{}
	}
	return out, nil
}

func (s *Service) SaveLastDownloadConfig(c LastDownloadConfig) error {
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return s.deps.Cache.SetLastDownloadConfig(string(data))
}

// ListSubFolders 返回目录的一级子目录，供音乐库工具预览扫描范围。
func (s *Service) ListSubFolders(dir string) ([]string, error) {
	target := s.resolve(dir)
	entries, err := os.ReadDir(target)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			out = append(out, filepath.Join(target, e.Name()))
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i]) < strings.ToLower(out[j])
	})
	return out, nil
}
