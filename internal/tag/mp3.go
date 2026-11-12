package tag

import (
	"fmt"
	"strconv"
	"strings"

	id3v2 "github.com/bogem/id3v2/v2"
)

type mp3Handler struct{}

func (mp3Handler) Read(path string) (TrackTags, error) {
	t, err := id3v2.Open(path, id3v2.Options{Parse: true})
	if err != nil {
		return TrackTags{}, err
	}
	defer t.Close()

	out := TrackTags{
		Title: t.Title(),
		Album: t.Album(),
		Year:  t.Year(),
		Genre: t.Genre(),
	}
	out.Artists = SplitArtists(t.Artist())
	out.TrackNumber = ParseTrackNumber(t.GetTextFrame("TRCK").Text)

	var lyrics strings.Builder
	for _, f := range t.GetFrames("COMM") {
		if cf, ok := f.(id3v2.CommentFrame); ok && cf.Text != "" {
			lyrics.WriteString(cf.Text)
		}
	}
	for _, f := range t.GetFrames("USLT") {
		if lf, ok := f.(id3v2.UnsynchronisedLyricsFrame); ok && lf.Lyrics != "" {
			lyrics.WriteString(lf.Lyrics)
		}
	}
	out.Lyrics = lyrics.String()

	for _, f := range t.GetFrames("TXXX") {
		if ud, ok := f.(id3v2.UserDefinedTextFrame); ok && ud.Description == "CMUSIC_ID" {
			out.CMusicID = ud.Value
		}
	}
	for _, f := range t.GetFrames("APIC") {
		if pf, ok := f.(id3v2.PictureFrame); ok {
			out.Cover = append([]byte(nil), pf.Picture...)
			out.CoverMIME = pf.MimeType
			break
		}
	}
	return out, nil
}

func (mp3Handler) Write(path string, tags TrackTags) error {
	t, err := id3v2.Open(path, id3v2.Options{Parse: true})
	if err != nil {
		return err
	}
	defer t.Close()

	t.SetVersion(3)
	t.SetDefaultEncoding(id3v2.EncodingUTF8)
	t.SetTitle(tags.Title)
	t.SetArtist(strings.Join(tags.Artists, ", "))
	t.SetAlbum(tags.Album)
	if tags.Year != "" {
		t.SetYear(tags.Year)
	}
	if tags.Genre != "" {
		t.SetGenre(tags.Genre)
	}

	t.DeleteFrames("TRCK")
	if tags.TrackNumber > 0 {
		t.AddTextFrame("TRCK", id3v2.EncodingUTF8, strconv.Itoa(tags.TrackNumber))
	}

	t.DeleteFrames("COMM")
	t.DeleteFrames("USLT")
	if tags.Lyrics != "" {
		chunk := tags.LyricChunkSize
		if chunk <= 0 {
			chunk = 5000
		}
		runes := []rune(tags.Lyrics)
		for i, part := 0, 0; i < len(runes); i, part = i+chunk, part+1 {
			end := i + chunk
			if end > len(runes) {
				end = len(runes)
			}
			desc := ""
			if part > 0 {
				desc = fmt.Sprintf("Lyric Part %d", part+1)
			}
			t.AddCommentFrame(id3v2.CommentFrame{
				Encoding:    id3v2.EncodingUTF8,
				Language:    "eng",
				Description: desc,
				Text:        string(runes[i:end]),
			})
		}
	}

	if tags.CMusicID != "" {
		t.AddUserDefinedTextFrame(id3v2.UserDefinedTextFrame{
			Encoding: id3v2.EncodingUTF8, Description: "CMUSIC_ID", Value: tags.CMusicID,
		})
	}

	if tags.ClearCover || len(tags.Cover) > 0 {
		t.DeleteFrames("APIC")
	}
	if len(tags.Cover) > 0 {
		mime := tags.CoverMIME
		if mime == "" {
			mime = MimeFromData(tags.Cover)
		}
		t.AddAttachedPicture(id3v2.PictureFrame{
			Encoding:    id3v2.EncodingUTF8,
			MimeType:    mime,
			PictureType: id3v2.PTFrontCover,
			Description: "Cover",
			Picture:     tags.Cover,
		})
	}
	return t.Save()
}
