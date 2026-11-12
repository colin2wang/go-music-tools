package api

import "testing"

func TestParseOOSearchArrayAndObject(t *testing.T) {
	arrayBody := []byte(`[{"song_title":"夜曲","singer_name":"周杰伦","song_mid":"mid001","pay":"0"}]`)
	songs, err := parseOOSearch(arrayBody)
	if err != nil {
		t.Fatal(err)
	}
	if len(songs) != 1 || songs[0].SongMid != "mid001" || songs[0].SongTitle != "夜曲" {
		t.Fatalf("数组响应解析错误: %+v", songs)
	}

	objectBody := []byte(`{"data":[{"song_name":"晴天","singer":"周杰伦","mid":"mid002","pay":1}]}`)
	songs, err = parseOOSearch(objectBody)
	if err != nil {
		t.Fatal(err)
	}
	if len(songs) != 1 || songs[0].SongTitle != "晴天" || songs[0].SongMid != "mid002" || songs[0].Pay != "1" {
		t.Fatalf("对象响应解析错误: %+v", songs)
	}
}

func TestOOInferExtension(t *testing.T) {
	cases := []struct {
		filename string
		quality  string
		want     string
	}{
		{"F000004abc.flac", "SQ", ".flac"},
		{"C600004abc.m4a", "HQ", ".m4a"},
		{"C400004abc.m4a", "standard", ".m4a"},
		{"C200004abc.m4a", "low", ".m4a"},
		{"", "SQ", ".flac"},
		{"", "low", ".m4a"},
	}
	for _, c := range cases {
		if got := OOInferExtension(c.filename, c.quality); got != c.want {
			t.Errorf("OOInferExtension(%q, %q) = %q, 期望 %q", c.filename, c.quality, got, c.want)
		}
	}
}

func ooTestDetail() map[string]any {
	return map[string]any{
		"song_name":        "稻香",
		"singer_name":      "周杰伦",
		"album_name":       "魔杰座",
		"song_mid":         "mid003",
		"song_play_time":   "223",
		"album_pic":        "https://example.com/cover.jpg",
		"song_lyric":       "[00:00.00]稻香",
		"lyric":            "还记得你说家是唯一的城堡",
		"language":         "国语",
		"song_play_url_sq": "https://example.com/f000.flac",
		"song_filename_sq": "F000003xyz.flac",
		"kbps_sq":          "855",
		"song_size_sq_str": "31.5MB",
		"song_play_url_fq": "https://example.com/c200.m4a",
		"song_filename_fq": "C200003xyz.m4a",
		"kbps_fq":          "48",
		"song_size_fq_str": "1.8MB",
	}
}

func TestOOListQualityOptionsAndFallback(t *testing.T) {
	detail := ooTestDetail()
	options := OOListQualityOptions(detail)
	if len(options) != 2 {
		t.Fatalf("可用音质数 = %d, 期望 2", len(options))
	}
	if options[0].Quality != "SQ" || options[0].Ext != ".flac" || options[0].Kbps != "855" {
		t.Fatalf("SQ 档解析错误: %+v", options[0])
	}
	if options[1].Quality != "low" || options[1].Ext != ".m4a" {
		t.Fatalf("low 档解析错误: %+v", options[1])
	}

	url, quality, ext, kbps, err := OOGetPlayURL(detail, "HQ")
	if err != nil {
		t.Fatal(err)
	}
	if url == "" || quality != "SQ" || ext != ".flac" || kbps != "855" {
		t.Fatalf("HQ 不可用时应回退 SQ: %q %q %q %q", url, quality, ext, kbps)
	}

	_, quality, ext, _, err = OOGetPlayURL(detail, "low")
	if err != nil || quality != "low" || ext != ".m4a" {
		t.Fatalf("低音质应保留 .m4a 扩展名: quality=%q ext=%q err=%v", quality, ext, err)
	}
}

func TestOOParseSongInfo(t *testing.T) {
	info := OOParseSongInfo(ooTestDetail())
	if info.Title != "稻香" || info.Artist != "周杰伦" || info.Album != "魔杰座" {
		t.Fatalf("基本信息错误: %+v", info)
	}
	if info.DurationStr != "03:43" {
		t.Fatalf("时长格式化错误: %q", info.DurationStr)
	}
	if info.CoverURL == "" || info.LyricLRC == "" || info.LyricText == "" {
		t.Fatalf("封面或歌词缺失: %+v", info)
	}
	if info.Quality != "SQ" || info.FileExtension != ".flac" || info.Kbps != "855" || info.SizeStr != "31.5MB" {
		t.Fatalf("最高可用音质摘要错误: %+v", info)
	}
	if len(info.QualityOptions) != 2 {
		t.Fatalf("音质列表长度 = %d", len(info.QualityOptions))
	}
}
