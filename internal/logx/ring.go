package logx

import "sync"

// Ring 是固定容量的日志环形缓冲，供前端挂载后回放启动早期日志。
type Ring struct {
	mu       sync.RWMutex
	entries  []LogEntry
	capacity int
	next     int
	count    int
}

func NewRing(capacity int) *Ring {
	if capacity < 1 {
		capacity = 1
	}
	return &Ring{entries: make([]LogEntry, capacity), capacity: capacity}
}

func (r *Ring) Add(e LogEntry) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries[r.next] = e
	r.next = (r.next + 1) % r.capacity
	if r.count < r.capacity {
		r.count++
	}
}

func (r *Ring) Snapshot() []LogEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]LogEntry, 0, r.count)
	start := (r.next - r.count + r.capacity) % r.capacity
	for i := 0; i < r.count; i++ {
		out = append(out, r.entries[(start+i)%r.capacity])
	}
	return out
}

func (r *Ring) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.next = 0
	r.count = 0
}

// Resize 保留最近的日志，避免配置热更新时丢失上下文。
func (r *Ring) Resize(capacity int) {
	if capacity < 1 {
		capacity = 1
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	kept := r.SnapshotLocked()
	if len(kept) > capacity {
		kept = kept[len(kept)-capacity:]
	}
	r.entries = make([]LogEntry, capacity)
	r.capacity = capacity
	r.next = 0
	r.count = 0
	for _, e := range kept {
		r.entries[r.next] = e
		r.next = (r.next + 1) % capacity
		r.count++
	}
}

// SnapshotLocked 仅允许在已持有写锁时调用。
func (r *Ring) SnapshotLocked() []LogEntry {
	out := make([]LogEntry, 0, r.count)
	start := (r.next - r.count + r.capacity) % r.capacity
	for i := 0; i < r.count; i++ {
		out = append(out, r.entries[(start+i)%r.capacity])
	}
	return out
}
