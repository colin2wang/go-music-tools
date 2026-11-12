package bindings

import (
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

// ClearCache 清空指定缓存区；key 非空时仅删除该条目。返回删除的条目数。
func (s *Service) ClearCache(region, key string) (int, error) {
	if region == "oo_search" {
		return s.ooService().ClearSearchCache(key), nil
	}
	switch region {
	case store.RegionPrefs, store.RegionMusicInfo, store.RegionMusicURL:
		count := s.deps.Cache.Count(region)
		s.deps.Cache.Delete(region, key)
		if key != "" {
			return 1, nil
		}
		return count, nil
	}
	return 0, nil
}
