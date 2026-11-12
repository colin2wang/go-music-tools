package library

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go-music-tools/internal/music"
	"go-music-tools/internal/tag"
)

// FixMetadata 仅对缺失标题 / 艺术家 / 专辑的文件回填标签，年份取 publish date 前四位。
func (s *Service) FixMetadata(ctx context.Context, folder string, onlyMissing, recursive bool) (string, error) {
	files, err := FindAudioFiles(folder, recursive)
	if err != nil {
		return "", err
	}
	info, err := LoadAlbumInfo(folder)
	if err != nil && s.Logger != nil {
		s.Logger.Warn("未找到 album_info.txt，将仅依据文件名回填", "folder", filepath.Base(folder))
	}
	year := ""
	if len(info.PublishDate) >= 4 {
		year = info.PublishDate[:4]
	}

	coverPath := FindCoverImage(folder)
	var cover []byte
	coverMIME := ""
	if coverPath != "" {
		if data, err := os.ReadFile(coverPath); err == nil && len(data) > 0 {
			cover = data
			coverMIME = tag.MimeFromData(data)
		}
	}

	fixed, skipped := 0, 0
	for i, file := range files {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		hasTitle, hasArtist, hasAlbum, _, statusErr := tag.MetadataStatus(file)
		if statusErr == nil && onlyMissing && hasTitle && hasArtist && hasAlbum {
			skipped++
			continue
		}
		stem := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
		track, artist, title := music.ParseMusicFilename(stem)
		if track == 0 {
			track = i + 1
		}
		albumName := info.AlbumName
		if albumName == "" {
			existing, err := tag.Read(file)
			if err == nil && existing.Album != "" {
				albumName = existing.Album
			}
		}
		patch := tag.TrackTags{
			Title:       title,
			Artists:     tag.SplitArtists(artist),
			Album:       albumName,
			TrackNumber: track,
			Year:        year,
			Cover:       cover,
			CoverMIME:   coverMIME,
		}
		if err := tag.Write(file, patch); err != nil {
			if s.Logger != nil {
				s.Logger.Warn("回填元数据失败", "file", stem, "error", err)
			}
			continue
		}
		fixed++
	}
	return fmt.Sprintf("补全 %d 个文件，跳过 %d 个", fixed, skipped), nil
}
