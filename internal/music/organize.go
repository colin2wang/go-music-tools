package music

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"go-music-tools/internal/api"
	"go-music-tools/internal/config"
	"go-music-tools/internal/httpclient"
	"go-music-tools/internal/model"
	"go-music-tools/internal/task"
)

// Store 由 internal/db.DB 实现，同时承担元数据限流时间戳与备用通道缓存。
type Store interface {
	httpclient.MetadataIntervalStore
	api.MusicInfoCache
}

type Service struct {
	*Downloader

	Cfg      config.Config
	Root     string
	HTTP     *httpclient.Client
	NetEase  *api.NetEaseClient
	Metadata *MetadataService
	Logger   *slog.Logger
}

func New(cfg config.Config, root string, hc *httpclient.Client, logger *slog.Logger, store Store) *Service {
	netease := api.NewNetEase(cfg, hc, logger)
	next := api.NewNextMusic(cfg, hc, logger, store)
	return &Service{
		Cfg:     cfg,
		Root:    root,
		HTTP:    hc,
		NetEase: netease,
		Logger:  logger,
		Downloader: &Downloader{
			Cfg: cfg, HTTP: hc, Logger: logger,
		},
		Metadata: &MetadataService{
			Cfg: cfg, NetEase: netease, Next: next, Store: store, Logger: logger,
		},
	}
}

func (s *Service) resolve(path string) string {
	if path == "" {
		return s.Root
	}
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(s.Root, path)
}

func (s *Service) DownloadDir() string {
	return s.resolve(s.Cfg.Download.DefaultDir)
}

func (s *Service) OODownloadDir() string {
	return s.resolve(s.Cfg.Advanced.OOMusicDownloadDir)
}

func report(r *task.Reporter, name, status, message string) {
	if r != nil {
		r.Item(name, status, message)
	}
}

func (s *Service) GetAlbum(ctx context.Context, id string) (*model.AlbumMeta, error) {
	return s.NetEase.ParseAlbum(ctx, api.ExtractResourceID(id))
}

func (s *Service) GetPlaylist(ctx context.Context, id string) (*model.PlaylistMeta, error) {
	return s.NetEase.ParsePlaylist(ctx, api.ExtractResourceID(id))
}

func (s *Service) Search(ctx context.Context, keyword string) ([]model.SearchSong, error) {
	return s.NetEase.Search(ctx, keyword)
}

// DownloadSong 下载单曲到默认下载目录，不带曲目序号。
func (s *Service) DownloadSong(ctx context.Context, songID, level string, r *task.Reporter) error {
	id := api.ExtractResourceID(songID)
	if r != nil {
		r.Stage("获取歌曲信息")
	}
	meta, err := s.Metadata.Get(ctx, id, level)
	if err != nil {
		report(r, id, "failed", err.Error())
		return err
	}
	if r != nil {
		r.Stage("下载歌曲")
	}
	result, err := s.DownloadSongAndResources(ctx, *meta, s.DownloadDir(), 0)
	if err != nil {
		report(r, meta.Name, "failed", err.Error())
		return err
	}
	status := "success"
	if result.Skipped {
		status = "skipped"
	}
	report(r, meta.Name, status, result.Path)
	return nil
}

// PrepareAlbumFolder 生成 "{artist} - {album} ({date})" 目录、说明文件、ID 文件与封面，
// 返回实际存放音频的子目录（受 download.sub_dir 控制）。
func (s *Service) PrepareAlbumFolder(ctx context.Context, album *model.AlbumMeta) (string, error) {
	replacement := s.Cfg.Filename.IllegalCharReplacement
	name := fmt.Sprintf("%s - %s", CleanFilename(album.Artist, replacement), CleanFilename(album.Name, replacement))
	if album.PublishDate != "" {
		name += " (" + album.PublishDate + ")"
	}
	root := filepath.Join(s.DownloadDir(), name)
	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", err
	}

	var info strings.Builder
	info.WriteString(fmt.Sprintf("Album: %s\n", album.Name))
	info.WriteString(fmt.Sprintf("Artist: %s\n", album.Artist))
	info.WriteString(fmt.Sprintf("Album id: %s\n", album.AlbumID))
	info.WriteString(fmt.Sprintf("Publish date: %s\n", album.PublishDate))
	info.WriteString(fmt.Sprintf("Song count: %d\n", album.SongCount))
	info.WriteString(strings.Repeat("=", 40) + "\n")
	info.WriteString("Description:\n")
	info.WriteString(album.Description + "\n")
	if err := writeText(filepath.Join(root, "album_info.txt"), info.String()); err != nil {
		return "", err
	}
	if album.AlbumID != "" {
		if err := writeText(filepath.Join(root, album.AlbumID+".txt"), album.Name+"\n"); err != nil {
			return "", err
		}
	}
	if album.CoverURL != "" {
		cover := strings.SplitN(album.CoverURL, "?", 2)[0]
		data, _, err := s.HTTP.DownloadBytes(ctx, cover, nil)
		if err != nil {
			if s.Logger != nil {
				s.Logger.Warn("下载专辑封面失败", "album", album.Name, "error", err)
			}
		} else {
			if err := os.WriteFile(filepath.Join(root, "cover.jpg"), data, 0o644); err != nil {
				return "", err
			}
		}
	}

	songDir := root
	if s.Cfg.Download.SubDir != "" {
		songDir = filepath.Join(root, s.Cfg.Download.SubDir)
	}
	if err := os.MkdirAll(songDir, 0o755); err != nil {
		return "", err
	}
	return songDir, nil
}

