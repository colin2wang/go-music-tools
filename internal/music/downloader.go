package music

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"go-music-tools/internal/config"
	"go-music-tools/internal/httpclient"
	"go-music-tools/internal/model"
	"go-music-tools/internal/tag"
)

type DownloadResult struct {
	Path    string
	Skipped bool
	Size    int64
	Quality string
}

type Downloader struct {
	Cfg    config.Config
	HTTP   *httpclient.Client
	Logger *slog.Logger

	coverMu    sync.Mutex
	coverCache map[string][]byte // 专辑内封面字节去重：同一封面 URL 只下载一次
}

func coverURL(raw string, size int) string {
	if raw == "" || size <= 0 {
		return raw
	}
	param := fmt.Sprintf("param=%dx%d", size, size)
	separator := "?"
	if strings.Contains(raw, "?") {
		separator = "&"
	}
	return raw + separator + param
}

func writeText(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

// DownloadSongAndResources 完成单曲的下载、LRC、封面与标签写入全流程。
func (d *Downloader) DownloadSongAndResources(ctx context.Context, meta model.SongMeta, dir string, index int) (*DownloadResult, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	ext := ExtractExtension(meta.URL)
	base := EnsureWritableName(dir, BuildFilename(d.Cfg, meta, index), ext)
	dest := filepath.Join(dir, base+ext)

	if d.Cfg.Download.SkipExisting {
		if stat, err := os.Stat(dest); err == nil && stat.Size() > 0 {
			if d.Logger != nil {
				d.Logger.Info("文件已存在，跳过下载", "file", base+ext)
			}
			return &DownloadResult{Path: dest, Skipped: true, Size: stat.Size(), Quality: meta.UsedQuality}, nil
		}
	}

	if err := httpclient.RandomDelay(ctx, d.Cfg.Network.APIDelayMin, d.Cfg.Network.RandomDelayMax); err != nil {
		return nil, err
	}

	result := &DownloadResult{Path: dest, Quality: meta.UsedQuality}
	progress := httpclient.ThrottledProgress(d.Logger, base+ext)
	_, err := d.HTTP.Download(ctx, meta.URL, dest, nil, func(written, total int64) {
		result.Size = written
		if progress != nil {
			progress(written, total)
		}
	})
	if err != nil {
		return nil, fmt.Errorf("下载音频失败: %w", err)
	}
	if stat, err := os.Stat(dest); err == nil {
		result.Size = stat.Size()
	}
	if d.Logger != nil {
		d.Logger.Info("音频下载完成", "file", base+ext, "size", tag.FormatFileSize(result.Size), "quality", meta.UsedQuality)
	}

	stem := filepath.Join(dir, base)
	lyrics := meta.Lyric
	if !d.Cfg.Metadata.WriteLyrics {
		lyrics = ""
	}
	if d.Cfg.Metadata.SaveLRCFile && lyrics != "" {
		if err := writeText(stem+".lrc", lyrics); err != nil {
			d.warn("写入 LRC 文件失败", err)
		}
	}

	var cover []byte
	coverMIME := ""
	if d.Cfg.Metadata.WriteCover && meta.Pic != "" {
		assetURL := coverURL(meta.Pic, d.Cfg.Metadata.CoverSize)
		data, contentType, err := d.coverBytes(ctx, assetURL)
		if err != nil {
			d.warn("下载封面失败", err)
		} else if len(data) > 0 {
			cover = data
			coverMIME = tag.MimeFromData(data)
			if contentType != "" && strings.Contains(contentType, "png") {
				coverMIME = "image/png"
			}
			coverPath := stem + ".jpg"
			if coverMIME == "image/png" {
				coverPath = stem + ".png"
			}
			if err := os.WriteFile(coverPath, data, 0o644); err != nil {
				d.warn("保存封面失败", err)
			}
		}
	}

	tagLyrics := lyrics
	if d.Cfg.Metadata.MaxLyricLength > 0 && isMP4Ext(ext) {
		runes := []rune(tagLyrics)
		if len(runes) > d.Cfg.Metadata.MaxLyricLength {
			tagLyrics = string(runes[:d.Cfg.Metadata.MaxLyricLength])
		}
	}
	if !d.Cfg.Metadata.WriteMetadata {
		if len(cover) > 0 {
			if err := tag.WriteCover(dest, cover, coverMIME); err != nil {
				d.warn("单独嵌入封面失败", err)
			}
		}
		return result, nil
	}

	trackTags := tag.TrackTags{
		Title:       meta.Name,
		Artists:     tag.SplitArtists(meta.Artists),
		Album:       meta.Album,
		Lyrics:      tagLyrics,
		CMusicID:    meta.ID,
		TrackNumber: index,
		Year:        strconv.Itoa(time.Now().Year()),
		Cover:       cover,
		CoverMIME:   coverMIME,
	}
	if err := tag.Write(dest, trackTags); err != nil {
		d.warn("写入音频标签失败", err)
	} else if d.Logger != nil {
		d.Logger.Info("元数据与封面已写入", "file", base+ext, "cmusic_id", meta.ID)
	}
	return result, nil
}

func isMP4Ext(ext string) bool {
	switch strings.ToLower(ext) {
	case ".m4a", ".mp4", ".aac":
		return true
	default:
		return false
	}
}

func (d *Downloader) warn(message string, err error) {
	if d.Logger != nil {
		d.Logger.Warn(message, "error", err)
	}
}

// coverBytes 下载封面字节并做进程内去重：同一 URL（专辑内多首歌共用）只下载一次，
// 下载成功且字节数大于 0 才入缓存；失败不入缓存，下次仍会重试。
func (d *Downloader) coverBytes(ctx context.Context, assetURL string) ([]byte, string, error) {
	d.coverMu.Lock()
	if d.coverCache == nil {
		d.coverCache = make(map[string][]byte)
	}
	cached, ok := d.coverCache[assetURL]
	d.coverMu.Unlock()
	if ok {
		return cached, "", nil
	}
	data, contentType, err := d.HTTP.DownloadBytes(ctx, assetURL, nil)
	if err != nil {
		return nil, contentType, err
	}
	if len(data) > 0 {
		d.coverMu.Lock()
		d.coverCache[assetURL] = data
		d.coverMu.Unlock()
	}
	return data, contentType, nil
}
