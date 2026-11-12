package tag

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// 纯 Go 的 MP4 原子读写：定位 moov/udta/meta/ilst 并重建标签原子。
// 当 moov 位于 mdat 之前时，moov 体积变化会移动 mdat，需要同步修正 stco/co64 中的绝对偏移。

type mp4Box struct {
	typ     string
	raw     []byte
	payload []byte
}

// MP4 原子名固定 4 字节，© 在文件中是单字节 0xA9；Go 源码里的 "©" 是 UTF-8 双字节，
// 因此写入与读取时都要做一次归一化，否则原子名会占 5 字节而破坏整个文件。
func boxTypeToBytes(typ string) []byte {
	b := []byte(typ)
	if len(b) >= 2 && b[0] == 0xC2 && b[1] == 0xA9 {
		b = append([]byte{0xA9}, b[2:]...)
	}
	if len(b) > 4 {
		b = b[:4]
	}
	for len(b) < 4 {
		b = append(b, 0)
	}
	return b
}

func boxTypeFromBytes(b []byte) string {
	if len(b) > 0 && b[0] == 0xA9 {
		return "©" + string(b[1:])
	}
	return string(b)
}

func parseMP4Boxes(data []byte) []mp4Box {
	boxes := make([]mp4Box, 0, 8)
	for off := 0; off+8 <= len(data); {
		size := int(binary.BigEndian.Uint32(data[off : off+4]))
		typ := boxTypeFromBytes(data[off+4 : off+8])
		header := 8
		if size == 1 {
			if off+16 > len(data) {
				break
			}
			extended := binary.BigEndian.Uint64(data[off+8 : off+16])
			if extended > uint64(len(data)) {
				break
			}
			size = int(extended)
			header = 16
		} else if size == 0 {
			size = len(data) - off
		}
		if size < header || off+size > len(data) {
			break
		}
		boxes = append(boxes, mp4Box{
			typ: typ, raw: data[off : off+size], payload: data[off+header : off+size],
		})
		off += size
	}
	return boxes
}

func makeMP4Box(typ string, payload []byte) []byte {
	out := make([]byte, 8+len(payload))
	binary.BigEndian.PutUint32(out[0:4], uint32(8+len(payload)))
	copy(out[4:8], boxTypeToBytes(typ))
	copy(out[8:], payload)
	return out
}

func findMP4Box(boxes []mp4Box, typ string) int {
	for i, b := range boxes {
		if b.typ == typ {
			return i
		}
	}
	return -1
}

func replaceMP4Box(boxes []mp4Box, typ string, payload []byte) []byte {
	var out bytes.Buffer
	replaced := false
	for _, b := range boxes {
		if b.typ == typ && !replaced {
			out.Write(makeMP4Box(typ, payload))
			replaced = true
		} else {
			out.Write(b.raw)
		}
	}
	if !replaced {
		out.Write(makeMP4Box(typ, payload))
	}
	return out.Bytes()
}

func mp4DataAtom(flags uint32, value []byte) []byte {
	body := make([]byte, 8+len(value))
	binary.BigEndian.PutUint32(body[0:4], flags)
	copy(body[8:], value)
	return makeMP4Box("data", body)
}

func mp4TextAtom(typ, value string) []byte {
	return makeMP4Box(typ, mp4DataAtom(1, []byte(value)))
}

func mp4ArtistsAtom(values []string) []byte {
	var body bytes.Buffer
	for _, v := range values {
		body.Write(mp4DataAtom(1, []byte(v)))
	}
	return makeMP4Box("©ART", body.Bytes())
}

func mp4TrackAtom(n int) []byte {
	value := make([]byte, 8)
	binary.BigEndian.PutUint16(value[2:4], uint16(n))
	return makeMP4Box("trkn", mp4DataAtom(0, value))
}

func mp4CoverAtom(data []byte, mime string) []byte {
	flag := uint32(13) // JPEG
	if strings.Contains(mime, "png") {
		flag = 14 // PNG
	}
	return makeMP4Box("covr", mp4DataAtom(flag, data))
}

func mp4FreeformAtom(name, value string) []byte {
	mean := makeMP4Box("mean", append([]byte{0, 0, 0, 0}, []byte("com.apple.iTunes")...))
	field := makeMP4Box("name", append([]byte{0, 0, 0, 0}, []byte(name)...))
	data := makeMP4Box("data", append([]byte{0, 0, 0, 1, 0, 0, 0, 0}, []byte(value)...))
	var body bytes.Buffer
	body.Write(mean)
	body.Write(field)
	body.Write(data)
	return makeMP4Box("----", body.Bytes())
}

func mp4FreeformName(payload []byte) string {
	for _, b := range parseMP4Boxes(payload) {
		if b.typ == "name" && len(b.payload) >= 4 {
			return string(b.payload[4:])
		}
	}
	return ""
}

