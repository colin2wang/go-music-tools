package task

import "strings"

const (
	EventLogBatch      = "log:batch"
	EventTaskProgress  = "task:progress"
	EventTaskItem      = "task:item"
	EventTaskDone      = "task:done"
	EventTaskQueued    = "task:queued"
	EventConfigChanged = "config:changed"
	EventAppReady      = "app:ready"
)

// Lane 是一条互不阻塞的执行通道：通道内严格 FIFO 串行，通道之间可以并行。
type Lane string

const (
	LaneWorkbench Lane = "workbench"
	LaneOOMusic   Lane = "oomusic"
)

// LaneFor 按任务类型路由通道：OO Music 独立一条，其余归音乐工作台。
func LaneFor(kind string) Lane {
	if strings.HasPrefix(kind, "oomusic") {
		return LaneOOMusic
	}
	return LaneWorkbench
}

// 任务生命周期状态，用于 TaskEntry.Status。
const (
	TaskStatusQueued    = "queued"
	TaskStatusRunning   = "running"
	TaskStatusDone      = "done"
	TaskStatusFailed    = "failed"
	TaskStatusCancelled = "cancelled"
	TaskStatusRemoved   = "removed"
)

// Progress 是 task:progress 的载荷。原字段保持不变，新增字段只做加法以兼容既有订阅方。
// 注意 Status 沿用 running | done 两态；任务五态见 TaskEntry.Status。
type Progress struct {
	TaskID      string `json:"task_id"`
	Kind        string `json:"kind"`
	Lane        Lane   `json:"lane"`
	Stage       string `json:"stage"`
	Status      string `json:"status"` // running | done
	CurrentItem string `json:"current_item"`
	Current     int    `json:"current"`
	Total       int    `json:"total"`
	Success     int    `json:"success"`
	Skipped     int    `json:"skipped"`
	Failed      int    `json:"failed"`
	Done        bool   `json:"done"`
	Cancelled   bool   `json:"cancelled"`
	Error       string `json:"error"`
	EnqueuedAt  int64  `json:"enqueued_at"`
	StartedAt   int64  `json:"started_at"`
	ElapsedMS   int64  `json:"elapsed_ms"`
	CanRetry    bool   `json:"can_retry"`
}

type ItemResult struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	Message   string `json:"message"`
	ElapsedMS int64  `json:"elapsed_ms"`
}

// QueuedTask 描述一个尚未开始执行的排队任务。
type QueuedTask struct {
	TaskID     string `json:"task_id"`
	Kind       string `json:"kind"`
	Lane       Lane   `json:"lane"`
	Status     string `json:"status"`
	Total      int    `json:"total"`
	EnqueuedAt int64  `json:"enqueued_at"` // unix 秒
}

// LaneState 汇总一条通道的运行与排队情况。
type LaneState struct {
	Lane    Lane   `json:"lane"`
	Running string `json:"running"` // 当前运行 task_id，空表示空闲
	Queued  int    `json:"queued"`
}

// TaskEntry 是一个独立任务的完整生命周期记录，按 task_id 保存在进程内注册表。
// Items 在快照里只带最近若干条，ItemCount 给出子项总数。
type TaskEntry struct {
	TaskID      string       `json:"task_id"`
	Kind        string       `json:"kind"`
	Lane        Lane         `json:"lane"`
	Status      string       `json:"status"` // queued | running | done | failed | cancelled | removed
	Stage       string       `json:"stage"`
	EnqueuedAt  int64        `json:"enqueued_at"`
	StartedAt   int64        `json:"started_at"`
	FinishedAt  int64        `json:"finished_at"`
	ElapsedMS   int64        `json:"elapsed_ms"`
	CurrentItem string       `json:"current_item"`
	Current     int          `json:"current"`
	Total       int          `json:"total"`
	Success     int          `json:"success"`
	Skipped     int          `json:"skipped"`
	Failed      int          `json:"failed"`
	Cancelled   bool         `json:"cancelled"`
	Error       string       `json:"error"`
	Items       []ItemResult `json:"items"`
	ItemCount   int          `json:"item_count"`
	CanRetry    bool         `json:"can_retry"`
}

// QueueSnapshot 是 task:queued 的载荷：当前运行进度、等待队列、通道状态与任务注册表快照。
type QueueSnapshot struct {
	Current *Progress    `json:"current"`
	Queued  []QueuedTask `json:"queued"`
	Lanes   []LaneState  `json:"lanes"`
	Tasks   []TaskEntry  `json:"tasks"`
}

// TaskBoard 是 GetTaskBoard 的一次性全量快照。
type TaskBoard struct {
	Lanes []LaneState `json:"lanes"`
	Tasks []TaskEntry `json:"tasks"`
}

// ItemEvent 用于实时推送逐目录 / 逐曲结果，不必等到任务结束才刷新界面。
type ItemEvent struct {
	TaskID string     `json:"task_id"`
	Lane   Lane       `json:"lane"`
	Item   ItemResult `json:"item"`
}

type Done struct {
	TaskID     string       `json:"task_id"`
	Kind       string       `json:"kind"`
	Lane       Lane         `json:"lane"`
	Status     string       `json:"status"`
	Success    int          `json:"success"`
	Skipped    int          `json:"skipped"`
	Failed     int          `json:"failed"`
	Cancelled  bool         `json:"cancelled"`
	Error      string       `json:"error"`
	Items      []ItemResult `json:"items"`
	ElapsedMS  int64        `json:"elapsed_ms"`
	StartedAt  int64        `json:"started_at"`
	FinishedAt int64        `json:"finished_at"`
}

// RetryFactory 根据失败子项重建一个可重新入队的任务函数。
// 由绑定层在提交时捕获原始请求生成，避免改动业务循环中大量 r.Item 调用。
type RetryFactory func(failed []ItemResult) (TaskFunc, error)

type startOptions struct {
	retry RetryFactory
}

// StartOption 在提交任务时附加可选行为。
type StartOption func(*startOptions)

// WithRetry 注入失败子项重试工厂；未注入的任务 CanRetry 为 false，界面会禁用重试。
func WithRetry(fn RetryFactory) StartOption {
	return func(o *startOptions) { o.retry = fn }
}
