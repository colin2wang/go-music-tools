package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

func SleepContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// RandomDelay 等待 [minSeconds, maxSeconds] 区间内的随机时长；logger 非 nil 时
// 先输出等待日志再进入等待，保证日志顺序与实际等待一致。
func RandomDelay(ctx context.Context, logger *slog.Logger, minSeconds, maxSeconds float64) error {
	if maxSeconds <= 0 {
		return nil
	}
	if minSeconds < 0 {
		minSeconds = 0
	}
	if maxSeconds < minSeconds {
		maxSeconds = minSeconds
	}
	d := minSeconds
	if maxSeconds > minSeconds {
		d = minSeconds + rand.Float64()*(maxSeconds-minSeconds)
	}
	if d > 0 && logger != nil {
		logger.Info("请求等待", "seconds", d)
	}
	return SleepContext(ctx, time.Duration(d*float64(time.Second)))
}

// PayloadRateLimited 检测代理把 429 藏在 JSON 里的情况，只检查 msg/message 字段，
// 避免把歌名或歌词里的 "429" 误判为限流。
func PayloadRateLimited(body []byte) bool {
	var probe struct {
		Msg     string `json:"msg"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &probe); err != nil {
		return false
	}
	return strings.Contains(probe.Msg, "429") || strings.Contains(probe.Message, "429")
}

func IsTooManyRequests(resp *Response) bool {
	return resp != nil && (resp.StatusCode == http.StatusTooManyRequests || PayloadRateLimited(resp.Body))
}

// RequestWithRetry 统一处理前置延迟、429 退避与网络错误重试；bodyFactory 每次重新生成请求体。
func (c *Client) RequestWithRetry(ctx context.Context, method, rawURL string, bodyFactory func() io.Reader,
	headers map[string]string, extraAttempts int) (*Response, error) {

	attempts := 1 + extraAttempts
	if attempts < 1 {
		attempts = 1
	}
	var lastErr error
	for i := 0; i < attempts; i++ {
		if err := RandomDelay(ctx, c.logger, c.cfg.APIDelayMin, c.cfg.RandomDelayMax); err != nil {
			return nil, err
		}
		var body io.Reader
		if bodyFactory != nil {
			body = bodyFactory()
		}
		resp, err := c.Request(ctx, method, rawURL, body, headers)
		if err != nil {
			lastErr = err
			if i == attempts-1 {
				break
			}
			if waitErr := c.backoff(ctx, i); waitErr != nil {
				return nil, waitErr
			}
			continue
		}
		if IsTooManyRequests(resp) && i < attempts-1 {
			if c.logger != nil {
				c.logger.Warn("接口返回限流，退避后重试", "attempt", i+1, "status", resp.StatusCode)
			}
			if waitErr := c.backoff(ctx, i); waitErr != nil {
				return nil, waitErr
			}
			continue
		}
		return resp, nil
	}
	if lastErr != nil {
		return nil, fmt.Errorf("请求失败（已重试 %d 次）: %w", attempts-1, lastErr)
	}
	return nil, fmt.Errorf("请求失败（已重试 %d 次）", attempts-1)
}

func (c *Client) backoff(ctx context.Context, attempt int) error {
	min := c.cfg.RetryBackoffMin
	max := c.cfg.RetryBackoffMax
	if max < min {
		max = min
	}
	d := min * float64(attempt+1)
	if d > max {
		d = max
	}
	if c.logger != nil {
		c.logger.Warn("等待后重试", "seconds", d, "attempt", attempt+1)
	}
	return SleepContext(ctx, time.Duration(d*float64(time.Second)))
}

// MetadataIntervalStore 由 internal/store.Store 实现，避免 httpclient 依赖具体存储。
type MetadataIntervalStore interface {
	GetMetadataTimestamp() (float64, error)
	SetMetadataTimestamp(float64) error
}

// WaitMetadataInterval 等待两次元数据请求之间的最小间隔；logger 非 nil 时
// 先输出等待日志再进入等待，保证日志顺序与实际等待一致。
func WaitMetadataInterval(ctx context.Context, logger *slog.Logger, store MetadataIntervalStore, interval float64) error {
	if store == nil || interval <= 0 {
		return nil
	}
	last, _ := store.GetMetadataTimestamp()
	now := float64(time.Now().UnixNano()) / 1e9
	if wait := interval - (now - last); wait > 0 {
		if logger != nil {
			logger.Info("元数据限流等待", "seconds", wait)
		}
		return SleepContext(ctx, time.Duration(wait*float64(time.Second)))
	}
	return nil
}

func MarkMetadataFinish(store MetadataIntervalStore) error {
	if store == nil {
		return nil
	}
	return store.SetMetadataTimestamp(float64(time.Now().UnixNano()) / 1e9)
}
