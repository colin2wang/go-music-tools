package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

// ValidationError 会被绑定层展开成字段级错误返回给设置页。
type ValidationError struct {
	Errors map[string]string `json:"errors"`
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("配置校验失败，共 %d 项", len(e.Errors))
}

// Store 以「默认值合并 → 校验 → 临时文件 → 原子 rename → 广播」的流程管理 config.yml。
type Store struct {
	mu       sync.RWMutex
	path     string
	cfg      Config
	onChange func(Config)
}

func NewStore(path string) *Store {
	return &Store{path: path, cfg: Default()}
}

func (s *Store) Path() string { return s.path }

func (s *Store) Get() Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

func (s *Store) SetOnChange(fn func(Config)) {
	s.mu.Lock()
	s.onChange = fn
	s.mu.Unlock()
}

func (s *Store) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		c := Default()
		if err := s.writeLocked(c); err != nil {
			return fmt.Errorf("生成默认配置: %w", err)
		}
		s.cfg = c
		return nil
	}
	if err != nil {
		return fmt.Errorf("读取配置文件: %w", err)
	}

	c := Default()
	if err := yaml.Unmarshal(data, &c); err != nil {
		return fmt.Errorf("解析 YAML: %w", err)
	}
	if errs := c.Validate(); len(errs) > 0 {
		return &ValidationError{Errors: errs}
	}
	s.cfg = c
	return nil
}

func (s *Store) Reload() error { return s.Load() }

func (s *Store) Reset() error {
	return s.Save(Default())
}

func (s *Store) Save(c Config) error {
	if errs := c.Validate(); len(errs) > 0 {
		return &ValidationError{Errors: errs}
	}

	s.mu.Lock()
	if err := s.writeLocked(c); err != nil {
		s.mu.Unlock()
		return fmt.Errorf("写入配置: %w", err)
	}
	s.cfg = c
	cb := s.onChange
	s.mu.Unlock()

	if cb != nil {
		cb(c)
	}
	return nil
}

// Export 把当前生效配置序列化到指定文件，同样使用临时文件 + rename 避免半文件。
func (s *Store) Export(path string) error {
	s.mu.RLock()
	c := s.cfg
	s.mu.RUnlock()

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("序列化配置: %w", err)
	}
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	header := "# Music Tools 配置导出\n# 可通过「设置」页的导入按钮恢复。\n\n"
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append([]byte(header), data...), 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// Import 读取外部 YAML，按默认值合并并校验后保存，从而复用 Save 的广播流程。
func (s *Store) Import(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取导入文件: %w", err)
	}
	c := Default()
	if err := yaml.Unmarshal(data, &c); err != nil {
		return fmt.Errorf("解析导入的 YAML: %w", err)
	}
	return s.Save(c)
}

// writeLocked 必须在外层持有写锁时调用。
func (s *Store) writeLocked(c Config) error {
	if dir := filepath.Dir(s.path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	header := "# Music Tools 配置，由应用自动生成与维护\n# 可在界面「设置」页修改，也可手工编辑后点击「重新加载」。\n\n"
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, append([]byte(header), data...), 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, s.path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
