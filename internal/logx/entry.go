package logx

import (
	"log/slog"
	"strings"
)

// LogEntry 是全局日志窗口的单行 DTO，字段命名与前端 LogEntry 对齐。
type LogEntry struct {
	Time    string            `json:"time"`
	Level   string            `json:"level"`
	Source  string            `json:"source"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func levelName(l slog.Level) string {
	switch {
	case l >= slog.LevelError:
		return "ERROR"
	case l >= slog.LevelWarn:
		return "WARN"
	case l < slog.LevelInfo:
		return "DEBUG"
	default:
		return "INFO"
	}
}

func formatLine(e LogEntry) string {
	var b strings.Builder
	b.WriteString(e.Time)
	b.WriteString(" [")
	b.WriteString(e.Level)
	b.WriteString("] ")
	if e.Source != "" {
		b.WriteString(e.Source)
		b.WriteString(" - ")
	}
	b.WriteString(e.Message)
	if len(e.Fields) > 0 {
		for k, v := range e.Fields {
			b.WriteString(" ")
			b.WriteString(k)
			b.WriteString("=")
			b.WriteString(v)
		}
	}
	return b.String()
}
