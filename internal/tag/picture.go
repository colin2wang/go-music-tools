package tag

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

var imageExtensions = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".webp": true, ".bmp": true,
}

func MimeFromURL(raw string) string {
	lower := strings.ToLower(raw)
	if u, err := url.Parse(raw); err == nil {
		lower = strings.ToLower(u.Path)
	}
	switch {
	case strings.HasSuffix(lower, ".png"):
		return "image/png"
	case strings.HasSuffix(lower, ".webp"):
		return "image/webp"
	case strings.HasSuffix(lower, ".bmp"):
		return "image/bmp"
	default:
		return "image/jpeg"
	}
}

func MimeFromData(data []byte) string {
	if len(data) >= 8 && bytes.Equal(data[:8], []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}) {
		return "image/png"
	}
	if len(data) >= 2 && data[0] == 0xFF && data[1] == 0xD8 {
		return "image/jpeg"
	}
	return "image/jpeg"
}

// ImageSize 用于存量库封面分辨率检查（原版阈值 400x400）。
func ImageSize(data []byte) (int, int) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return 0, 0
	}
	return cfg.Width, cfg.Height
}

func MP4ImageFlag(mime string) uint32 {
	if strings.Contains(strings.ToLower(mime), "png") {
		return 14
	}
	return 13
}

func FormatFileSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for value := n / unit; value >= unit; value /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}

// FindCoverImage 优先 cover.*，其次任意图片文件。
func FindCoverImage(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	var fallback string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if !imageExtensions[ext] {
			continue
		}
		full := filepath.Join(dir, e.Name())
		if strings.HasPrefix(strings.ToLower(e.Name()), "cover.") {
			return full
		}
		if fallback == "" {
			fallback = full
		}
	}
	return fallback
}
