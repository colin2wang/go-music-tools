package bindings

import (
	"encoding/json"
	"fmt"

	"go-music-tools/internal/store"
)

// CacheRegionStat 是一个缓存区的概要信息，供缓存检视页面汇总展示。
type CacheRegionStat struct {
	Region string `json:"region"`
	Label  string `json:"label"`
	Count  int    `json:"count"`
}

// CacheEntry 是缓存区中的一条缓存内容。
type CacheEntry struct {
	Key     string `json:"key"`
	Preview string `json:"preview"`
	Bytes   int    `json:"bytes"`
}

var cacheRegions = []struct {
	Region string
	Label  string
}{
	{store.RegionPrefs, "偏好（上次下载参数、限流时间戳）"},
	{store.RegionMusicInfo, "歌曲信息缓存"},
	{store.RegionMusicURL, "播放地址缓存"},
	{store.RegionNetEase, "网易云歌曲缓存（主通道）"},
	{store.RegionOODetail, "OO Music 详情缓存"},
	{"oo_search", "OO Music 搜索缓存"},
}

// GetCacheStats 返回各缓存区的条目数量。
func (s *Service) GetCacheStats() []CacheRegionStat {
	out := make([]CacheRegionStat, 0, len(cacheRegions))
	for _, r := range cacheRegions {
		count := 0
		if r.Region == "oo_search" {
			count = len(s.ooService().ListSearchCache())
		} else {
			count = s.deps.Cache.Count(r.Region)
		}
		out = append(out, CacheRegionStat{Region: r.Region, Label: r.Label, Count: count})
	}
	return out
}

// ListCache 返回指定缓存区的全部条目；region 无效时返回错误。
func (s *Service) ListCache(region string) ([]CacheEntry, error) {
	if region == "oo_search" {
		list := s.ooService().ListSearchCache()
		out := make([]CacheEntry, 0, len(list))
		for _, item := range list {
			out = append(out, CacheEntry{
				Key:     item.Keyword,
				Preview: fmt.Sprintf("%d 条结果", item.Count),
				Bytes:   item.Count,
			})
		}
		return out, nil
	}
	list := s.deps.Cache.List(region)
	if list == nil {
		return nil, nil
	}
	out := make([]CacheEntry, 0, len(list))
	for _, item := range list {
		out = append(out, CacheEntry{Key: item.Key, Preview: item.Preview, Bytes: item.Bytes})
	}
	return out, nil
}

// CacheDetail 是单个缓存条目的详情视图：格式化后的内容与原始响应 JSON。
type CacheDetail struct {
	Formatted string `json:"formatted"`
	Raw       string `json:"raw"`
}

// GetCacheDetail 返回指定缓存区某条目的格式化内容；
// OO Music 详情缓存额外返回缓存的原始响应 JSON（raw 为空表示该区无原始响应）。
func (s *Service) GetCacheDetail(region, key string) (CacheDetail, error) {
	if region == "oo_search" {
		songs := s.ooService().GetSearchCacheSongs(key)
		formatted, err := json.MarshalIndent(songs, "", "  ")
		if err != nil {
			return CacheDetail{}, err
		}
		return CacheDetail{Formatted: string(formatted)}, nil
	}
	value, ok := s.deps.Cache.Get(region, key)
	if !ok {
		return CacheDetail{}, fmt.Errorf("缓存条目不存在: %s", key)
	}
	formatted, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return CacheDetail{}, err
	}
	detail := CacheDetail{Formatted: string(formatted)}
	if region == store.RegionOODetail {
		if val, ok := value.(store.OODetailValue); ok {
			detail.Raw = val.Raw
		}
	}
	return detail, nil
}

// ClearCache 清空指定缓存区；key 非空时仅删除该条目。返回删除的条目数。
func (s *Service) ClearCache(region, key string) (int, error) {
	if region == "oo_search" {
		return s.ooService().ClearSearchCache(key), nil
	}
	switch region {
	case store.RegionPrefs, store.RegionMusicInfo, store.RegionMusicURL, store.RegionNetEase, store.RegionOODetail:
		count := s.deps.Cache.Count(region)
		s.deps.Cache.Delete(region, key)
		if key != "" {
			return 1, nil
		}
		return count, nil
	}
	return 0, nil
}
