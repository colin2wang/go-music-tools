package task

const (
	EventLogBatch      = "log:batch"
	EventTaskProgress  = "task:progress"
	EventTaskItem      = "task:item"
	EventTaskDone      = "task:done"
	EventConfigChanged = "config:changed"
	EventAppReady      = "app:ready"
)

type Progress struct {
	TaskID      string `json:"task_id"`
	Kind        string `json:"kind"`
	Stage       string `json:"stage"`
	CurrentItem string `json:"current_item"`
	Current     int    `json:"current"`
	Total       int    `json:"total"`
	Success     int    `json:"success"`
	Skipped     int    `json:"skipped"`
	Failed      int    `json:"failed"`
	Done        bool   `json:"done"`
	Cancelled   bool   `json:"cancelled"`
	Error       string `json:"error"`
}

type ItemResult struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	Message   string `json:"message"`
	ElapsedMS int64  `json:"elapsed_ms"`
}

// ItemEvent 用于实时推送逐目录 / 逐曲结果，不必等到任务结束才刷新界面。
type ItemEvent struct {
	TaskID string     `json:"task_id"`
	Item   ItemResult `json:"item"`
}

type Done struct {
	TaskID    string       `json:"task_id"`
	Kind      string       `json:"kind"`
	Success   int          `json:"success"`
	Skipped   int          `json:"skipped"`
	Failed    int          `json:"failed"`
	Cancelled bool         `json:"cancelled"`
	Error     string       `json:"error"`
	Items     []ItemResult `json:"items"`
	ElapsedMS int64        `json:"elapsed_ms"`
}
