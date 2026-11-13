package logx

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"go-music-tools/internal/config"
	"go-music-tools/internal/task"
	"gopkg.in/natefinch/lumberjack.v2"
)

const logTimeLayout = "2006-01-02 15:04:05.000"

// Hub 是日志中枢：一条 slog 记录同时进入轮转文件、控制台、环形缓冲和前端事件批量通道。
type Hub struct {
	mu      sync.RWMutex
	root    string
	cfg     config.LoggingConfig
	level   *slog.LevelVar
	ring    *Ring
	batch   *batchHandler
	file    *lumberjack.Logger
	fileDay string
	logger  *slog.Logger
	logDir  string
	emitter EmitFunc
	closed  bool
}

func NewHub(cfg config.LoggingConfig, root string) (*Hub, error) {
	dir := resolveDir(root, cfg.LogDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("创建日志目录 %s: %w", dir, err)
	}

	level := new(slog.LevelVar)
	level.Set(cfg.SlogLevel())

	h := &Hub{
		root:   root,
		cfg:    cfg,
		level:  level,
		ring:   NewRing(cfg.BufferSize),
		batch:  newBatchHandler(cfg.EventBatchSize, cfg.EventFlushMS),
		logDir: dir,
	}
	h.fileDay = time.Now().Format("2006-01-02")
	h.file = h.newFileLogger(dir, cfg, h.fileDay)
	h.logger = slog.New(&hubHandler{hub: h})
	return h, nil
}

func (h *Hub) newFileLogger(dir string, cfg config.LoggingConfig, day string) *lumberjack.Logger {
	return &lumberjack.Logger{
		Filename:   filepath.Join(dir, "app-"+day+".log"),
		MaxSize:    cfg.MaxFileSizeMB,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAgeDays,
	}
}

func resolveDir(root, dir string) string {
	if dir == "" {
		dir = "logs"
	}
	if filepath.IsAbs(dir) {
		return dir
	}
	return filepath.Join(root, dir)
}

// fileFor 返回当前应写入的日志文件；跨天时切换到新日期文件（按天 rolling，
// 天内仍由 lumberjack 按大小轮转）。写路径在每次 Handle 时调用，代价是一次读锁。
func (h *Hub) fileFor(now time.Time) *lumberjack.Logger {
	day := now.Format("2006-01-02")
	h.mu.RLock()
	if h.fileDay == day {
		f := h.file
		h.mu.RUnlock()
		return f
	}
	h.mu.RUnlock()

	h.mu.Lock()
	defer h.mu.Unlock()
	if h.fileDay != day {
		if h.file != nil {
			_ = h.file.Close()
		}
		h.fileDay = day
		h.file = h.newFileLogger(h.logDir, h.cfg, day)
	}
	return h.file
}

func (h *Hub) Logger() *slog.Logger { return h.logger }
func (h *Hub) LogDir() string       { return h.logDir }

// ValidateLevel 校验级别名（debug/info/warn/error，大小写不敏感）。
func ValidateLevel(name string) error {
	switch strings.ToLower(name) {
	case "debug", "info", "warn", "error":
		return nil
	}
	return fmt.Errorf("无效日志级别: %s", name)
}

// SetLevel 运行时调整全局日志级别，立即生效；不写入 config.yml。
func (h *Hub) SetLevel(name string) error {
	if err := ValidateLevel(name); err != nil {
		return err
	}
	h.level.Set(config.LoggingConfig{Level: strings.ToLower(name)}.SlogLevel())
	return nil
}

func (h *Hub) Recent() []LogEntry { return h.ring.Snapshot() }
func (h *Hub) Clear()             { h.ring.Clear() }
func (h *Hub) Flush()             { h.batch.Flush() }

// SetEmitter 在 Wails startup 时注入；注入前的日志仍保留在环形缓冲中供前端回放。
func (h *Hub) SetEmitter(fn EmitFunc) {
	h.mu.Lock()
	h.emitter = fn
	h.mu.Unlock()
	h.batch.SetEmitter(func(entries []LogEntry) {
		h.mu.RLock()
		emit := h.emitter
		h.mu.RUnlock()
		if emit != nil {
			emit(task.EventLogBatch, entries)
		}
	})
	h.Flush()
}

