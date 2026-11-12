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

func formatTrackIndex(index int, format string) string {
	switch format {
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

func renameCompanions(oldBase, newBase string) {
	for _, ext := range []string{".lrc", ".jpg", ".jpeg", ".png"} {
		old := oldBase + ext
		if _, err := os.Stat(old); err == nil {
			_ = os.Rename(old, newBase+ext)
		}
	}
}

// FixArtists 把含 / & 、 的艺术家标签拆成多值，并按 index_format 重建文件名。
func (s *Service) FixArtists(ctx context.Context, folder string, recursive bool) (string, error) {
	files, err := FindAudioFiles(folder, recursive)
	if err != nil {
		return "", err
	}
	changed, skipped := 0, 0
	for _, file := range files {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		current, err := tag.Read(file)
		if err != nil {
			skipped++
			continue
		}
		needsFix := false
		for _, artist := range current.Artists {
			if strings.ContainsAny(artist, "/&、") {
				needsFix = true
				break
			}
		}
		if !needsFix {
			skipped++
			continue
		}
		artists := tag.SplitArtists(strings.Join(current.Artists, ", "))
		if len(artists) == 0 {
			skipped++
			continue
		}
		if err := tag.WriteArtists(file, artists); err != nil {
			return "", err
		}

		ext := filepath.Ext(file)
		oldBase := strings.TrimSuffix(file, ext)
		stem := filepath.Base(oldBase)
		track, _, title := music.ParseMusicFilename(stem)
		if track == 0 {
			track = 1
		}
		joiner := s.Cfg.Filename.ArtistDelimiterReplacement
		if joiner == "" {
			joiner = ", "
		}
		newStem := fmt.Sprintf("%s - %s - %s",
			formatTrackIndex(track, s.Cfg.Download.IndexFormat),
			strings.Join(artists, joiner), title)
		newStem = music.CleanFilename(newStem, s.Cfg.Filename.IllegalCharReplacement)
		if newStem == stem {
			changed++
			continue
		}
		newBase := filepath.Join(filepath.Dir(file), newStem)
		if err := os.Rename(file, newBase+ext); err != nil {
			if s.Logger != nil {
				s.Logger.Warn("重建文件名失败", "file", stem, "error", err)
			}
			continue
		}
		renameCompanions(oldBase, newBase)
		changed++
		if s.Logger != nil {
			s.Logger.Info("艺术家标签与文件名已修复", "file", newStem+ext)
		}
	}
	return fmt.Sprintf("修复 %d 个文件，跳过 %d 个", changed, skipped), nil
}
