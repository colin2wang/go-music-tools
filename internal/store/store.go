// Package store 提供进程内的偏好、缓存与任务历史存储。
// 数据仅本次运行有效，应用重启后清空。
package store

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	gocache "github.com/patrickmn/go-cache"
	"gopkg.in/yaml.v3"

	"go-music-tools/internal/model"
)

const (
	keyLastDownloadConfig = "last_download_config"
	keyMetadataTimestamp  = "last_get_metadata_timestamp"
)

// Store 用 go-cache 承载原 SQLite 的全部职责：上次下载偏好、
// 元数据限流时间戳、歌曲信息与播放地址缓存、任务历史。
// 其中偏好与限流时间戳持久化到 YAML 状态文件（如 state.yml），
// 缓存与任务历史仅本次运行有效。
type Store struct {
	path string

	prefs *gocache.Cache
	info  *gocache.Cache
	url   *gocache.Cache
	ne    *gocache.Cache
	oo    *gocache.Cache

	mu     sync.Mutex
	tasks  []model.TaskRecord
	nextID int64
}

// stateFile 是状态文件的磁盘格式，只含需要跨启动保留的配置项。
type stateFile struct {
	LastDownloadConfig string  `yaml:"last_download_config"`
	MetadataTimestamp  float64 `yaml:"metadata_timestamp"`
}

// New 创建存储；path 非空时加载该 YAML 状态文件并在写入偏好时回存。
func New(path string) *Store {
	s := &Store{
		path:  path,
		prefs: gocache.New(gocache.NoExpiration, 10*time.Minute),
		info:  gocache.New(gocache.NoExpiration, 10*time.Minute),
		url:   gocache.New(gocache.NoExpiration, 10*time.Minute),
		ne:    gocache.New(gocache.NoExpiration, 10*time.Minute),
		oo:    gocache.New(gocache.NoExpiration, 10*time.Minute),
	}
	if path == "" {
		return s
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return s // 首次启动或文件缺失时从空状态开始
	}
	var sf stateFile
	if err := yaml.Unmarshal(data, &sf); err != nil {
		return s // 状态文件损坏时按空状态处理，不影响启动
	}
	if sf.LastDownloadConfig != "" {
		s.prefs.Set(keyLastDownloadConfig, sf.LastDownloadConfig, gocache.DefaultExpiration)
	}
	if sf.MetadataTimestamp > 0 {
		s.prefs.Set(keyMetadataTimestamp,
			strconv.FormatFloat(sf.MetadataTimestamp, 'f', 6, 64), gocache.DefaultExpiration)
	}
	return s
}