// ApplyConfig 让设置页的日志配置即时生效，不需要重启应用。
func (h *Hub) ApplyConfig(cfg config.LoggingConfig) {
	h.mu.Lock()
	dir := resolveDir(h.root, cfg.LogDir)
	if dir != h.logDir {
		_ = os.MkdirAll(dir, 0o755)
		if h.file != nil {
			_ = h.file.Close()
		}
		h.logDir = dir
		h.fileDay = time.Now().Format("2006-01-02")
		h.file = h.newFileLogger(dir, cfg, h.fileDay)
	} else {
		h.file.MaxSize = cfg.MaxFileSizeMB
		h.file.MaxBackups = cfg.MaxBackups
		h.file.MaxAge = cfg.MaxAgeDays
	}
	h.cfg = cfg
	h.level.Set(cfg.SlogLevel())
	h.mu.Unlock()

	h.ring.Resize(cfg.BufferSize)
	h.batch.SetConfig(cfg.EventBatchSize, cfg.EventFlushMS)
	h.Flush()
}

// Export 把当前可回放日志写成纯文本文件，供「导出日志」按钮使用。
func (h *Hub) Export(path string) error {
	entries := h.Recent()
	lines := make([]string, 0, len(entries))
	for _, e := range entries {
		lines = append(lines, formatLine(e))
	}
	data := []byte("")
	for _, line := range lines {
		data = append(data, []byte(line+"\n")...)
	}
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, data, 0o644)
}

func (h *Hub) Close() error {
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return nil
	}
	h.closed = true
	file := h.file
	h.mu.Unlock()

	h.batch.Close()
	if file != nil {
		return file.Close()
	}
	return nil
}

// hubHandler 实现 slog.Handler。WithAttrs 返回副本以避免不同 logger 的结构化字段互相污染。
type hubHandler struct {
	hub   *Hub
	attrs []slog.Attr
}

func (h *hubHandler) Enabled(_ context.Context, l slog.Level) bool {
	return l >= h.hub.level.Level()
}

func (h *hubHandler) Handle(_ context.Context, r slog.Record) error {
	fields := make(map[string]string)
	for _, a := range h.attrs {
		addAttr(fields, a)
	}
	r.Attrs(func(a slog.Attr) bool {
		addAttr(fields, a)
		return true
	})

	h.hub.mu.RLock()
	cfg := h.hub.cfg
	h.hub.mu.RUnlock()

	entry := LogEntry{
		Time:    r.Time.Format(logTimeLayout),
		Level:   levelName(r.Level),
		Message: r.Message,
	}
	if cfg.ShowSource {
		entry.Source = sourceOf(r)
	}
	if len(fields) > 0 {
		entry.Fields = fields
	}

	h.hub.ring.Add(entry)
	h.hub.batch.Add(entry)

	line := formatLine(entry)
	if cfg.ConsoleOutput {
		fmt.Fprintln(os.Stdout, line)
	}
	if cfg.FileOutput {
		if file := h.hub.fileFor(r.Time); file != nil {
			fmt.Fprintln(file, line)
		}
	}
	return nil
}

func (h *hubHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	clone := *h
	clone.attrs = append(append([]slog.Attr(nil), h.attrs...), attrs...)
	return &clone
}

func (h *hubHandler) WithGroup(_ string) slog.Handler { return h }

func addAttr(dst map[string]string, a slog.Attr) {
	if a.Key == "" {
		return
	}
	v := a.Value.Resolve().Any()
	switch x := v.(type) {
	case nil:
		dst[a.Key] = ""
	case string:
		dst[a.Key] = x
	case error:
		dst[a.Key] = x.Error()
	default:
		dst[a.Key] = fmt.Sprint(x)
	}
}

func sourceOf(r slog.Record) string {
	if r.PC == 0 {
		return ""
	}
	frames := runtime.CallersFrames([]uintptr{r.PC})
	f, _ := frames.Next()
	if f.File == "" {
		return ""
	}
	return fmt.Sprintf("%s:%d", filepath.Base(f.File), f.Line)
}
