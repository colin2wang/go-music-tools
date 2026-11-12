package music

import (
	"strings"
	"testing"

	"go-music-tools/internal/config"
	"go-music-tools/internal/model"
)

func TestCleanFilename(t *testing.T) {
	got := CleanFilename(`A/B:C*D?E"F<G>H|I`, "_")
	if strings.ContainsAny(got, `<>:"/\|?*`) {
		t.Fatalf("非法字符未清除: %s", got)
	}
	if got != "A_B_C_D_E_F_G_H_I" {
		t.Fatalf("替换结果不符: %s", got)
	}
}

func TestExtractExtension(t *testing.T) {
	cases := map[string]string{
		"https://x.com/a/song.flac?token=1": ".flac",
		"https://x.com/a/song.m4a":          ".m4a",
		"https://x.com/get?v=track.mp3":     ".mp3",
		"https://x.com/unknown":             ".mp3",
	}
	for raw, want := range cases {
		if got := ExtractExtension(raw); got != want {
			t.Errorf("ExtractExtension(%s) = %s, want %s", raw, got, want)
		}
	}
}

func TestBuildFilename(t *testing.T) {
	cfg := config.Default()
	meta := model.SongMeta{Name: "过火", Artists: "张信哲/陈奕迅", Album: "专辑"}
	got := BuildFilename(cfg, meta, 3)
	if got != "003 - 张信哲, 陈奕迅 - 过火" {
		t.Fatalf("带序号命名不符: %q", got)
	}
	got = BuildFilename(cfg, meta, 0)
	if got != "张信哲, 陈奕迅 - 过火" {
		t.Fatalf("无序号命名不符: %q", got)
	}
	cfg.Download.AddIndexToFilename = false
	if got = BuildFilename(cfg, meta, 3); strings.HasPrefix(got, "003") {
		t.Fatalf("关闭序号后仍带序号: %q", got)
	}
}

func TestBuildFilenameTruncatesLongName(t *testing.T) {
	cfg := config.Default()
	cfg.Filename.MaxLength = 40
	meta := model.SongMeta{Name: strings.Repeat("歌", 200), Artists: strings.Repeat("艺", 200)}
	got := BuildFilename(cfg, meta, 1)
	if len([]rune(got)) > 40 {
		t.Fatalf("文件名未按最大长度收缩: len=%d %q", len([]rune(got)), got)
	}
}

func TestParseMusicFilename(t *testing.T) {
	track, artist, title := ParseMusicFilename("003 - 张信哲 - 过火")
	if track != 3 || artist != "张信哲" || title != "过火" {
		t.Fatalf("带序号解析不符: %d %s %s", track, artist, title)
	}
	track, artist, title = ParseMusicFilename("张信哲 - 过火")
	if track != 0 || artist != "张信哲" || title != "过火" {
		t.Fatalf("无序号解析不符: %d %s %s", track, artist, title)
	}
}