type mp4Handler struct{}

func (mp4Handler) Read(path string) (TrackTags, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return TrackTags{}, err
	}
	return readMP4Tags(data)
}

func readMP4Tags(data []byte) (TrackTags, error) {
	out := TrackTags{}
	top := parseMP4Boxes(data)
	moovIdx := findMP4Box(top, "moov")
	if moovIdx < 0 {
		return out, fmt.Errorf("MP4 文件缺少 moov 原子")
	}
	udtaIdx := findMP4Box(parseMP4Boxes(top[moovIdx].payload), "udta")
	if udtaIdx < 0 {
		return out, nil
	}
	udta := parseMP4Boxes(top[moovIdx].payload)[udtaIdx]
	metaIdx := findMP4Box(parseMP4Boxes(udta.payload), "meta")
	if metaIdx < 0 {
		return out, nil
	}
	meta := parseMP4Boxes(udta.payload)[metaIdx]
	if len(meta.payload) < 4 {
		return out, nil
	}
	ilstIdx := findMP4Box(parseMP4Boxes(meta.payload[4:]), "ilst")
	if ilstIdx < 0 {
		return out, nil
	}
	ilst := parseMP4Boxes(meta.payload[4:])[ilstIdx]

	for _, child := range parseMP4Boxes(ilst.payload) {
		inner := parseMP4Boxes(child.payload)
		var values [][]byte
		var flags []uint32
		for _, b := range inner {
			if b.typ == "data" && len(b.payload) >= 8 {
				values = append(values, b.payload[8:])
				flags = append(flags, binary.BigEndian.Uint32(b.payload[0:4]))
			}
		}
		first := func() string {
			if len(values) == 0 {
				return ""
			}
			return string(values[0])
		}
		switch child.typ {
		case "©nam":
			out.Title = first()
		case "©ART":
			for _, v := range values {
				if s := string(v); s != "" {
					out.Artists = append(out.Artists, s)
				}
			}
		case "©alb":
			out.Album = first()
		case "©lyr":
			out.Lyrics = first()
		case "©gen":
			out.Genre = first()
		case "©day":
			out.Date = first()
			if len(first()) >= 4 {
				out.Year = first()[:4]
			}
		case "trkn":
			if len(values) > 0 && len(values[0]) >= 4 {
				out.TrackNumber = int(binary.BigEndian.Uint16(values[0][2:4]))
			}
		case "covr":
			if len(values) > 0 {
				out.Cover = append([]byte(nil), values[0]...)
				if len(flags) > 0 && flags[0] == 14 {
					out.CoverMIME = "image/png"
				} else {
					out.CoverMIME = "image/jpeg"
				}
			}
		case "----":
			name := mp4FreeformName(child.payload)
			switch name {
			case "CMUSIC_ID":
				out.CMusicID = first()
			case "LANGUAGE":
				out.Language = first()
			}
		}
	}
	return out, nil
}

func (mp4Handler) Write(path string, tags TrackTags) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	top := parseMP4Boxes(data)
	moovIdx := findMP4Box(top, "moov")
	if moovIdx < 0 {
		return fmt.Errorf("MP4 文件缺少 moov 原子，无法写入标签")
	}
	mdatIdx := findMP4Box(top, "mdat")

	newPayload := updateMP4Moov(top[moovIdx].payload, tags)
	newMoov := makeMP4Box("moov", newPayload)
	delta := len(newMoov) - len(top[moovIdx].raw)
	if delta != 0 && mdatIdx >= 0 && moovIdx < mdatIdx {
		newPayload = adjustMP4ChunkOffsets(newPayload, int64(delta))
		newMoov = makeMP4Box("moov", newPayload)
	}

	var out bytes.Buffer
	for i, b := range top {
		if i == moovIdx {
			out.Write(newMoov)
		} else {
			out.Write(b.raw)
		}
	}
	tmp := path + ".tagtmp"
	if err := os.WriteFile(tmp, out.Bytes(), 0o644); err != nil {
		return err
	}
	_ = os.Remove(path)
	return os.Rename(tmp, path)
}

func updateMP4Moov(payload []byte, tags TrackTags) []byte {
	children := parseMP4Boxes(payload)
	idx := findMP4Box(children, "udta")
	var udtaPayload []byte
	if idx >= 0 {
		udtaPayload = children[idx].payload
	}
	return replaceMP4Box(children, "udta", updateMP4Udta(udtaPayload, tags))
}

func updateMP4Udta(payload []byte, tags TrackTags) []byte {
	children := parseMP4Boxes(payload)
	idx := findMP4Box(children, "meta")
	prefix := []byte{0, 0, 0, 0}
	var rest []byte
	if idx >= 0 && len(children[idx].payload) >= 4 {
		copy(prefix, children[idx].payload[:4])
		rest = children[idx].payload[4:]
	}
	newMeta := append(append([]byte(nil), prefix...), updateMP4Meta(rest, tags)...)
	return replaceMP4Box(children, "meta", newMeta)
}

