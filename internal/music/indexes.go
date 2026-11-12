package music

import (
	"sort"
	"strconv"
	"strings"
)

// ParseIndexes 解析 "1 2 3-6 9-11"、逗号与中文逗号混排的曲目编号，排序去重，仅保留正整数。
func ParseIndexes(raw string) []int {
	normalized := strings.NewReplacer(",", " ", "，", " ", ";", " ", "；", " ").Replace(raw)
	seen := map[int]bool{}
	for _, token := range strings.Fields(normalized) {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		if strings.Contains(token, "-") {
			parts := strings.SplitN(token, "-", 2)
			start, errA := strconv.Atoi(strings.TrimSpace(parts[0]))
			end, errB := strconv.Atoi(strings.TrimSpace(parts[1]))
			if errA != nil || errB != nil || start <= 0 {
				continue
			}
			if end < start {
				start, end = end, start
			}
			if end-start > 5000 {
				end = start + 5000
			}
			for i := start; i <= end; i++ {
				seen[i] = true
			}
			continue
		}
		if n, err := strconv.Atoi(token); err == nil && n > 0 {
			seen[n] = true
		}
	}
	out := make([]int, 0, len(seen))
	for n := range seen {
		out = append(out, n)
	}
	sort.Ints(out)
	return out
}

// SelectSongs 按 1-based 编号筛选曲目；空编号表示全部。
func SelectSongs[T any](songs []T, indexes []int) []struct {
	Index int
	Item  T
} {
	if len(indexes) == 0 {
		out := make([]struct {
			Index int
			Item  T
		}, 0, len(songs))
		for i, song := range songs {
			out = append(out, struct {
				Index int
				Item  T
			}{Index: i + 1, Item: song})
		}
		return out
	}
	selected := map[int]bool{}
	for _, n := range indexes {
		selected[n] = true
	}
	out := make([]struct {
		Index int
		Item  T
	}, 0, len(selected))
	for i, song := range songs {
		if selected[i+1] {
			out = append(out, struct {
				Index int
				Item  T
			}{Index: i + 1, Item: song})
		}
	}
	return out
}