// save 状态文件以「临时文件 → 原子 rename」方式写入，避免半文件。
func (s *Store) saveStateLocked() error {
	if s.path == "" {
		return nil
	}
	sf := stateFile{}
	if raw, ok := s.getPref(keyLastDownloadConfig); ok {
		sf.LastDownloadConfig = raw
	}
	if ts, ok := s.getPref(keyMetadataTimestamp); ok {
		sf.MetadataTimestamp, _ = strconv.ParseFloat(ts, 64)
	}
	data, err := yaml.Marshal(sf)
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// --- 偏好 ---------------------------------------------------------------

func (s *Store) getPref(key string) (string, bool) {
	raw, ok := s.prefs.Get(key)
	if !ok {
		return "", false
	}
	return raw.(string), true
}

func (s *Store) setPref(key, value string) {
	s.prefs.Set(key, value, gocache.DefaultExpiration)
}

// GetLastDownloadConfig 返回界面上次使用的下载参数 JSON。
func (s *Store) GetLastDownloadConfig() (string, error) {
	if raw, ok := s.getPref(keyLastDownloadConfig); ok {
		return raw, nil
	}
	return "", nil
}

func (s *Store) SetLastDownloadConfig(raw string) error {
	s.setPref(keyLastDownloadConfig, raw)
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveStateLocked()
}

// GetMetadataTimestamp 用于跨任务的元数据请求限流（进程内）。
func (s *Store) GetMetadataTimestamp() (float64, error) {
	if raw, ok := s.getPref(keyMetadataTimestamp); ok {
		return strconv.ParseFloat(raw, 64)
	}
	return 0, nil
}

func (s *Store) SetMetadataTimestamp(ts float64) error {
	s.setPref(keyMetadataTimestamp, strconv.FormatFloat(ts, 'f', 6, 64))
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveStateLocked()
}

// --- 歌曲信息与播放地址缓存 ----------------------------------------------

func (s *Store) GetMusicInfo(musicID string) (*model.MusicInfo, bool, error) {
	v, ok := s.info.Get(musicID)
	if !ok {
		return nil, false, nil
	}
	info := v.(model.MusicInfo)
	return &info, true, nil
}

// UpsertMusicInfo 只覆盖非空字段，避免歌词接口先写入时把标题清空。
func (s *Store) UpsertMusicInfo(info model.MusicInfo) error {
	if old, ok := s.info.Get(info.MusicID); ok {
		prev := old.(model.MusicInfo)
		if info.Artists == "" {
			info.Artists = prev.Artists
		}
		if info.Title == "" {
			info.Title = prev.Title
		}
		if info.AlbumName == "" {
			info.AlbumName = prev.AlbumName
		}
		if info.CoverURL == "" {
			info.CoverURL = prev.CoverURL
		}
		if info.Lyrics == "" {
			info.Lyrics = prev.Lyrics
		}
		if info.Duration == 0 {
			info.Duration = prev.Duration
		}
	}
	s.info.Set(info.MusicID, info, gocache.DefaultExpiration)
	return nil
}

// urlCacheTTL 是播放地址缓存的存活时间。播放地址通常带时效签名，
// 过期后再用会导致下载 403/404，因此按常见有效期量级设短 TTL。
const urlCacheTTL = 20 * time.Minute

func (s *Store) GetMusicURL(musicID, quality string) (string, bool, error) {
	v, ok := s.url.Get(urlKey(musicID, quality))
	if !ok {
		return "", false, nil
	}
	u := v.(string)
	return u, u != "", nil
}

func (s *Store) UpsertMusicURL(musicID, quality, url string) error {
	s.url.Set(urlKey(musicID, quality), url, urlCacheTTL)
	return nil
}

// DeleteMusicURL 使指定歌曲与音质的播放地址缓存失效（地址过期重取时调用）。
func (s *Store) DeleteMusicURL(musicID, quality string) {
	s.url.Delete(urlKey(musicID, quality))
}

// --- 网易云主通道歌曲缓存 -------------------------------------------------

func neKey(songID, level string) string {
	data, _ := json.Marshal([2]string{songID, level})
	return string(data)
}

// GetNetEaseSong 读取网易云主通道的整曲元数据缓存（含播放地址，随 TTL 一起过期）。
func (s *Store) GetNetEaseSong(songID, level string) (*model.SongMeta, bool) {
	v, ok := s.ne.Get(neKey(songID, level))
	if !ok {
		return nil, false
	}
	meta := v.(model.SongMeta)
	if meta.Name == "" || meta.URL == "" {
		return nil, false
	}
	return &meta, true
}

// UpsertNetEaseSong 写入网易云主通道缓存；只有标题与地址都非空的成功结果才入缓存。
func (s *Store) UpsertNetEaseSong(meta model.SongMeta, level string) error {
	if meta.Name == "" || meta.URL == "" {
		return nil
	}
	s.ne.Set(neKey(meta.ID, level), meta, urlCacheTTL)
	return nil
}

// DeleteNetEaseSong 使指定歌曲与音质的网易云缓存失效（地址过期重取时调用）。
func (s *Store) DeleteNetEaseSong(songID, level string) {
	s.ne.Delete(neKey(songID, level))
}

// --- OO Music 详情缓存 ----------------------------------------------------

func ooDetailKey(keyword, songMid string) string {
	data, _ := json.Marshal([2]string{keyword, songMid})
	return string(data)
}

// OODetailValue 是 OO Music 详情缓存的存储结构，同时保留原始响应 JSON。
type OODetailValue struct {
	Detail map[string]any
	Raw    string
}

// GetOODetail 读取 OO Music 详情缓存（含播放地址，随 TTL 一起过期）。
func (s *Store) GetOODetail(keyword, songMid string) (map[string]any, bool) {
	detail, _, ok := s.getOODetailValue(keyword, songMid)
	return detail, ok
}

// GetOODetailRaw 读取 OO Music 详情缓存及其原始响应 JSON。
func (s *Store) GetOODetailRaw(keyword, songMid string) (map[string]any, string, bool) {
	return s.getOODetailValue(keyword, songMid)
}

func (s *Store) getOODetailValue(keyword, songMid string) (map[string]any, string, bool) {
	v, ok := s.oo.Get(ooDetailKey(keyword, songMid))
	if !ok {
		return nil, "", false
	}
	val, ok := v.(OODetailValue)
	if !ok || len(val.Detail) == 0 {
		return nil, "", false
	}
	return val.Detail, val.Raw, true
}

// UpsertOODetail 写入 OO Music 详情缓存，并保留原始响应 JSON；空详情不入缓存。
// TTL 与播放地址一致：详情内含时效性地址，过期后整体重新拉取。
func (s *Store) UpsertOODetail(keyword, songMid string, detail map[string]any, raw string) error {
	if len(detail) == 0 {
		return nil
	}
	s.oo.Set(ooDetailKey(keyword, songMid), OODetailValue{Detail: detail, Raw: raw}, urlCacheTTL)
	return nil
}

// DeleteOODetail 使指定 OO Music 歌曲的详情缓存失效（地址过期重取时调用）。
func (s *Store) DeleteOODetail(keyword, songMid string) {
	s.oo.Delete(ooDetailKey(keyword, songMid))
}

func urlKey(musicID, quality string) string {
	data, _ := json.Marshal([2]string{musicID, quality})
	return string(data)
}

// --- 任务历史 ------------------------------------------------------------

func (s *Store) AddTaskHistory(rec model.TaskRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if rec.CreatedAt == 0 {
		rec.CreatedAt = time.Now().Unix()
	}
	s.nextID++
	rec.ID = s.nextID
	s.tasks = append(s.tasks, rec)
	return nil
}

func (s *Store) ListTaskHistory(limit int) ([]model.TaskRecord, error) {
	if limit < 1 || limit > 200 {
		limit = 20
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]model.TaskRecord, len(s.tasks))
	copy(out, s.tasks)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].CreatedAt != out[j].CreatedAt {
			return out[i].CreatedAt > out[j].CreatedAt
		}
		return out[i].ID > out[j].ID
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// --- 缓存检视 ------------------------------------------------------------

