package tag

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func samplePNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 108, G: 92, B: 231, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestDetectAndMerge(t *testing.T) {
	cases := map[string]Format{
		"song.mp3": FormatMP3, "song.FLAC": FormatFLAC,
		"song.m4a": FormatMP4, "song.mp4": FormatMP4, "song.txt": FormatUnknown,
	}
	for name, want := range cases {
		if got := Detect(name); got != want {
			t.Errorf("Detect(%s) = %s, want %s", name, got, want)
		}
	}

	base := TrackTags{Title: "原曲", Artists: []string{"A"}, Album: "专辑", Cover: []byte{1, 2}}
	patch := TrackTags{Artists: []string{"A", "B"}, CMusicID: "123"}
	got := Merge(base, patch)
	if got.Title != "原曲" || got.Album != "专辑" || len(got.Artists) != 2 || got.CMusicID != "123" {
		t.Fatalf("Merge 结果不正确: %+v", got)
	}
	if len(got.Cover) != 2 {
		t.Fatal("Merge 丢失了原有封面")
	}
}

func writeDummy(t *testing.T, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestMP3RoundTrip(t *testing.T) {
	path := writeDummy(t, "song.mp3", bytes.Repeat([]byte{0xFF, 0xFB, 0x00, 0x00}, 8))
	cover := samplePNG(t)
	err := Write(path, TrackTags{
		Title: "测试歌曲", Artists: []string{"歌手甲", "歌手乙"}, Album: "测试专辑",
		Year: "2026", TrackNumber: 3, Lyrics: "[00:00.00]第一行\n[00:10.00]第二行",
		CMusicID: "3383993462", Cover: cover, CoverMIME: "image/png",
	})
	if err != nil {
		t.Fatalf("写入 MP3 标签失败: %v", err)
	}
	got, err := Read(path)
	if err != nil {
		t.Fatalf("读取 MP3 标签失败: %v", err)
	}
	if got.Title != "测试歌曲" || got.Album != "测试专辑" || got.TrackNumber != 3 {
		t.Fatalf("基础标签不符: %+v", got)
	}
	if got.CMusicID != "3383993462" {
		t.Fatalf("CMUSIC_ID 未写入: %q", got.CMusicID)
	}
	if !bytes.Contains([]byte(got.Lyrics), []byte("第二行")) {
		t.Fatalf("歌词未写入: %q", got.Lyrics)
	}
	if len(got.Cover) == 0 {
		t.Fatal("封面未写入")
	}
}

func minimalFLAC() []byte {
	streamInfo := make([]byte, 34)
	var out bytes.Buffer
	out.WriteString("fLaC")
	out.WriteByte(0x80) // 最后一个块 + StreamInfo
	out.Write([]byte{0, 0, 34})
	out.Write(streamInfo)
	out.Write([]byte{0xFF, 0xF8, 0x00, 0x00})
	return out.Bytes()
}

func TestFLACRoundTrip(t *testing.T) {
	path := writeDummy(t, "song.flac", minimalFLAC())
	err := Write(path, TrackTags{
		Title: "无损曲目", Artists: []string{"歌手甲"}, Album: "无损专辑",
		TrackNumber: 2, Lyrics: "[00:00.00]歌词", CMusicID: "9988",
		Cover: samplePNG(t), CoverMIME: "image/png",
	})
	if err != nil {
		t.Fatalf("写入 FLAC 标签失败: %v", err)
	}
	got, err := Read(path)
	if err != nil {
		t.Fatalf("读取 FLAC 标签失败: %v", err)
	}
	if got.Title != "无损曲目" || got.Album != "无损专辑" || got.TrackNumber != 2 || got.CMusicID != "9988" {
		t.Fatalf("FLAC 标签不符: %+v", got)
	}
	if len(got.Cover) == 0 {
		t.Fatal("FLAC 封面未写入")
	}
}

func minimalMP4() []byte {
	stco := make([]byte, 12)
	binary.BigEndian.PutUint32(stco[4:8], 1)
	binary.BigEndian.PutUint32(stco[8:12], 12345)
	moovPayload := makeMP4Box("trak", makeMP4Box("mdia", makeMP4Box("minf",
		makeMP4Box("stbl", makeMP4Box("stco", stco)))))
	var out bytes.Buffer
	out.Write(makeMP4Box("ftyp", []byte("M4A \x00\x00\x00\x00")))
	out.Write(makeMP4Box("moov", moovPayload))
	out.Write(makeMP4Box("mdat", []byte{0, 1, 2, 3, 4, 5, 6, 7}))
	return out.Bytes()
}

func TestMP4RoundTripAndChunkOffset(t *testing.T) {
	original := minimalMP4()
	path := writeDummy(t, "song.m4a", original)
	err := Write(path, TrackTags{
		Title: "M4A 曲目", Artists: []string{"歌手甲", "歌手乙"}, Album: "M4A 专辑",
		Lyrics: "[00:00.00]M4A 歌词", TrackNumber: 5, CMusicID: "7766",
		Cover: samplePNG(t), CoverMIME: "image/png", Language: "chi",
	})
	if err != nil {
		t.Fatalf("写入 MP4 标签失败: %v", err)
	}
	got, err := Read(path)
	if err != nil {
		t.Fatalf("读取 MP4 标签失败: %v", err)
	}
	if got.Title != "M4A 曲目" || got.Album != "M4A 专辑" || got.TrackNumber != 5 {
		t.Fatalf("MP4 基础标签不符: %+v", got)
	}
	if got.CMusicID != "7766" || got.Language != "chi" {
		t.Fatalf("MP4 freeform 标签不符: %+v", got)
	}
	if len(got.Artists) != 2 || len(got.Cover) == 0 {
		t.Fatalf("MP4 多值艺术家或封面不符: %+v", got)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	oldMoov := parseMP4Boxes(original)[1]
	newMoov := parseMP4Boxes(data)[1]
	delta := len(newMoov.raw) - len(oldMoov.raw)
	if delta == 0 {
		t.Fatal("moov 体积没有变化，无法验证 chunk offset 调整")
	}
	stbl := parseMP4Boxes(parseMP4Boxes(parseMP4Boxes(parseMP4Boxes(newMoov.payload)[0].payload)[0].payload)[0].payload)[0]
	adjusted := parseMP4Boxes(stbl.payload)[0].payload
	if len(adjusted) < 12 {
		t.Fatal("stco 原子丢失")
	}
	want := uint32(12345 + delta)
	if gotOffset := binary.BigEndian.Uint32(adjusted[8:12]); gotOffset != want {
		t.Fatalf("stco 偏移未修正: got %d, want %d", gotOffset, want)
	}
}

func TestMimeAndImageSize(t *testing.T) {
	data := samplePNG(t)
	if MimeFromData(data) != "image/png" {
		t.Fatal("PNG 识别失败")
	}
	w, h := ImageSize(data)
	if w != 2 || h != 2 {
		t.Fatalf("图片尺寸识别失败: %dx%d", w, h)
	}
	if MimeFromURL("https://example.com/pic/abc.jpg?param=320x320") != "image/jpeg" {
		t.Fatal("URL 后缀识别失败")
	}
}