func updateMP4Meta(payload []byte, tags TrackTags) []byte {
	children := parseMP4Boxes(payload)
	idx := findMP4Box(children, "ilst")
	var ilst []byte
	if idx >= 0 {
		ilst = children[idx].payload
	}
	return replaceMP4Box(children, "ilst", buildMP4Ilst(ilst, tags))
}

var managedMP4Atoms = map[string]bool{
	"©nam": true, "©ART": true, "©alb": true, "©lyr": true, "©gen": true,
	"©day": true, "trkn": true, "covr": true,
}

func buildMP4Ilst(existing []byte, tags TrackTags) []byte {
	var out bytes.Buffer
	for _, b := range parseMP4Boxes(existing) {
		if managedMP4Atoms[b.typ] {
			continue
		}
		if b.typ == "----" {
			name := mp4FreeformName(b.payload)
			if name == "CMUSIC_ID" || name == "LANGUAGE" {
				continue
			}
		}
		out.Write(b.raw)
	}
	if tags.Title != "" {
		out.Write(mp4TextAtom("©nam", tags.Title))
	}
	if len(tags.Artists) > 0 {
		out.Write(mp4ArtistsAtom(tags.Artists))
	}
	if tags.Album != "" {
		out.Write(mp4TextAtom("©alb", tags.Album))
	}
	if tags.Lyrics != "" {
		out.Write(mp4TextAtom("©lyr", tags.Lyrics))
	}
	if tags.Genre != "" {
		out.Write(mp4TextAtom("©gen", tags.Genre))
	}
	date := tags.Date
	if date == "" {
		date = tags.Year
	}
	if date != "" {
		out.Write(mp4TextAtom("©day", date))
	}
	if tags.TrackNumber > 0 {
		out.Write(mp4TrackAtom(tags.TrackNumber))
	}
	if !tags.ClearCover && len(tags.Cover) > 0 {
		mime := tags.CoverMIME
		if mime == "" {
			mime = MimeFromData(tags.Cover)
		}
		out.Write(mp4CoverAtom(tags.Cover, mime))
	}
	if tags.CMusicID != "" {
		out.Write(mp4FreeformAtom("CMUSIC_ID", tags.CMusicID))
	}
	if tags.Language != "" {
		out.Write(mp4FreeformAtom("LANGUAGE", tags.Language))
	}
	return out.Bytes()
}

// adjustMP4ChunkOffsets 在 moov 体积变化后修正 stco/co64 的绝对 chunk 偏移。
func adjustMP4ChunkOffsets(payload []byte, delta int64) []byte {
	var out bytes.Buffer
	for _, b := range parseMP4Boxes(payload) {
		body := b.payload
		switch b.typ {
		case "trak", "mdia", "minf", "stbl", "edts":
			body = adjustMP4ChunkOffsets(body, delta)
		case "stco":
			body = adjustMP4Offsets32(body, delta)
		case "co64":
			body = adjustMP4Offsets64(body, delta)
		}
		out.Write(makeMP4Box(b.typ, body))
	}
	return out.Bytes()
}

func adjustMP4Offsets32(body []byte, delta int64) []byte {
	if len(body) < 8 || delta == 0 {
		return body
	}
	out := append([]byte(nil), body...)
	count := int(binary.BigEndian.Uint32(out[4:8]))
	for i := 0; i < count; i++ {
		off := 8 + i*4
		if off+4 > len(out) {
			break
		}
		v := int64(binary.BigEndian.Uint32(out[off:off+4])) + delta
		if v < 0 {
			v = 0
		}
		binary.BigEndian.PutUint32(out[off:off+4], uint32(v))
	}
	return out
}

func adjustMP4Offsets64(body []byte, delta int64) []byte {
	if len(body) < 8 || delta == 0 {
		return body
	}
	out := append([]byte(nil), body...)
	count := int(binary.BigEndian.Uint32(out[4:8]))
	for i := 0; i < count; i++ {
		off := 8 + i*8
		if off+8 > len(out) {
			break
		}
		v := int64(binary.BigEndian.Uint64(out[off:off+8])) + delta
		if v < 0 {
			v = 0
		}
		binary.BigEndian.PutUint64(out[off:off+8], uint64(v))
	}
	return out
}

// MP4TagSummary 用于修补流程判断标签是否齐全。
func MP4TagSummary(path string) (hasTitle, hasArtist, hasAlbum, hasCover bool, err error) {
	tags, err := Read(path)
	if err != nil {
		return false, false, false, false, err
	}
	return tags.Title != "", len(tags.Artists) > 0, tags.Album != "", len(tags.Cover) > 0, nil
}

func MP4TrackLabel(n int) string { return strconv.Itoa(n) }
