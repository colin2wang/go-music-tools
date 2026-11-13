package httpclient

import (
	"fmt"
	"log/slog"
	"sync"
)

// ThrottledProgress 返回按百分比节流的进度回调：总大小已知时每变化 ≥10% 或完成时
// 记一条 Info 日志；总大小未知时每 8MB 记一条。logger 为 nil 时返回 nil（无日志）。
func ThrottledProgress(logger *slog.Logger, label string) func(written, total int64) {
	if logger == nil {
		return nil
	}
	var mu sync.Mutex
	lastPercent := -10
	lastBucket := int64(-1)
	return func(written, total int64) {
		mu.Lock()
		defer mu.Unlock()
		if total > 0 {
			percent := int(written * 100 / total)
			if written == total {
				percent = 100
			}
			if percent != 100 && percent-lastPercent < 10 {
				return
			}
			lastPercent = percent
			logger.Info("下载进度", "file", label,
				"percent", fmt.Sprintf("%d%%", percent),
				"written", formatBytes(written), "total", formatBytes(total))
			return
		}
		bucket := written >> 23 // 8MB
		if bucket == lastBucket {
			return
		}
		lastBucket = bucket
		logger.Info("下载进度", "file", label, "written", formatBytes(written))
	}
}

func formatBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}
