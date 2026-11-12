package library

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go-music-tools/internal/tag"
)

// FixCover 检查专辑封面分辨率，低于阈值时重新下载；覆盖前把旧封面备份到 .delete/。
func (s *Service) FixCover(ctx context.Context, folder string, force bool) (string, error) {
	threshold := s.Cfg.Advanced.CoverMinResolution
	cover := FindCoverImage(folder)
	if !force && cover != "" {
		data, err := os.ReadFile(cover)
		if err == nil {
			w, h := tag.ImageSize(data)
			if threshold > 0 && w >= threshold && h >= threshold {
				return fmt.Sprintf("封面分辨率 %dx%d 达标，跳过", w, h), nil
			}
			if s.Logger != nil {
				s.Logger.Info("封面分辨率不足，准备重下", "folder", filepath.Base(folder), "width", w, "height", h)
			}
		}
	}

	info, err := LoadAlbumInfo(folder)
	if err != nil {
		return "", err
	}
	if info.AlbumID == "" {
		return "", fmt.Errorf("album_info.txt 中缺少 Album id，无法获取专辑封面")
	}
	album, err := s.Music.GetAlbum(ctx, info.AlbumID)
	if err != nil {
		return "", err
	}
	coverURL := strings.SplitN(album.CoverURL, "?", 2)[0]
	if coverURL == "" {
		return "", fmt.Errorf("专辑 %s 未返回封面地址", info.AlbumID)
	}
	data, _, err := s.Music.HTTP.DownloadBytes(ctx, coverURL, nil)
	if err != nil {
		return "", err
	}
	if len(data) == 0 {
		return "", fmt.Errorf("下载到的封面为空")
	}

	if cover != "" {
		backupDir := filepath.Join(folder, ".delete")
		_ = os.MkdirAll(backupDir, 0o755)
		backup := filepath.Join(backupDir, fmt.Sprintf("cover_backup_%s.jpg", time.Now().Format("20060102-150405")))
		if old, err := os.ReadFile(cover); err == nil {
			_ = os.WriteFile(backup, old, 0o644)
		}
		_ = os.Remove(cover)
	}
	target := filepath.Join(folder, "cover.jpg")
	if err := os.WriteFile(target, data, 0o644); err != nil {
		return "", err
	}
	w, h := tag.ImageSize(data)
	return fmt.Sprintf("封面已重下（%dx%d，%s）", w, h, tag.FormatFileSize(int64(len(data)))), nil
}
