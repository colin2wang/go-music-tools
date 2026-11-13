package bindings

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"go-music-tools/internal/config"
	"go-music-tools/internal/task"
)

// GetConfigSchema 返回设置页动态表单的字段元数据，前端不需要硬编码字段定义。
func (s *Service) GetConfigSchema() []config.Group {
	return config.Schema(s.cfg())
}

func (s *Service) GetConfig() config.Config { return s.cfg() }

func (s *Service) GetConfigPath() string { return s.deps.Config.Path() }

// ValidateConfig 供设置页在保存前做字段级校验，直接把错误映射展示到对应字段。
func (s *Service) ValidateConfig(c config.Config) map[string]string {
	return c.Validate()
}

func (s *Service) SaveConfig(c config.Config) error {
	if err := s.deps.Config.Save(c); err != nil {
		if ve, ok := err.(*config.ValidationError); ok {
			return fmt.Errorf("配置校验失败: %v", ve.Errors)
		}
		return err
	}
	return nil
}

func (s *Service) ReloadConfig() error {
	if err := s.deps.Config.Reload(); err != nil {
		return err
	}
	s.deps.Logs.ApplyConfig(s.cfg().Logging)
	s.Emit(task.EventConfigChanged, s.cfg())
	s.deps.Logger.Info("配置已从磁盘重新加载")
	return nil
}

func (s *Service) ResetConfig() error { return s.deps.Config.Reset() }

func (s *Service) OpenConfigFile() error {
	path := s.deps.Config.Path()
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", path)
	case "darwin":
		cmd = exec.Command("open", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	return cmd.Start()
}

// ExportConfig 把当前生效配置导出为 YAML 备份，返回目标路径（用户取消时返回空串）。
func (s *Service) ExportConfig() (string, error) {
	path, err := s.requestSavePath("导出配置为 YAML", "config-backup.yml")
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(path) == "" {
		return "", nil
	}
	lower := strings.ToLower(path)
	if !strings.HasSuffix(lower, ".yaml") && !strings.HasSuffix(lower, ".yml") {
		path += ".yaml"
	}
	if err := s.deps.Config.Export(path); err != nil {
		return "", fmt.Errorf("导出配置: %w", err)
	}
	s.deps.Logger.Info("配置已导出", "path", path)
	return path, nil
}

// ImportConfig 从外部 YAML 恢复配置；Store.Save 会校验、原子写并广播 config:changed。
func (s *Service) ImportConfig() (string, error) {
	_, fileFn, _ := s.dialogs()
	if fileFn == nil {
		return "", fmt.Errorf("文件对话框尚未就绪")
	}
	path, err := fileFn("选择要导入的 YAML 配置文件")
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(path) == "" {
		return "", nil
	}
	if err := s.deps.Config.Import(path); err != nil {
		if ve, ok := err.(*config.ValidationError); ok {
			return "", fmt.Errorf("导入的配置不合法: %v", ve.Errors)
		}
		return "", err
	}
	s.deps.Logger.Info("配置已导入并生效", "path", path)
	return path, nil
}