// Region 各缓存区的标识，供缓存检视页面使用。
const (
	RegionPrefs     = "prefs"
	RegionMusicInfo = "music_info"
	RegionMusicURL  = "music_url"
	RegionNetEase   = "netease_song"
	RegionOODetail  = "oo_detail"
)

// Entry 是缓存检视页面展示的单条缓存内容。
type Entry struct {
	Key     string `json:"key"`
	Preview string `json:"preview"`
	Bytes   int    `json:"bytes"`
}

func regionCache(s *Store, region string) *gocache.Cache {
	switch region {
	case RegionPrefs:
		return s.prefs
	case RegionMusicInfo:
		return s.info
	case RegionMusicURL:
		return s.url
	case RegionNetEase:
		return s.ne
	case RegionOODetail:
		return s.oo
	}
	return nil
}

// List 返回指定缓存区的全部条目，按 key 排序；region 无效时返回 nil。
func (s *Store) List(region string) []Entry {
	c := regionCache(s, region)
	if c == nil {
		return nil
	}
	items := c.Items()
	out := make([]Entry, 0, len(items))
	for key, item := range items {
		out = append(out, Entry{
			Key:     key,
			Preview: previewValue(item.Object),
			Bytes:   len(previewValue(item.Object)),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}

// Count 返回指定缓存区的条目数量；region 无效时返回 -1。
func (s *Store) Count(region string) int {
	c := regionCache(s, region)
	if c == nil {
		return -1
	}
	return c.ItemCount()
}

// Get 读取指定缓存区的一个条目原始值；region 或 key 不存在时返回 false。
func (s *Store) Get(region, key string) (any, bool) {
	c := regionCache(s, region)
	if c == nil {
		return nil, false
	}
	return c.Get(key)
}

// Delete 删除指定缓存区的一个条目；key 为空时清空整个区域。
func (s *Store) Delete(region, key string) {
	c := regionCache(s, region)
	if c == nil {
		return
	}
	if key == "" {
		c.Flush()
		return
	}
	c.Delete(key)
}

// previewValue 生成缓存值的单行预览文本。
func previewValue(v any) string {
	switch val := v.(type) {
	case string:
		return strings.ReplaceAll(val, "\n", " ")
	case model.MusicInfo:
		return val.Artists + " - " + val.Title
	case model.SongMeta:
		return val.Artists + " - " + val.Name + " (" + val.UsedQuality + ")"
	default:
		return fmt.Sprintf("%v", v)
	}
}
