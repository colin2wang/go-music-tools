package music

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"go-music-tools/internal/config"
	"go-music-tools/internal/model"
	"go-music-tools/internal/tag"
)

var illegalCharsRe = regexp.MustCompile(`[<>:"/\\|?*]`)
var indexedNameRe = regexp.MustCompile(`^(\d+)\s*-\s*(.+?)\s*-\s*(.+)$`)
var plainNameRe = regexp.MustCompile(`^(.+?)\s*-\s*(.+)$`)

func CleanFilename(name, replacement string) string {
	if replacement == "" {
		replacement = "_"
	}
	cleaned := illegalCharsRe.ReplaceAllString(name, replacement)
	cleaned = strings.TrimSpace(cleaned)
	return strings.TrimRight(cleaned, ". ")
}

// ExtractExtension 从下载地址末段、query 的 v 参数推断扩展名，默认 .mp3。
func ExtractExtension(raw string) string {
	parsed, err := url.Parse(raw)
	candidates := []string{}
	if err == nil {
		candidates = append(candidates, filepath.Ext(parsed.Path))
		if v := parsed.Query().Get("v"); v != "" {
			candidates = append(candidates, filepath.Ext(v))
		}
	} else {
		candidates = append(candidates, filepath.Ext(strings.SplitN(raw, "?", 2)[0]))
	}
	for _, ext := range candidates {
		switch strings.ToLower(ext) {
		case ".mp3", ".mp2", ".mp1", ".flac", ".m4a", ".mp4", ".aac":
			return strings.ToLower(ext)
		}
	}
	return ".mp3"
}

func formatIndex(index int, pattern string) string {
	switch pattern {
	case "d", "1d":
		return fmt.Sprintf("%d", index)
	case "2d", "02d":
		return fmt.Sprintf("%02d", index)
	case "4d", "04d":
		return fmt.Sprintf("%04d", index)
	case "5d", "05d":
		return fmt.Sprintf("%05d", index)
	default:
		return fmt.Sprintf("%03d", index)
	}
}

// BuildFilename 按配置模板生成文件名（不含扩展名），并按最大长度逐级收缩。
func BuildFilename(cfg config.Config, meta model.SongMeta, index int) string {
	pattern := cfg.Filename.Pattern
	if !cfg.Download.AddIndexToFilename || index <= 0 {
		pattern = strings.ReplaceAll(pattern, "{index}", "")
		pattern = strings.TrimLeft(pattern, " -_")
		if pattern == "" {
			pattern = "{artist} - {title}"
		}
	}
	artists := tag.SplitArtists(meta.Artists)
	if len(artists) == 0 {
		artists = []string{"Unknown Artist"}
	}
	replacement := cfg.Filename.IllegalCharReplacement
	if replacement == "" {
		replacement = "_"
	}
	joiner := cfg.Filename.ArtistDelimiterReplacement
	if joiner == "" {
		joiner = ", "
	}

	build := func(names []string, artistLimit, titleLimit int) string {
		cleaned := make([]string, 0, len(names))
		for _, name := range names {
			cleaned = append(cleaned, CleanFilename(name, replacement))
		}
		artist := strings.Join(cleaned, joiner)
		if artistLimit > 0 && len([]rune(artist)) > artistLimit {
			artist = string([]rune(artist)[:artistLimit])
		}
		title := CleanFilename(meta.Name, replacement)
		if titleLimit > 0 && len([]rune(title)) > titleLimit {
			title = string([]rune(title)[:titleLimit])
		}
		if title == "" {
			title = "Unknown Title"
		}
		name := pattern
		name = strings.ReplaceAll(name, "{index}", formatIndex(index, cfg.Download.IndexFormat))
		name = strings.ReplaceAll(name, "{artist}", artist)
		name = strings.ReplaceAll(name, "{title}", title)
		name = strings.ReplaceAll(name, "{album}", CleanFilename(meta.Album, replacement))
		return strings.TrimSpace(strings.TrimRight(name, ". "))
	}

	limit := cfg.Filename.MaxLength
	if limit <= 0 {
		limit = 120
	}
	name := build(artists, 0, 0)
	if len([]rune(name)) <= limit {
		return name
	}
	for _, candidate := range []int{limit, 80, 60, 40} {
		if candidate > limit {
			continue
		}
		shrunk := build(artists[:1], candidate/3, candidate/2)
		if len([]rune(shrunk)) <= candidate {
			return shrunk
		}
	}
	return string([]rune(build(artists[:1], 20, 40))[:40])
}

// EnsureWritableName 通过实际创建临时文件探测 Windows 长路径限制，逐级缩短文件名。
func EnsureWritableName(dir, base, ext string) string {
	limits := []int{120, 80, 60, 40}
	for _, limit := range limits {
		candidate := base
		if len([]rune(candidate)) > limit {
			candidate = string([]rune(candidate)[:limit])
		}
		candidate = strings.TrimRight(candidate, ". ")
		if candidate == "" {
			continue
		}
		probe := filepath.Join(dir, candidate+ext+".probe")
		f, err := os.OpenFile(probe, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			_ = f.Close()
			_ = os.Remove(probe)
			return candidate
		}
		if !os.IsExist(err) {
			continue
		}
	}
	return strings.TrimRight(string([]rune(base)[:min(40, len([]rune(base)))]), ". ")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ParseMusicFilename 解析 "{index} - {artist} - {title}" 或 "{artist} - {title}"。
func ParseMusicFilename(stem string) (track int, artist, title string) {
	if m := indexedNameRe.FindStringSubmatch(stem); len(m) == 4 {
		n := 0
		for _, r := range m[1] {
			n = n*10 + int(r-'0')
		}
		return n, m[2], m[3]
	}
	if m := plainNameRe.FindStringSubmatch(stem); len(m) == 3 {
		return 0, m[1], m[2]
	}
	return 0, "", stem
}
