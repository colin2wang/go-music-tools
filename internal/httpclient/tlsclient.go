package httpclient

import (
	"crypto/tls"
	"net/http"
	"time"
)

const Chrome122UA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36"

// chromeTLSConfig 用标准库近似 Chrome 的 TLS 握手特征（ALPN、曲线、套件顺序）。
// Python 版用 curl_cffi 的 impersonate="chrome120" 做 JA3 指纹；这里不引入 uTLS 大依赖，
// 标准 net/http + 完整 Chrome 请求头优先，仅在需要时切换到该握手配置。
func chromeTLSConfig(insecure bool) *tls.Config {
	return &tls.Config{
		InsecureSkipVerify: insecure, //nolint:gosec 第三方代理需要
		MinVersion:         tls.VersionTLS12,
		MaxVersion:         tls.VersionTLS13,
		CurvePreferences:   []tls.CurveID{tls.X25519, tls.CurveP256, tls.CurveP384},
		NextProtos:         []string{"h2", "http/1.1"},
		CipherSuites: []uint16{
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
		},
	}
}

// WithChromeFingerprint 返回使用指纹握手配置的客户端副本，供 NextMusic V2 通道使用。
func (c *Client) WithChromeFingerprint() *Client {
	transport := &http.Transport{
		TLSClientConfig:   chromeTLSConfig(c.insecure),
		MaxIdleConns:      10,
		IdleConnTimeout:   60 * time.Second,
		ForceAttemptHTTP2: true,
	}
	clone := *c
	clone.http = &http.Client{Timeout: c.http.Timeout, Transport: transport}
	clone.ua = Chrome122UA
	return &clone
}
