package task

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type EmitFunc func(event string, payload any)
type TaskFunc func(ctx context.Context, r *Reporter) error

const (
	maxTaskEntries   = 50  // 注册表最多保留的任务数（运行中/排队中永不淘汰）
	maxEntryItems    = 200 // 单个任务在注册表里保留的子项数上限
	boardItemPreview = 40  // 快照里每个任务携带的最近子项数
	retryKindSuffix  = ":retry"
)

type running struct {
	id       string
	kind     string
	lane     Lane
	ctx      context.Context
	cancel   context.CancelFunc
	started  time.Time
	progress Progress
	finished bool
}

// queuedEntry 是等待队列中的一个待执行任务；Start 返回的 id 即 TaskID。
type queuedEntry struct {
	queued QueuedTask
	fn     TaskFunc
	retry  RetryFactory
}

type laneRuntime struct {
	lane    Lane
	current *running
	queue   []queuedEntry
}

// Manager 按通道维护互不阻塞的 FIFO 队列：每条通道同一时刻只有一个任务运行，
// 通道之间可以并行；注册表以 task_id 为键追踪每个独立任务的完整生命周期。
type Manager struct {
	mu       sync.Mutex
	emit     EmitFunc
	logger   *slog.Logger
	onFinish func(Done)
	lanes    map[Lane]*laneRuntime
	entries  map[string]*TaskEntry
	retries  map[string]RetryFactory
	order    []string // 提交顺序，用于淘汰与快照排序
	last     Progress // 最近结束任务的进度，供 Status() 兼容派生
	lastID   string
	notify   chan struct{}
	closed   chan struct{}
}

var laneOrder = []Lane{LaneWorkbench, LaneOOMusic}

func NewManager(logger *slog.Logger) *Manager {
	m := &Manager{
		logger: logger,
		lanes: map[Lane]*laneRuntime{
			LaneWorkbench: &laneRuntime{lane: LaneWorkbench},
			LaneOOMusic:   &laneRuntime{lane: LaneOOMusic},
		},
		entries: make(map[string]*TaskEntry),
		retries: make(map[string]RetryFactory),
		notify:  make(chan struct{}, 1),
		closed:  make(chan struct{}),
	}
	go m.dispatchLoop()
	return m
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

func (m *Manager) Close() {
	select {
	case <-m.closed:
	default:
		close(m.closed)
	}
}

// Start 提交任务并立即返回 task_id：所属通道空闲则直接执行，否则进入该通道的等待队列。
// 可选 WithRetry 注入失败子项重试工厂，签名对既有调用方完全兼容。
func (m *Manager) Start(kind string, total int, fn TaskFunc, opts ...StartOption) (string, error) {
	var options startOptions
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}
	lane := LaneFor(kind)
	id := uuid.NewString()
	now := time.Now().Unix()

	m.mu.Lock()
	lr := m.lanes[lane]
	entry := &TaskEntry{
		TaskID: id, Kind: kind, Lane: lane,
		Status: TaskStatusQueued, Stage: "排队中",
		EnqueuedAt: now, Total: total,
		CanRetry: options.retry != nil,
		Items:    make([]ItemResult, 0),
	}
	m.putEntryLocked(id, entry, options.retry)
	qe := queuedEntry{
		queued: QueuedTask{
			TaskID: id, Kind: kind, Lane: lane,
			Status: TaskStatusQueued, Total: total, EnqueuedAt: now,
		},
		fn: fn, retry: options.retry,
	}

	if lr.current != nil && !lr.current.finished {
		lr.queue = append(lr.queue, qe)
		emit := m.emit
		snap := m.snapshotLocked()
		m.mu.Unlock()
		if emit != nil {
			emit(EventTaskQueued, snap)
		}
		return id, nil
	}

	r := m.startLocked(lr, qe)
	m.lastID = id
	p := r.progress
	emit := m.emit
	snap := m.snapshotLocked()
	m.mu.Unlock()
	if emit != nil {
		emit(EventTaskProgress, p)
		emit(EventTaskQueued, snap)
	}
	go m.execute(r, fn)
	return id, nil
}

