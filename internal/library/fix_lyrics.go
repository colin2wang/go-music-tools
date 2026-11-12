package library

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"go-music-tools/internal/httpclient"
	"go-music-tools/internal/music"
	"go-music-tools/internal/tag"
)

var fuzzyRe = regexp.MustCompile(`[!？。、，\s&/,]+`)

func normalizeForMatch(value string) string {
	return strings.ToLower(fuzzyRe.ReplaceAllString(value, ""))
}

func findMusicFile(files []string, artist, title string) string {
	targetTitle := normalizeForMatch(title)
	targetArtist := normalizeForMatch(artist)
	for _, file := range files {
		stem := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
		_, parsedArtist, parsedTitle := music.ParseMusicFilename(stem)
		if normalizeForMatch(parsedTitle) == targetTitle && normalizeForMatch(parsedArtist) == targetArtist {
			return file
		}
	}
	for _, file := range files {
		stem := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
		_, parsedArtist, parsedTitle := music.ParseMusicFilename(stem)
		if normalizeForMatch(parsedTitle) != targetTitle {
			continue
		}
		flat := normalizeForMatch(parsedArtist)
		if flat == "" || strings.Contains(flat, targetArtist) || strings.Contains(targetArtist, flat) {
			return file
		}
	}
	return ""
}

// FixLyrics 以 CMUSIC_ID 作为幂等判据，按专辑 ID 拉取完整歌词并嵌入标签。
func (s *Service) FixLyrics(ctx context.Context, folder string, recursive bool) (string, error) {
	files, err := FindAudioFiles(folder, recursive)
	if err != nil {
		return "", err
	}
	if len(files) == 0 {
		return "目录中没有音频文件", nil
	}
	allTagged := true
	for _, file := range files {
		if !tag.HasCMUSICID(file) {
			allTagged = false
			break
		}
	}
	if allTagged {
		return "所有文件均含 CMUSIC_ID，跳过修复", nil
	}

	info, err := LoadAlbumInfo(folder)
	if err != nil {
		return "", err
	}
	if info.AlbumID == "" {
		return "", fmt.Errorf("album_info.txt 中缺少 Album id，无法获取曲目表")
	}
	album, err := s.Music.GetAlbum(ctx, info.AlbumID)
	if err != nil {
		return "", err
	}

	repaired, skipped := 0, 0
	for _, song := range album.Songs {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		file := findMusicFile(files, song.Artists, song.Name)
		if file == "" {
			skipped++
			continue
		}
		if tag.HasCMUSICID(file) {
			skipped++
			continue
		}
		meta, err := s.Music.Metadata.Get(ctx, song.ID, s.Cfg.Download.DefaultQuality)
		if err != nil || meta.Lyric == "" {
			if s.Logger != nil {
				s.Logger.Warn("未获取到完整歌词", "song", song.Name, "error", err)
			}
			skipped++
			continue
		}
		if err := tag.WriteLyrics(file, meta.Lyric, 5000); err != nil {
			return "", err
		}
		if err := tag.WriteCMUSICID(file, song.ID); err != nil {
			return "", err
		}
		stem := strings.TrimSuffix(file, filepath.Ext(file))
		if err := os.WriteFile(stem+".lrc", []byte(meta.Lyric), 0o644); err != nil && s.Logger != nil {
			s.Logger.Warn("写入 LRC 失败", "song", song.Name, "error", err)
		}
		repaired++
		if s.Logger != nil {
			s.Logger.Info("歌词已修复", "song", song.Name, "file", filepath.Base(file))
		}
		if err := httpclient.RandomDelay(ctx, 2.0, 2.0); err != nil {
			return "", err
		}
	}

	archived := archiveLyrics(folder)
	return fmt.Sprintf("修复 %d 首，跳过 %d 首，归档 %d 个 LRC", repaired, skipped, archived), nil
}

// archiveLyrics 把 .lrc 移入 .delete/，与源项目保持一致。
func archiveLyrics(folder string) int {
	files := FindLyricsFiles(folder)
	if len(files) == 0 {
		return 0
	}
	target := filepath.Join(folder, ".delete")
	if err := os.MkdirAll(target, 0o755); err != nil {
		return 0
	}
	moved := 0
	for _, file := range files {
		dest := filepath.Join(target, filepath.Base(file))
		if err := os.Rename(file, dest); err == nil {
			moved++
		}
	}
	return moved
}
