package music

import (
	"context"
	"fmt"
	"log/slog"

	"go-music-tools/internal/api"
	"go-music-tools/internal/config"
	"go-music-tools/internal/httpclient"
	"go-music-tools/internal/model"
)

type MetadataService struct {
	Cfg     config.Config
	NetEase *api.NetEaseClient
	Next    *api.NextMusicClient
	Store   httpclient.MetadataIntervalStore
	Logger  *slog.Logger
}

// Get 是元数据请求的唯一入口：跨任务限流 → 重试 → 音质校验 → 记录时间戳。
func (m *MetadataService) Get(ctx context.Context, songID, level string) (*model.SongMeta, error) {
	if err := httpclient.WaitMetadataInterval(ctx, m.Logger, m.Store, m.Cfg.Network.GetMetadataInterval); err != nil {
		return nil, err
	}
	defer func() { _ = httpclient.MarkMetadataFinish(m.Store) }()

	attempts := m.Cfg.Network.MaxRetries + 1
	if attempts < 1 {
		attempts = 1
	}
	var lastErr error = fmt.Errorf("未获取到歌曲 %s 的元数据", songID)
	for i := 0; i < attempts; i++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		var meta *model.SongMeta
		var err error
		if m.Cfg.Metadata.UseNextMusicTool && m.Next != nil {
			meta, err = m.Next.Resolve(ctx, songID, level)
		} else {
			meta, err = m.NetEase.ParseSong(ctx, songID, level)
		}
		if err != nil {
			lastErr = err
			if m.Logger != nil {
				m.Logger.Warn("元数据请求失败，准备重试", "song_id", songID, "attempt", i+1, "error", err)
			}
			continue
		}
		if meta.URL == "" {
			lastErr = fmt.Errorf("歌曲 %s 未返回下载地址", songID)
			continue
		}
		if i < attempts-1 && level != "" && meta.Level != "" && meta.Level != level {
			if m.Logger != nil {
				m.Logger.Info("返回音质与请求不符，重试", "song_id", songID, "requested", level, "got", meta.Level)
			}
			continue
		}
		return meta, nil
	}
	return nil, fmt.Errorf("获取歌曲 %s 元数据失败: %w", songID, lastErr)
}

// Invalidate 使两个通道的整曲缓存同时失效（播放地址过期重取时调用；
// 实际走哪个通道由配置决定，双双失效最简单也最安全）。
func (m *MetadataService) Invalidate(songID, level string) {
	if m.NetEase != nil {
		m.NetEase.InvalidateSong(songID, level)
	}
	if m.Next != nil {
		m.Next.InvalidateSong(songID, level)
	}
}