// startLocked 在持锁状态下把一个排队项转为运行中，并同步注册表。
func (m *Manager) startLocked(lr *laneRuntime, qe queuedEntry) *running {
	ctx, cancel := context.WithCancel(context.Background())
	now := time.Now()
	r := &running{
		id: qe.queued.TaskID, kind: qe.queued.Kind, lane: lr.lane,
		ctx: ctx, cancel: cancel, started: now,
		progress: Progress{
			TaskID: qe.queued.TaskID, Kind: qe.queued.Kind, Lane: lr.lane,
			Stage: "准备中", Status: "running", Total: qe.queued.Total,
			EnqueuedAt: qe.queued.EnqueuedAt, StartedAt: now.Unix(),
			CanRetry: qe.retry != nil,
		},
	}
	lr.current = r
	if e := m.entries[r.id]; e != nil {
		e.Status = TaskStatusRunning
		e.StartedAt = r.progress.StartedAt
		e.Total = qe.queued.Total
		syncEntryFromProgress(e, r.progress)
	}
	return r
}

// dispatchLoop 在任一任务结束后为每条空闲通道派发一个队头任务。
func (m *Manager) dispatchLoop() {
	for {
		select {
		case <-m.closed:
			return
		case <-m.notify:
		}
		m.dispatch()
	}
}

type pendingStart struct {
	r  *running
	fn TaskFunc
	p  Progress
}

func (m *Manager) dispatch() {
	m.mu.Lock()
	var starts []pendingStart
	for _, lane := range laneOrder {
		lr := m.lanes[lane]
		if lr == nil || (lr.current != nil && !lr.current.finished) || len(lr.queue) == 0 {
			continue
		}
		qe := lr.queue[0]
		lr.queue = lr.queue[1:]
		r := m.startLocked(lr, qe)
		m.lastID = r.id
		starts = append(starts, pendingStart{r: r, fn: qe.fn, p: r.progress})
	}
	var snap QueueSnapshot
	if len(starts) > 0 {
		snap = m.snapshotLocked()
	}
	emit := m.emit
	m.mu.Unlock()

	if emit != nil {
		for _, s := range starts {
			emit(EventTaskProgress, s.p)
		}
		if len(starts) > 0 {
			emit(EventTaskQueued, snap)
		}
	}
	for _, s := range starts {
		go m.execute(s.r, s.fn)
	}
}

func (m *Manager) execute(r *running, fn TaskFunc) {
	started := time.Now()
	var runErr error
	defer func() {
		if recovered := recover(); recovered != nil {
			runErr = fmt.Errorf("任务内部异常: %v", recovered)
			if m.logger != nil {
				m.logger.Error("任务异常", "task", r.id, "lane", r.lane, "error", runErr)
			}
		}
		m.finish(r, runErr, time.Since(started).Milliseconds())
	}()

	reporter := &Reporter{manager: m, ctx: r.ctx, taskID: r.id}
	reporter.Stage("运行中")
	runErr = fn(reporter.ctx, reporter)
	if errors.Is(reporter.ctx.Err(), context.Canceled) && runErr == nil {
		runErr = context.Canceled
	}
}

func (m *Manager) finish(r *running, runErr error, elapsed int64) {
	m.mu.Lock()
	r.finished = true
	r.progress.Done = true
	r.progress.Status = "done"
	r.progress.ElapsedMS = elapsed
	cancelled := errors.Is(runErr, context.Canceled) || r.progress.Cancelled
	r.progress.Cancelled = cancelled
	status := TaskStatusDone
	switch {
	case cancelled:
		r.progress.Stage = "已取消"
		status = TaskStatusCancelled
	case runErr != nil:
		r.progress.Stage = "失败"
		r.progress.Error = runErr.Error()
		status = TaskStatusFailed
	default:
		r.progress.Stage = "已完成"
	}
	finishedAt := time.Now().Unix()
	m.last = r.progress
	m.lastID = r.id

	var items []ItemResult
	if e := m.entries[r.id]; e != nil {
		e.Status = status
		e.Stage = r.progress.Stage
		e.FinishedAt = finishedAt
		e.ElapsedMS = elapsed
		e.Cancelled = cancelled
		e.Error = r.progress.Error
		syncEntryFromProgress(e, r.progress)
		items = append([]ItemResult(nil), e.Items...)
	}

	done := Done{
		TaskID: r.id, Kind: r.kind, Lane: r.lane, Status: status,
		Success: r.progress.Success, Skipped: r.progress.Skipped, Failed: r.progress.Failed,
		Cancelled: cancelled, Items: items, ElapsedMS: elapsed,
		StartedAt: r.progress.StartedAt, FinishedAt: finishedAt,
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
		emit(EventTaskQueued, m.Snapshot())
	}
	if onFinish != nil {
		onFinish(done)
	}
	if m.logger != nil {
		m.logger.Info("任务结束", "task", r.id, "kind", r.kind, "lane", r.lane,
			"status", status, "success", done.Success, "skipped", done.Skipped,
			"failed", done.Failed, "elapsed_ms", elapsed, "cancelled", cancelled)
	}
	m.kickDispatch()
}

