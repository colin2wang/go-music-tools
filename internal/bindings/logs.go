package bindings

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go-music-tools/internal/logx"
)

// GetRecentLogs 供全局日志窗口挂载后回放启动早期的日志。
func (s *Service) GetRecentLogs() []logx.LogEntry {
	return s.deps.Logs.Recent()
}

func (s *Service) ClearLogs() {
	s.deps.Logs.Clear()
	s.deps.Logger.Info("日志窗口已清空")
}

func (s *Service) ExportLogs() (string, error) {
	path, err := s.requestSavePath("导出日志", fmt.Sprintf("music-tools-%s.log", time.Now().Format("20060102-150405")))
	if err != nil || path == "" {
		if err == nil {
			return "", nil
		}
		// 对话框不可用时退回到日志目录，保证功能仍可用。
		path = filepath.Join(s.deps.Logs.LogDir(), fmt.Sprintf("export-%s.log", time.Now().Format("20060102-150405")))
	}
	if err := s.deps.Logs.Export(path); err != nil {
		return "", err
	}
	s.deps.Logger.Info("日志已导出", "file", filepath.Base(path))
	return path, nil
}

func (s *Service) OpenLogDir() error {
	dir := s.deps.Logs.LogDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return s.OpenPath(dir)
}
