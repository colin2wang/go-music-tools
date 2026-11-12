package logx

import (
	"sync"
	"time"
)

// EmitFunc 由 Wails 绑定层注入，业务包不依赖 Wails runtime。
type EmitFunc func(event string, payload any)

// batchHandler 把高频日志聚合成批量事件，避免逐条推送压垮 WebView2 与前端重渲染。
type batchHandler struct {
	mu       sync.Mutex
	entries  []LogEntry
	limit    int
	interval time.Duration
	emit     func([]LogEntry)
	stop     chan struct{}
	done     chan struct{}
	once     sync.Once
}

func newBatchHandler(limit int, flushMS int) *batchHandler {
	if limit < 1 {
		limit = 1
	}
	if flushMS < 10 {
		flushMS = 10
	}
	b := &batchHandler{
		entries:  make([]LogEntry, 0, limit),
		limit:    limit,
		interval: time.Duration(flushMS) * time.Millisecond,
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
	}
	go b.loop()
	return b
}

func (b *batchHandler) loop() {
	defer close(b.done)
	ticker := time.NewTicker(b.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			b.Flush()
		case <-b.stop:
			b.Flush()
			return
		}
	}
}

func (b *batchHandler) SetEmitter(emit func([]LogEntry)) {
	b.mu.Lock()
	b.emit = emit
	b.mu.Unlock()
}

func (b *batchHandler) SetConfig(limit int, flushMS int) {
	if limit < 1 {
		limit = 1
	}
	if flushMS < 10 {
		flushMS = 10
	}
	b.mu.Lock()
	b.limit = limit
	b.interval = time.Duration(flushMS) * time.Millisecond
	b.mu.Unlock()
}

func (b *batchHandler) Add(e LogEntry) {
	b.mu.Lock()
	b.entries = append(b.entries, e)
	shouldFlush := len(b.entries) >= b.limit
	b.mu.Unlock()
	if shouldFlush {
		b.Flush()
	}
}

func (b *batchHandler) Flush() {
	b.mu.Lock()
	if len(b.entries) == 0 {
		b.mu.Unlock()
		return
	}
	batch := b.entries
	b.entries = make([]LogEntry, 0, b.limit)
	emit := b.emit
	b.mu.Unlock()
	if emit != nil {
		emit(batch)
	}
}

func (b *batchHandler) Close() {
	b.once.Do(func() { close(b.stop) })
	<-b.done
}