func (s *Service) PreparePlaylistFolder(_ context.Context, playlist *model.PlaylistMeta) (string, error) {
	replacement := s.Cfg.Filename.IllegalCharReplacement
	name := fmt.Sprintf("%s - %s", CleanFilename(playlist.Creator, replacement), CleanFilename(playlist.Name, replacement))
	root := filepath.Join(s.DownloadDir(), name)
	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", err
	}

	var info strings.Builder
	info.WriteString(fmt.Sprintf("Playlist: %s\n", playlist.Name))
	info.WriteString(fmt.Sprintf("Creator: %s\n", playlist.Creator))
	info.WriteString(fmt.Sprintf("Playlist id: %s\n", playlist.PlaylistID))
	info.WriteString(fmt.Sprintf("Song count: %d\n", playlist.SongCount))
	info.WriteString(strings.Repeat("=", 40) + "\n")
	info.WriteString("Description:\n")
	info.WriteString(playlist.Description + "\n")
	for i, song := range playlist.Songs {
		info.WriteString(fmt.Sprintf("%d. %s - %s\n", i+1, song.Artists, song.Name))
	}
	if err := writeText(filepath.Join(root, "playlist_info.txt"), info.String()); err != nil {
		return "", err
	}
	if playlist.PlaylistID != "" {
		if err := writeText(filepath.Join(root, playlist.PlaylistID+".txt"), playlist.Name+"\n"); err != nil {
			return "", err
		}
	}
	songDir := root
	if s.Cfg.Download.SubDir != "" {
		songDir = filepath.Join(root, s.Cfg.Download.SubDir)
	}
	if err := os.MkdirAll(songDir, 0o755); err != nil {
		return "", err
	}
	return songDir, nil
}

func (s *Service) downloadTracks(ctx context.Context, songs []model.SongDetail, dir string, level string,
	withIndex bool, r *task.Reporter) error {

	for _, song := range songs {
		if err := ctx.Err(); err != nil {
			return err
		}
		if r != nil {
			r.Stage(fmt.Sprintf("下载 %d/%d：%s", song.Index, len(songs), song.Name))
		}
		meta, err := s.Metadata.Get(ctx, song.ID, level)
		if err != nil {
			if s.Logger != nil {
				s.Logger.Error("获取曲目元数据失败", "song_id", song.ID, "name", song.Name, "error", err)
			}
			report(r, song.Name, "failed", err.Error())
			continue
		}
		index := 0
		if withIndex {
			index = song.Index
		}
		result, err := s.DownloadSongAndResources(ctx, *meta, dir, index)
		if err != nil {
			if s.Logger != nil {
				s.Logger.Error("下载曲目失败", "song_id", song.ID, "name", song.Name, "error", err)
			}
			report(r, song.Name, "failed", err.Error())
			continue
		}
		status := "success"
		if result.Skipped {
			status = "skipped"
		}
		report(r, song.Name, status, result.Path)
	}
	return ctx.Err()
}

// DownloadAlbum 支持按编号筛选曲目，出错的曲目只记日志不中断整批。
func (s *Service) DownloadAlbum(ctx context.Context, albumID string, indexes []int, level string, r *task.Reporter) error {
	if r != nil {
		r.Stage("获取专辑曲目")
	}
	album, err := s.GetAlbum(ctx, albumID)
	if err != nil {
		return err
	}
	selected := SelectSongs(album.Songs, indexes)
	if len(selected) == 0 {
		return fmt.Errorf("专辑 %s 没有可下载的曲目", albumID)
	}
	songs := make([]model.SongDetail, 0, len(selected))
	for _, item := range selected {
		songs = append(songs, item.Item)
	}
	if r != nil {
		r.SetTotal(len(songs))
	}
	if s.Logger != nil {
		s.Logger.Info("开始下载专辑", "album", album.Name, "tracks", len(songs))
	}
	if r != nil {
		r.Stage("准备专辑目录")
	}
	dir, err := s.PrepareAlbumFolder(ctx, album)
	if err != nil {
		return err
	}
	return s.downloadTracks(ctx, songs, dir, level, true, r)
}

func (s *Service) DownloadPlaylist(ctx context.Context, playlistID string, indexes []int, level string, r *task.Reporter) error {
	if r != nil {
		r.Stage("获取歌单曲目")
	}
	playlist, err := s.GetPlaylist(ctx, playlistID)
	if err != nil {
		return err
	}
	selected := SelectSongs(playlist.Songs, indexes)
	if len(selected) == 0 {
		return fmt.Errorf("歌单 %s 没有可下载的曲目", playlistID)
	}
	songs := make([]model.SongDetail, 0, len(selected))
	for _, item := range selected {
		songs = append(songs, item.Item)
	}
	if r != nil {
		r.SetTotal(len(songs))
	}
	if s.Logger != nil {
		s.Logger.Info("开始下载歌单", "playlist", playlist.Name, "tracks", len(songs))
	}
	dir, err := s.PreparePlaylistFolder(ctx, playlist)
	if err != nil {
		return err
	}
	return s.downloadTracks(ctx, songs, dir, level, false, r)
}