func (m *Manager) kickDispatch() {
	select {
	case m.notify <- struct{}{}:
	default:
	}
}

// Cancel 取消全部通道中正在运行的任务（顶栏按钮与退出流程复用）。
func (m *Manager) Cancel() bool {
	m.mu.Lock()
	var progresses []Progress
	for _, lane := range laneOrder {
		lr := m.lanes[lane]
		if lr == nil || lr.current == nil || lr.current.finished {
			continue
		}
		lr.current.progress.Cancelled = true
		lr.current.progress.Stage = "正在取消"
		lr.current.cancel()
		if e := m.entries[lr.current.id]; e != nil {
			e.Stage = "正在取消"
		}
		progresses = append(progresses, lr.current.progress)
	}
	emit := m.emit
	m.mu.Unlock()
	if emit != nil {
		for _, p := range progresses {
			emit(EventTaskProgress, p)
		}
	}
	return len(progresses) > 0
}

// CancelByID 只取消指定任务：运行中发取消信号，排队中则直接移出队列。
func (m *Manager) CancelByID(taskID string) bool {
	m.mu.Lock()
	var progress *Progress
	var snap QueueSnapshot
	removed := false
	for _, lane := range laneOrder {
		lr := m.lanes[lane]
		if lr == nil || lr.current == nil || lr.current.finished || lr.current.id != taskID {
			continue
		}
		lr.current.progress.Cancelled = true
		lr.current.progress.Stage = "正在取消"
		lr.current.cancel()
		if e := m.entries[taskID]; e != nil {
			e.Stage = "正在取消"
		}
		p := lr.current.progress
		progress = &p
		break
	}
	if progress == nil {
		for _, lane := range laneOrder {
			lr := m.lanes[lane]
			if lr == nil {
				continue
			}
			for i, qe := range lr.queue {
				if qe.queued.TaskID != taskID {
					continue
				}
				lr.queue = append(lr.queue[:i], lr.queue[i+1:]...)
				m.markRemovedLocked(taskID)
				removed = true
				break
			}
			if removed {
				break
			}
		}
		if removed {
			snap = m.snapshotLocked()
		}
	}
	emit := m.emit
	m.mu.Unlock()
	if emit != nil {
		if progress != nil {
			emit(EventTaskProgress, *progress)
		}
		if removed {
			emit(EventTaskQueued, snap)
		}
	}
	return progress != nil || removed
}

// RemoveQueued 把一个尚未开始的任务移出所属通道的等待队列。
func (m *Manager) RemoveQueued(taskID string) bool {
	m.mu.Lock()
	removed := false
	for _, lane := range laneOrder {
		lr := m.lanes[lane]
		if lr == nil {
			continue
		}
		for i, qe := range lr.queue {
			if qe.queued.TaskID != taskID {
				continue
			}
			lr.queue = append(lr.queue[:i], lr.queue[i+1:]...)
			m.markRemovedLocked(taskID)
			removed = true
			break
		}
		if removed {
			break
		}
	}
	var snap QueueSnapshot
	if removed {
		snap = m.snapshotLocked()
	}
	emit := m.emit
	m.mu.Unlock()
	if removed && emit != nil {
		emit(EventTaskQueued, snap)
	}
	return removed
}

