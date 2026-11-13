package bindings

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"go-music-tools/internal/config"
	"go-music-tools/internal/httpclient"
	"go-music-tools/internal/logx"
	"go-music-tools/internal/model"
	"go-music-tools/internal/oomusic"
	"go-music-tools/internal/store"
	"go-music-tools/internal/task"
)

type Deps struct {
	Root   string
	Config *config.Store
	Logs   *logx.Hub
	Cache  *store.Store
	Logger *slog.Logger
}

// Service 承载全部 Wails 绑定方法。它不 import Wails runtime：事件发射器与文件对话框
// 由 main 包在 startup 时注入，因此本包可独立单元测试。
type Service struct {
	deps Deps

	mu         sync.RWMutex
	emit       func(string, any)
	dirDialog  func(string) (string, error)
	fileDialog func(string) (string, error)
	saveDialog func(string, string) (string, error)

	tasks *task.Manager

	ooMu  sync.Mutex
	ooSvc *oomusic.Service
}

func NewService(d Deps) (*Service, error) {
	if d.Config == nil || d.Logs == nil || d.Cache == nil || d.Logger == nil {
		return nil, fmt.Errorf("绑定服务缺少必要依赖")
	}
	s := &Service{deps: d, tasks: task.NewManager(d.Logger)}
	s.tasks.SetOnFinish(func(done task.Done) {
		_ = d.Cache.AddTaskHistory(model.TaskRecord{
			Kind: done.Kind, Success: done.Success, Skipped: done.Skipped, Failed: done.Failed,
		})
	})

	d.Config.SetOnChange(func(c config.Config) {
		d.Logs.ApplyConfig(c.Logging)
		s.ooMu.Lock()
		s.ooSvc = nil
		s.ooMu.Unlock()
		s.Emit(task.EventConfigChanged, c)
		d.Logger.Info("配置已更新并生效", "source", "settings")
	})
	return s, nil
}

func (s *Service) SetEmitter(fn func(string, any)) {
	s.mu.Lock()
	s.emit = fn
	s.mu.Unlock()
	s.deps.Logs.SetEmitter(logx.EmitFunc(fn))
	s.tasks.SetEmitter(task.EmitFunc(fn))
}

func (s *Service) SetDirectoryDialog(fn func(string) (string, error)) {
	s.mu.Lock()
	s.dirDialog = fn
	s.mu.Unlock()
}

func (s *Service) SetFileDialog(fn func(string) (string, error)) {
	s.mu.Lock()
	s.fileDialog = fn
	s.mu.Unlock()
}

func (s *Service) SetSaveDialog(fn func(string, string) (string, error)) {
	s.mu.Lock()
	s.saveDialog = fn
	s.mu.Unlock()
}

func (s *Service) Emit(event string, payload any) {
	s.mu.RLock()
	fn := s.emit
	s.mu.RUnlock()
	if fn != nil {
		fn(event, payload)
	}
}

func (s *Service) cfg() config.Config { return s.deps.Config.Get() }

func (s *Service) resolve(path string) string {
	if path == "" {
		return s.deps.Root
	}
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(s.deps.Root, path)
}

func (s *Service) httpClient() *httpclient.Client {
	c := s.cfg()
	return httpclient.New(c.Network, c.API.InsecureSkipVerify, s.deps.Logger)
}

// Start 保留原有导出签名以兼容 Wails 绑定生成；内部提交统一走 start。
func (s *Service) Start(kind string, total int, fn task.TaskFunc) (string, error) {
	return s.start(kind, total, fn)
}

// start 提交任务，可附加 task.WithRetry 等选项。
func (s *Service) start(kind string, total int, fn task.TaskFunc, opts ...task.StartOption) (string, error) {
	return s.tasks.Start(kind, total, fn, opts...)
}

func (s *Service) ConfigPath() string { return s.deps.Config.Path() }

// CancelTask 取消两条通道中全部正在运行的任务（顶栏按钮与退出流程复用）。
func (s *Service) CancelTask() bool { return s.tasks.Cancel() }

// CancelTaskByID 只取消指定任务：运行中发取消信号，排队中直接移出队列。
func (s *Service) CancelTaskByID(taskID string) bool { return s.tasks.CancelByID(taskID) }

// RetryFailedItems 用原请求里失败的子项重建新任务，投递回原通道并返回新的 task_id。
func (s *Service) RetryFailedItems(taskID string) (string, error) {
	return s.tasks.RetryFailedItems(taskID)
}

// GetTaskQueue 返回运行进度、等待队列、通道状态与任务注册表的兼容快照。
func (s *Service) GetTaskQueue() task.QueueSnapshot { return s.tasks.Snapshot() }

// RemoveQueuedTask 把一个尚未开始的任务移出等待队列。
func (s *Service) RemoveQueuedTask(taskID string) bool { return s.tasks.RemoveQueued(taskID) }

// ClearTaskQueue 清空全部等待中的任务，返回清除的数量。
func (s *Service) ClearTaskQueue() int { return s.tasks.ClearQueue() }

func (s *Service) Close() error {
	s.tasks.Cancel()
	return nil
}

func (s *Service) dialogs() (func(string) (string, error), func(string) (string, error), func(string, string) (string, error)) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.dirDialog, s.fileDialog, s.saveDialog
}

// OpenPath 用系统文件管理器打开目录或文件所在位置。
// Windows 上优先复用已显示该目录的资源管理器窗口（见 openpath_windows.go），
// 避免重复点击开出多个相同窗口；Win10 与 Win11（标签页）行为一致。
func (s *Service) OpenPath(path string) error {
	target := s.resolve(path)
	if _, err := os.Stat(target); err != nil {
		return fmt.Errorf("路径不存在: %s", target)
	}
	abs, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	if info, err := os.Stat(abs); err == nil && !info.IsDir() {
		return openFileInFileManager(abs)
	}
	return openInFileManager(abs)
}

// SelectDirectory 由「目录选择」按钮调用；未注入对话框时返回提示错误。
func (s *Service) SelectDirectory(title string) (string, error) {
	dirFn, _, _ := s.dialogs()
	if dirFn == nil {
		return "", fmt.Errorf("文件对话框尚未就绪，请在配置中直接填写路径")
	}
	return dirFn(title)
}

func (s *Service) SelectFile(title string) (string, error) {
	_, fileFn, _ := s.dialogs()
	if fileFn == nil {
		return "", fmt.Errorf("文件对话框尚未就绪")
	}
	return fileFn(title)
}

func (s *Service) requestSavePath(title, defaultName string) (string, error) {
	_, _, saveFn := s.dialogs()
	if saveFn == nil {
		return "", fmt.Errorf("保存对话框尚未就绪")
	}
	return saveFn(title, defaultName)
}
