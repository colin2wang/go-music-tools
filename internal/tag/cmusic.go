package tag

// cmusic.go 提供修补流程需要的细粒度标签操作，全部基于统一的 Read/Write 合并语义。

func ReadAll(path string) (TrackTags, error) { return Read(path) }

func ReadArtists(path string) ([]string, error) {
	tags, err := Read(path)
	if err != nil {
		return nil, err
	}
	return tags.Artists, nil
}

func WriteArtists(path string, artists []string) error {
	return Write(path, TrackTags{Artists: artists})
}

func ReadCMUSICID(path string) (string, error) {
	tags, err := Read(path)
	if err != nil {
		return "", err
	}
	return tags.CMusicID, nil
}

// HasCMUSICID 是歌词修复流程的幂等判据：只判断 tag 是否存在，不比对具体值。
func HasCMUSICID(path string) bool {
	tags, err := Read(path)
	if err != nil {
		return true // 读取失败时保守视为已处理，避免重复破坏文件
	}
	return tags.CMusicID != ""
}

func WriteCMUSICID(path, id string) error {
	return Write(path, TrackTags{CMusicID: id})
}

func ReadLyrics(path string) (string, error) {
	tags, err := Read(path)
	if err != nil {
		return "", err
	}
	return tags.Lyrics, nil
}

// WriteLyrics 写入完整歌词；MP3 会按 chunkSize 分块写多个 COMM 帧。
func WriteLyrics(path, lyrics string, chunkSize int) error {
	return Write(path, TrackTags{Lyrics: lyrics, LyricChunkSize: chunkSize})
}

func WriteCover(path string, data []byte, mime string) error {
	return Write(path, TrackTags{Cover: data, CoverMIME: mime})
}

func ClearCover(path string) error {
	return Write(path, TrackTags{ClearCover: true})
}

// MetadataStatus 供「仅修缺失」判断使用，只看标题 / 艺术家 / 专辑三个字段。
func MetadataStatus(path string) (hasTitle, hasArtist, hasAlbum, hasCover bool, err error) {
	tags, err := Read(path)
	if err != nil {
		return false, false, false, false, err
	}
	return tags.Title != "", len(tags.Artists) > 0, tags.Album != "", tags.Cover != nil, nil
}