// ClearQueue 清空全部通道中等待的任务，返回清除的数量。
func (m *Manager) ClearQueue() int {
	m.mu.Lock()
	n := 0
	for _, lane := range laneOrder {
		lr := m.lanes[lane]
		if lr == nil {
			continue
		}
		for _, qe := range lr.queue {
			m.markRemovedLocked(qe.queued.TaskID)
			n++
		}
		lr.queue = nil
	}
	var snap QueueSnapshot
	if n > 0 {
		snap = m.snapshotLocked()
	}
	emit := m.emit
	m.mu.Unlock()
	if n > 0 && emit != nil {
		emit(EventTaskQueued, snap)
	}
	return n
}

// RetryFailedItems 用注册表中该任务的失败子项重建一个新任务，投递回原通道。
// 需要提交时通过 WithRetry 注入工厂；不支持的类型或仍在执行中的任务返回错误。
func (m *Manager) RetryFailedItems(taskID string) (string, error) {
	m.mu.Lock()
	e := m.entries[taskID]
	factory := m.retries[taskID]
	if e == nil || factory == nil {
		m.mu.Unlock()
		return "", fmt.Errorf("该任务类型不支持失败子项重试")
	}
	if e.Status == TaskStatusQueued || e.Status == TaskStatusRunning {
		m.mu.Unlock()
		return "", fmt.Errorf("任务尚未结束，无法重试失败项")
	}
	failed := make([]ItemResult, 0)
	for _, item := range e.Items {
		if item.Status == "failed" {
			failed = append(failed, item)
		}
	}
	kind, lane := e.Kind, e.Lane
	m.mu.Unlock()

	if len(failed) == 0 {
		return "", fmt.Errorf("该任务没有失败的子项")
	}
	fn, err := factory(failed)
	if err != nil {
		return "", err
	}
	if fn == nil {
		return "", fmt.Errorf("重试任务构建失败")
	}
	// 保证通道路由与最大重试深度都不随重试次数漂移。
	newKind := strings.TrimSuffix(kind, retryKindSuffix) + retryKindSuffix
	if LaneFor(newKind) != lane {
		return "", fmt.Errorf("重试任务通道不匹配")
	}
	if m.logger != nil {
		m.logger.Info("重试失败子项", "source_task", taskID, "kind", newKind,
			"lane", lane, "count", len(failed))
	}
	return m.Start(newKind, len(failed), fn, WithRetry(factory))
}

func (m *Manager) Queued() []QueuedTask {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.queuedLocked()
}

// Snapshot 组合运行进度、等待队列、通道状态与注册表，供事件与查询绑定使用。
func (m *Manager) Snapshot() QueueSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.snapshotLocked()
}

// Board 返回前端启动时一次性拉取的全量任务面板快照。
func (m *Manager) Board() TaskBoard {
	m.mu.Lock()
	defer m.mu.Unlock()
	return TaskBoard{Lanes: m.laneStatesLocked(), Tasks: m.snapshotEntriesLocked()}
}

// Status 是兼容视图：优先返回任一运行中的进度，否则返回最近结束任务的进度。
func (m *Manager) Status() Progress {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, lane := range laneOrder {
		lr := m.lanes[lane]
		if lr != nil && lr.current != nil && !lr.current.finished {
			return lr.current.progress
		}
	}
	return m.last
}

// Reporter 由单个任务协程使用，所有状态更新都经过 Manager 的锁并按 task_id 隔离。
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
	r.manager.recordItem(r.taskID, ItemResult{
		Name: name, Status: status, Message: message, ElapsedMS: elapsedMS,
	})
}

func (m *Manager) update(taskID string, mutate func(*Progress)) {
	m.mu.Lock()
	r := m.runningByIDLocked(taskID)
	if r == nil {
		m.mu.Unlock()
		return
	}
	mutate(&r.progress)
	r.progress.ElapsedMS = time.Since(r.started).Milliseconds()
	if e := m.entries[taskID]; e != nil {
		syncEntryFromProgress(e, r.progress)
		e.ElapsedMS = r.progress.ElapsedMS
	}
	p := r.progress
	emit := m.emit
	m.mu.Unlock()
	if emit != nil {
		emit(EventTaskProgress, p)
	}
}

