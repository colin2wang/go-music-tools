package httpclient

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go-music-tools/internal/config"
)

const ChromeUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36"

// Response 把响应体一次性读出，便于统一做 429 payload 检测与 JSON 解析。
type Response struct {
	StatusCode int
	Body       []byte
	Header     http.Header
}

type Client struct {
	http     *http.Client
	cfg      config.NetworkConfig
	insecure bool
	logger   *slog.Logger
	ua       string
	origin   string
}

func New(cfg config.NetworkConfig, insecure bool, logger *slog.Logger) *Client {
	transport := &http.Transport{
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: insecure}, //nolint:gosec 第三方代理需要
		MaxIdleConns:      20,
		IdleConnTimeout:   60 * time.Second,
		ForceAttemptHTTP2: true,
	}
	return &Client{
		http:     &http.Client{Timeout: seconds(cfg.Timeout), Transport: transport},
		cfg:      cfg,
		insecure: insecure,
		logger:   logger,
		ua:       ChromeUA,
	}
}

func seconds(v float64) time.Duration {
	if v <= 0 {
		v = 30
	}
	return time.Duration(v * float64(time.Second))
}

func (c *Client) ApplyConfig(cfg config.NetworkConfig, insecure bool) {
	c.cfg = cfg
	c.insecure = insecure
	c.http.Timeout = seconds(cfg.Timeout)
	if transport, ok := c.http.Transport.(*http.Transport); ok {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: insecure} //nolint:gosec
	}
}

func (c *Client) Do(ctx context.Context, method, rawURL string, body io.Reader, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, rawURL, body)
	if err != nil {
		return nil, fmt.Errorf("构造请求 %s: %w", method, err)
	}
	c.applyHeaders(req, rawURL)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return c.http.Do(req)
}

func (c *Client) applyHeaders(req *http.Request, rawURL string) {
	req.Header.Set("user-agent", c.ua)
	req.Header.Set("accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("accept-language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("sec-ch-ua", `"Chromium";v="132", "Not_A Brand";v="8", "Google Chrome";v="132"`)
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", `"Windows"`)
	req.Header.Set("sec-fetch-dest", "empty")
	req.Header.Set("sec-fetch-mode", "cors")
	req.Header.Set("sec-fetch-site", "same-origin")

	origin := c.origin
	if origin == "" {
		if u, err := url.Parse(rawURL); err == nil {
			origin = u.Scheme + "://" + u.Host
		}
	}
	if origin != "" {
		req.Header.Set("origin", origin)
		req.Header.Set("referer", origin+"/")
	}
}

func (c *Client) SetOrigin(origin string) { c.origin = strings.TrimRight(origin, "/") }
func (c *Client) SetUserAgent(ua string)  { c.ua = ua }

func (c *Client) Request(ctx context.Context, method, rawURL string, body io.Reader, headers map[string]string) (*Response, error) {
	var params string
	if body != nil {
		// 请求体都是小体积 JSON/表单，读出来既能打 debug 日志也能复用为请求体。
		data, err := io.ReadAll(io.LimitReader(body, 64<<10))
		if err != nil {
			return nil, fmt.Errorf("读取请求体: %w", err)
		}
		body = bytes.NewReader(data)
		params = string(data)
	}
	if c.logger != nil {
		c.logger.Debug("HTTP 请求", "method", method, "url", rawURL, "params", params)
	}
	resp, err := c.Do(ctx, method, rawURL, body, headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return nil, fmt.Errorf("读取响应体: %w", err)
	}
	if c.logger != nil {
		c.logger.Debug("HTTP 响应", "status", resp.StatusCode, "url", rawURL, "body", truncateForLog(data))
	}
	return &Response{StatusCode: resp.StatusCode, Body: data, Header: resp.Header.Clone()}, nil
}

// truncateForLog 截断超长响应体，避免 debug 日志被单曲列表等大 JSON 撑爆。
func truncateForLog(data []byte) string {
	const max = 4096
	if len(data) <= max {
		return string(data)
	}
	return string(data[:max]) + fmt.Sprintf("...(截断，共 %d 字节)", len(data))
}

func (c *Client) Get(ctx context.Context, rawURL string, headers map[string]string) (*Response, error) {
	return c.Request(ctx, http.MethodGet, rawURL, nil, headers)
}

func (c *Client) PostForm(ctx context.Context, rawURL string, form url.Values, headers map[string]string) (*Response, error) {
	if headers == nil {
		headers = map[string]string{}
	}
	headers["content-type"] = "application/x-www-form-urlencoded; charset=UTF-8"
	return c.Request(ctx, http.MethodPost, rawURL, strings.NewReader(form.Encode()), headers)
}

func (c *Client) PostJSON(ctx context.Context, rawURL string, payload any, headers map[string]string) (*Response, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	if headers == nil {
		headers = map[string]string{}
	}
	headers["content-type"] = "application/json"
	return c.Request(ctx, http.MethodPost, rawURL, strings.NewReader(string(data)), headers)
}

// URLError 表示音频地址请求层面的失败（HTTP 状态非 200），带状态码供上层判断地址是否已失效。
type URLError struct {
	StatusCode int
	URL        string
}

func (e *URLError) Error() string {
	return fmt.Sprintf("下载失败，HTTP %d: %s", e.StatusCode, e.URL)
}

// IsURLExpired 判断错误是否由播放地址失效（403/404/410 等）引起，此时应重新获取地址再重试。
func IsURLExpired(err error) bool {
	var ue *URLError
	return errors.As(err, &ue) && (ue.StatusCode == http.StatusForbidden || ue.StatusCode == http.StatusNotFound || ue.StatusCode == http.StatusGone)
}

// Download 流式写入 .part 后原子 rename，取消时清理临时文件。
func (c *Client) Download(ctx context.Context, rawURL, dest string, headers map[string]string, progress func(written, total int64)) (string, error) {
	if c.logger != nil {
		c.logger.Debug("下载音频", "url", rawURL, "dest", dest)
	}
	resp, err := c.Do(ctx, http.MethodGet, rawURL, nil, headers)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", &URLError{StatusCode: resp.StatusCode, URL: rawURL}
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return "", err
	}

	part := dest + ".part"
	f, err := os.Create(part)
	if err != nil {
		return "", err
	}
	ok := false
	defer func() {
		_ = f.Close()
		if !ok {
			_ = os.Remove(part)
		}
	}()

	buf := make([]byte, 32*1024)
	var written int64
	for {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, err := f.Write(buf[:n]); err != nil {
				return "", err
			}
			written += int64(n)
			if progress != nil {
				progress(written, resp.ContentLength)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return "", readErr
		}
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	_ = os.Remove(dest)
	if err := os.Rename(part, dest); err != nil {
		return "", err
	}
	ok = true
	return dest, nil
}

// DownloadBytes 用于封面、歌词等小体积资源。
func (c *Client) DownloadBytes(ctx context.Context, rawURL string, headers map[string]string) ([]byte, string, error) {
	resp, err := c.Do(ctx, http.MethodGet, rawURL, nil, headers)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("请求失败，HTTP %d: %s", resp.StatusCode, rawURL)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		return nil, "", err
	}
	return data, resp.Header.Get("content-type"), nil
}
