package oomusic

import (
	"go-music-tools/internal/model"
	"go-music-tools/internal/tag"
)

// WriteTags 写入 OO Music 专用标签集合：除基础字段外带 GENRE / DATE / LANGUAGE，
// 不写入 CMUSIC_ID 与曲目号（与源项目行为一致）。
func WriteTags(path string, info model.OODetail, cover []byte, coverMIME string) error {
	return tag.Write(path, tag.TrackTags{
		Title:     info.Title,
		Artists:   tag.SplitArtists(info.Artist),
		Album:     info.Album,
		Lyrics:    info.LyricLRC,
		Genre:     info.Genre,
		Date:      info.PublicTime,
		Language:  info.Language,
		Cover:     cover,
		CoverMIME: coverMIME,
	})
}
