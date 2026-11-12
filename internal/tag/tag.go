package tag

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

type Format string

const (
	FormatMP3     Format = "mp3"
	FormatFLAC    Format = "flac"
	FormatMP4     Format = "mp4"
	FormatUnknown Format = ""
)

var (
	mp3Ext  = map[string]bool{".mp3": true, ".mp2": true, ".mp1": true}
	flacExt = map[string]bool{".flac": true}
	mp4Ext  = map[string]bool{".m4a": true, ".mp4": true, ".aac": true}
)

func Detect(path string) Format {
	ext := strings.ToLower(filepath.Ext(path))
	switch {
	case mp3Ext[ext]:
		return FormatMP3
	case flacExt[ext]:
		return FormatFLAC
	case mp4Ext[ext]:
		return FormatMP4
	default:
		return FormatUnknown
	}
}

func IsAudio(path string) bool { return Detect(path) != FormatUnknown }

// TrackTags 是 MP3 / FLAC / MP4 的统一标签视图。空字段在 Write 时表示「保留原值」。
type TrackTags struct {
	Title       string
	Album       string
	Lyrics      string
	CMusicID    string
	Artists     []string
	TrackNumber int
	Year        string
	Genre       string
	Date        string
	Language    string
	Cover       []byte
	CoverMIME   string
	ClearCover  bool
	// LyricChunkSize 用于 MP3 COMM 分块；0 表示不限制。
	LyricChunkSize int
}

type Handler interface {
	Read(path string) (TrackTags, error)
	Write(path string, tags TrackTags) error
}

func HandlerFor(f Format) (Handler, error) {
	switch f {
	case FormatMP3:
		return mp3Handler{}, nil
	case FormatFLAC:
		return flacHandler{}, nil
	case FormatMP4:
		return mp4Handler{}, nil
	default:
		return nil, fmt.Errorf("不支持的音频格式: %q", f)
	}
}

// Merge 让修补流程可以只传需要改动的字段，其余保留文件中已有值。
func Merge(base, patch TrackTags) TrackTags {
	out := base
	if patch.Title != "" {
		out.Title = patch.Title
	}
	if patch.Album != "" {
		out.Album = patch.Album
	}
	if patch.Lyrics != "" {
		out.Lyrics = patch.Lyrics
	}
	if patch.CMusicID != "" {
		out.CMusicID = patch.CMusicID
	}
	if len(patch.Artists) > 0 {
		out.Artists = patch.Artists
	}
	if patch.TrackNumber > 0 {
		out.TrackNumber = patch.TrackNumber
	}
	if patch.Year != "" {
		out.Year = patch.Year
	}
	if patch.Genre != "" {
		out.Genre = patch.Genre
	}
	if patch.Date != "" {
		out.Date = patch.Date
	}
	if patch.Language != "" {
		out.Language = patch.Language
	}
	if len(patch.Cover) > 0 {
		out.Cover = patch.Cover
		out.CoverMIME = patch.CoverMIME
	}
	if patch.ClearCover {
		out.ClearCover = true
		out.Cover = nil
	}
	if patch.LyricChunkSize > 0 {
		out.LyricChunkSize = patch.LyricChunkSize
	}
	return out
}

// Read 读取任意受支持格式的标签；读取失败时返回空标签而不是中断调用方。
func Read(path string) (TrackTags, error) {
	h, err := HandlerFor(Detect(path))
	if err != nil {
		return TrackTags{}, err
	}
	return h.Read(path)
}

// Write 先读后合并再写，保证修补流程不会误清空其它标签。
func Write(path string, patch TrackTags) error {
	format := Detect(path)
	h, err := HandlerFor(format)
	if err != nil {
		return err
	}
	current, err := h.Read(path)
	if err != nil {
		current = TrackTags{}
	}
	merged := Merge(current, patch)
	if merged.Year == "" {
		merged.Year = current.Year
	}
	return h.Write(path, merged)
}

func ParseTrackNumber(raw string) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	if i := strings.IndexAny(raw, "/"); i > 0 {
		raw = raw[:i]
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return 0
	}
	return n
}

// SplitArtists 把 /、&、, 、、 分隔的艺术家字符串拆为多值。
func SplitArtists(raw string) []string {
	replacer := strings.NewReplacer("/", ",", "&", ",", "、", ",")
	parts := strings.Split(replacer.Replace(raw), ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func First(values []string, err error) string {
	if err != nil || len(values) == 0 {
		return ""
	}
	return values[0]
}

func All(values []string, err error) []string {
	if err != nil {
		return nil
	}
	return values
}
