package music

import (
	"reflect"
	"testing"
)

func TestParseIndexes(t *testing.T) {
	cases := []struct {
		raw  string
		want []int
	}{
		{"1 2 3", []int{1, 2, 3}},
		{"1,2,3", []int{1, 2, 3}},
		{"1，2，3", []int{1, 2, 3}},
		{"3-6", []int{3, 4, 5, 6}},
		{"1 2 3-6 9-11", []int{1, 2, 3, 4, 5, 6, 9, 10, 11}},
		{"5 5 3 0 -2 abc", []int{3, 5}},
		{"", []int{}},
		{"6-3", []int{3, 4, 5, 6}},
	}
	for _, tc := range cases {
		got := ParseIndexes(tc.raw)
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("ParseIndexes(%q) = %v, want %v", tc.raw, got, tc.want)
		}
	}
}

func TestSelectSongs(t *testing.T) {
	songs := []string{"a", "b", "c", "d"}
	all := SelectSongs(songs, nil)
	if len(all) != 4 || all[0].Index != 1 || all[3].Item != "d" {
		t.Fatalf("全选结果不正确: %+v", all)
	}
	part := SelectSongs(songs, []int{2, 4})
	if len(part) != 2 || part[0].Item != "b" || part[1].Item != "d" {
		t.Fatalf("筛选结果不正确: %+v", part)
	}
}
