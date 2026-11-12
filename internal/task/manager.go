package task

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
)

var ErrBusy = errors.New("已有任务正在运行，请先等待完成或取消当前任务")

type EmitFunc func(event string, payload any)
type TaskFunc func(ctx context.Context, r *Reporter) error

type running struct {
	id       string
	kind     string
	cancel   context.CancelFunc
	progress Progress
	items    []ItemResult
	finished bool
}

// Manager 保证同一时刻只有一个主任务，并把取消信号贯穿到所有业务循环。
type Manager struct {
	mu       sync.Mutex
	emit     EmitFunc
	logger   *slog.Logger
	current  *running
	last     Progress
	lastID   string
	onFinish func(Done)
}

func NewManager(logger *slog.Logger) *Manager {
	return &Manager{logger: logger}
}

func (m *Manager) SetEmitter(fn EmitFunc) {
	m.mu.Lock()
	m.emit = fn
	m.mu.Unlock()
}

// SetOnFinish 让绑定层把任务结果落到历史记录，无需轮询状态。
func (m *Manager) SetOnFinish(fn func(Done)) {
	m.mu.Lock()
	m.onFinish = fn
	m.mu.Unlock()
}

func (m *Manager) Start(kind string, total int, fn TaskFunc) (string, error) {
	m.mu.Lock()
	if m.current != nil && !m.current.finished {
		m.mu.Unlock()
		return "", ErrBusy
	}
	id := uuid.NewString()
	ctx, cancel := context.WithCancel(context.Background())
	r := &running{
		id:     id,
		kind:   kind,
		cancel: cancel,
		progress: Progress{
			TaskID: id, Kind: kind, Stage: "准备中", Total: total,
		},
		items: make([]ItemResult, 0, total),
	}
	m.current = r
	m.lastID = id
	emit := m.emit
	m.mu.Unlock()

	if emit != nil {
		emit(EventTaskProgress, r.progress)
	}

	go m.execute(r, ctx, fn)
	return id, nil
}

func (m *Manager) execute(r *running, ctx context.Context, fn TaskFunc) {
	started := time.Now()
	var runErr error
	defer func() {
		if recovered := recover(); recovered != nil {
			runErr = fmt.Errorf("任务内部异常: %v", recovered)
			if m.logger != nil {
				m.logger.Error("任务异常", "task", r.id, "error", runErr)
			}
		}
		m.finish(r, runErr, time.Since(started).Milliseconds())
	}()

	reporter := &Reporter{manager: m, ctx: ctx, taskID: r.id}
	reporter.Stage("运行中")
	runErr = fn(ctx, reporter)
	if errors.Is(ctx.Err(), context.Canceled) && runErr == nil {
		runErr = context.Canceled
	}
}

func (m *Manager) finish(r *running, runErr error, elapsed int64) {
	m.mu.Lock()
	r.finished = true
	r.progress.Done = true
	cancelled := errors.Is(runErr, context.Canceled) || r.progress.Cancelled
	r.progress.Cancelled = cancelled
	if cancelled {
		r.progress.Stage = "已取消"
	} else if runErr != nil {
		r.progress.Stage = "失败"
		r.progress.Error = runErr.Error()
	} else {
		r.progress.Stage = "已完成"
	}
	m.last = r.progress

	done := Done{
		TaskID:    r.id,
		Kind:      r.kind,
		Success:   r.progress.Success,
		Skipped:   r.progress.Skipped,
		Failed:    r.progress.Failed,
		Cancelled: cancelled,
		Items:     append([]ItemResult(nil), r.items...),
		ElapsedMS: elapsed,
	}
	if runErr != nil && !cancelled {
		done.Error = runErr.Error()
	}
	emit := m.emit
	onFinish := m.onFinish
	progress := r.progress
	m.mu.Unlock()

	if emit != nil {
		emit(EventTaskProgress, progress)
		emit(EventTaskDone, done)
	}
	if onFinish != nil {
		onFinish(done)
	}
	if m.logger != nil {
		m.logger.Info("任务结束", "task", r.id, "kind", r.kind,
			"success", done.Success, "skipped", done.Skipped, "failed", done.Failed,
			"elapsed_ms", elapsed, "cancelled", cancelled)
	}
}

// Cancel 只发出取消信号，实际退出由业务循环检查 ctx.Done() 完成。
func (m *Manager) Cancel() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.current == nil || m.current.finished {
		return false
	}
	m.current.progress.Cancelled = true
	m.current.progress.Stage = "正在取消"
	m.current.cancel()
	if m.emit != nil {
		m.emit(EventTaskProgress, m.current.progress)
	}
	return true
}

func (m *Manager) Status() Progress {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.current != nil && !m.current.finished {
		return m.current.progress
	}
	return m.last
}

// Reporter 由单个任务协程使用，所有状态更新都经过 Manager 的锁。
type Reporter struct {
	manager *Manager
	ctx     context.Context
	taskID  string
}

func (r *Reporter) Context() context.Context { return r.ctx }
func (r *Reporter) Err() error               { return r.ctx.Err() }

func (r *Reporter) Stage(stage string) {
	r.manager.update(r.taskID, func(p *Progress) { p.Stage = stage })
}

func (r *Reporter) SetTotal(total int) {
	r.manager.update(r.taskID, func(p *Progress) { p.Total = total })
}

func (r *Reporter) Item(name, status, message string) {
	r.ItemWithElapsed(name, status, message, 0)
}

// ItemWithElapsed 记录单项耗时，并通过 task:item 实时推送，供逐目录进度界面使用。
func (r *Reporter) ItemWithElapsed(name, status, message string, elapsedMS int64) {
	r.manager.update(r.taskID, func(p *Progress) {
		p.CurrentItem = name
		p.Current++
		switch status {
		case "success":
			p.Success++
		case "skipped":
			p.Skipped++
		case "failed":
			p.Failed++
		}
	})
	item := ItemResult{Name: name, Status: status, Message: message, ElapsedMS: elapsedMS}
	m := r.manager
	m.mu.Lock()
	if m.current != nil && m.current.id == r.taskID {
		m.current.items = append(m.current.items, item)
	}
	emit := m.emit
	m.mu.Unlock()
	if emit != nil {
		emit(EventTaskItem, ItemEvent{TaskID: r.taskID, Item: item})
	}
}

func (m *Manager) update(taskID string, mutate func(*Progress)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.current == nil || m.current.id != taskID {
		return
	}
	mutate(&m.current.progress)
	if m.emit != nil {
		m.emit(EventTaskProgress, m.current.progress)
	}
}
