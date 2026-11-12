package library

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"go-music-tools/internal/tag"
)

const AlbumInfoFilename = "album_info.txt"

type AlbumInfo struct {
	AlbumName   string
	Artist      string
	AlbumID     string
	PublishDate string
	SongCount   int
	Description string
}

func normalizeInfoKey(key string) string {
	key = strings.ToLower(strings.TrimSpace(key))
	key = strings.NewReplacer("_", " ", "-", " ").Replace(key)
	return strings.Join(strings.Fields(key), " ")
}

// LoadAlbumInfo 兼容 Go 版与 Python 版生成的 album_info.txt 字段名。
func LoadAlbumInfo(dir string) (AlbumInfo, error) {
	data, err := os.ReadFile(filepath.Join(dir, AlbumInfoFilename))
	if err != nil {
		return AlbumInfo{}, fmt.Errorf("读取 %s 失败: %w", AlbumInfoFilename, err)
	}
	info := AlbumInfo{}
	var description strings.Builder
	inDescription := false
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimRight(line, "\r")
		if inDescription {
			description.WriteString(trimmed + "\n")
			continue
		}
		idx := strings.Index(trimmed, ":")
		if idx < 0 {
			continue
		}
		key := normalizeInfoKey(trimmed[:idx])
		value := strings.TrimSpace(trimmed[idx+1:])
		switch key {
		case "album", "album name":
			info.AlbumName = value
		case "artist", "album artist":
			info.Artist = value
		case "album id":
			info.AlbumID = value
		case "publish date":
			info.PublishDate = value
		case "song count":
			info.SongCount, _ = strconv.Atoi(value)
		case "description":
			inDescription = true
		}
	}
	info.Description = strings.TrimSpace(description.String())
	return info, nil
}

func FindAudioFiles(dir string, recursive bool) ([]string, error) {
	var files []string
	if !recursive {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if !entry.IsDir() && tag.IsAudio(entry.Name()) {
				files = append(files, filepath.Join(dir, entry.Name()))
			}
		}
	} else {
		err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !entry.IsDir() && tag.IsAudio(entry.Name()) {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Slice(files, func(i, j int) bool {
		return strings.ToLower(filepath.Base(files[i])) < strings.ToLower(filepath.Base(files[j]))
	})
	return files, nil
}

func FindCoverImage(dir string) string { return tag.FindCoverImage(dir) }

// FindLyricsFiles 返回目录下所有 .lrc 文件，供歌词修复结束后归档。
func FindLyricsFiles(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	out := make([]string, 0)
	for _, entry := range entries {
		if !entry.IsDir() && strings.EqualFold(filepath.Ext(entry.Name()), ".lrc") {
			out = append(out, filepath.Join(dir, entry.Name()))
		}
	}
	return out
}
