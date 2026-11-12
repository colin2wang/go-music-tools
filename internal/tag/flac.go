package tag

import (
	"strconv"

	"github.com/go-flac/flacpicture"
	"github.com/go-flac/flacvorbis"
	flac "github.com/go-flac/go-flac"
)

type flacHandler struct{}

func (flacHandler) Read(path string) (TrackTags, error) {
	f, err := flac.ParseFile(path)
	if err != nil {
		return TrackTags{}, err
	}
	out := TrackTags{}
	for _, block := range f.Meta {
		switch block.Type {
		case flac.VorbisComment:
			vc, err := flacvorbis.ParseFromMetaDataBlock(*block)
			if err != nil {
				continue
			}
			out.Title = First(vc.Get("TITLE"))
			out.Album = First(vc.Get("ALBUM"))
			out.Lyrics = First(vc.Get("LYRICS"))
			out.CMusicID = First(vc.Get("CMUSIC_ID"))
			out.Genre = First(vc.Get("GENRE"))
			out.Date = First(vc.Get("DATE"))
			out.Language = First(vc.Get("LANGUAGE"))
			out.Year = First(vc.Get("DATE"))
			out.TrackNumber = ParseTrackNumber(First(vc.Get("TRACKNUMBER")))
			for _, artist := range All(vc.Get("ARTIST")) {
				out.Artists = append(out.Artists, artist)
			}
			if len(out.Artists) == 0 {
				out.Artists = SplitArtists(First(vc.Get("ARTIST")))
			}
		case flac.Picture:
			pic, err := flacpicture.ParseFromMetaDataBlock(*block)
			if err == nil && len(pic.ImageData) > 0 {
				out.Cover = append([]byte(nil), pic.ImageData...)
				out.CoverMIME = pic.MIME
			}
		}
	}
	return out, nil
}

func (flacHandler) Write(path string, tags TrackTags) error {
	f, err := flac.ParseFile(path)
	if err != nil {
		return err
	}

	vc := flacvorbis.New()
	kept := make([]*flac.MetaDataBlock, 0, len(f.Meta)+2)
	var originalPictures []*flac.MetaDataBlock
	for _, block := range f.Meta {
		switch block.Type {
		case flac.VorbisComment:
			old, err := flacvorbis.ParseFromMetaDataBlock(*block)
			if err == nil && old.Vendor != "" {
				vc.Vendor = old.Vendor
			}
		case flac.Picture:
			originalPictures = append(originalPictures, block)
		case flac.Padding:
			// 丢弃填充块，写入时按新内容重建
		default:
			kept = append(kept, block)
		}
	}

	add := func(key, value string) {
		if value != "" {
			_ = vc.Add(key, value)
		}
	}
	add("TITLE", tags.Title)
	for _, artist := range tags.Artists {
		add("ARTIST", artist)
	}
	add("ALBUM", tags.Album)
	add("LYRICS", tags.Lyrics)
	add("CMUSIC_ID", tags.CMusicID)
	add("GENRE", tags.Genre)
	add("DATE", tags.Date)
	if tags.Date == "" && tags.Year != "" {
		add("DATE", tags.Year)
	}
	add("LANGUAGE", tags.Language)
	if tags.TrackNumber > 0 {
		add("TRACKNUMBER", strconv.Itoa(tags.TrackNumber))
	}

	vcBlock := vc.Marshal()
	kept = append(kept, &vcBlock)

	if !tags.ClearCover && len(tags.Cover) > 0 {
		mime := tags.CoverMIME
		if mime == "" {
			mime = MimeFromData(tags.Cover)
		}
		pic, err := flacpicture.NewFromImageData(flacpicture.PictureTypeFrontCover, "Cover", tags.Cover, mime)
		if err == nil {
			picBlock := pic.Marshal()
			kept = append(kept, &picBlock)
		} else {
			kept = append(kept, originalPictures...)
		}
	}

	f.Meta = kept
	return f.Save(path)
}