func (m *Manager) recordItem(taskID string, item ItemResult) {
	m.mu.Lock()
	e := m.entries[taskID]
	if e != nil {
		e.Items = append(e.Items, item)
		if len(e.Items) > maxEntryItems {
			e.Items = append([]ItemResult(nil), e.Items[len(e.Items)-maxEntryItems:]...)
		}
	}
	ev := ItemEvent{TaskID: taskID, Item: item}
	if e != nil {
		ev.Lane = e.Lane
	}
	emit := m.emit
	m.mu.Unlock()
	if emit != nil {
		emit(EventTaskItem, ev)
	}
}

func (m *Manager) runningByIDLocked(taskID string) *running {
	for _, lane := range laneOrder {
		lr := m.lanes[lane]
		if lr != nil && lr.current != nil && !lr.current.finished && lr.current.id == taskID {
			return lr.current
		}
	}
	return nil
}

func (m *Manager) putEntryLocked(id string, e *TaskEntry, retry RetryFactory) {
	m.entries[id] = e
	m.order = append(m.order, id)
	if retry != nil {
		m.retries[id] = retry
	}
	for len(m.order) > maxTaskEntries {
		evicted := false
		for i, oldID := range m.order {
			old := m.entries[oldID]
			if old == nil || old.Status == TaskStatusQueued || old.Status == TaskStatusRunning {
				continue
			}
			m.order = append(m.order[:i], m.order[i+1:]...)
			delete(m.entries, oldID)
			delete(m.retries, oldID)
			evicted = true
			break
		}
		if !evicted {
			break
		}
	}
}

func (m *Manager) markRemovedLocked(taskID string) {
	if e := m.entries[taskID]; e != nil {
		e.Status = TaskStatusRemoved
		e.Stage = "已移除队列"
		e.FinishedAt = time.Now().Unix()
	}
}

func (m *Manager) queuedLocked() []QueuedTask {
	out := make([]QueuedTask, 0)
	for _, lane := range laneOrder {
		lr := m.lanes[lane]
		if lr == nil {
			continue
		}
		for _, qe := range lr.queue {
			out = append(out, qe.queued)
		}
	}
	return out
}

func (m *Manager) laneStatesLocked() []LaneState {
	out := make([]LaneState, 0, len(laneOrder))
	for _, lane := range laneOrder {
		state := LaneState{Lane: lane}
		if lr := m.lanes[lane]; lr != nil {
			if lr.current != nil && !lr.current.finished {
				state.Running = lr.current.id
			}
			state.Queued = len(lr.queue)
		}
		out = append(out, state)
	}
	return out
}

// snapshotEntriesLocked 按提交时间倒序输出注册表，子项只带最近若干条以控制事件体积。
func (m *Manager) snapshotEntriesLocked() []TaskEntry {
	out := make([]TaskEntry, 0, len(m.order))
	for i := len(m.order) - 1; i >= 0; i-- {
		e := m.entries[m.order[i]]
		if e == nil {
			continue
		}
		copyEntry := *e
		copyEntry.ItemCount = len(e.Items)
		from := len(e.Items) - boardItemPreview
		if from < 0 {
			from = 0
		}
		copyEntry.Items = make([]ItemResult, 0, len(e.Items[from:]))
		copyEntry.Items = append(copyEntry.Items, e.Items[from:]...)
		out = append(out, copyEntry)
	}
	return out
}

func (m *Manager) snapshotLocked() QueueSnapshot {
	snap := QueueSnapshot{
		Queued: m.queuedLocked(),
		Lanes:  m.laneStatesLocked(),
		Tasks:  m.snapshotEntriesLocked(),
	}
	for _, lane := range laneOrder {
		lr := m.lanes[lane]
		if lr != nil && lr.current != nil && !lr.current.finished {
			p := lr.current.progress
			snap.Current = &p
			break
		}
	}
	return snap
}

func syncEntryFromProgress(e *TaskEntry, p Progress) {
	e.Stage = p.Stage
	e.CurrentItem = p.CurrentItem
	e.Current = p.Current
	e.Total = p.Total
	e.Success = p.Success
	e.Skipped = p.Skipped
	e.Failed = p.Failed
	e.Cancelled = p.Cancelled
	e.Error = p.Error
}
