package tag

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	id3v2 "github.com/bogem/id3v2/v2"
	"github.com/go-flac/flacpicture/v2"
	"github.com/go-flac/flacvorbis/v2"
	flac "github.com/go-flac/go-flac/v2"
)

// ErrUnsupported reports a file whose tags this package cannot read or write.
var ErrUnsupported = errors.New("仅支持 MP3 / FLAC 文件")

type Info struct {
	Title       string
	Artist      string
	Album       string
	Lyrics      string
	Cover       []byte
	CoverMIME   string
	TrackNumber int
	TrackTotal  int
	DiscNumber  int
	DiscTotal   int
	Date        string
	Genre       string
	Language    string
	Composer    string
	Lyricist    string
}

func Supported(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".mp3", ".flac":
		return true
	default:
		return false
	}
}

// Read loads the tags the editor shows. Fields the file does not carry stay
// empty; the first front cover picture wins.
func Read(path string) (Info, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".flac":
		return readFLAC(path)
	case ".mp3":
		return readMP3(path)
	default:
		return Info{}, ErrUnsupported
	}
}

func readMP3(path string) (Info, error) {
	file, err := id3v2.Open(path, id3v2.Options{Parse: true})
	if err != nil {
		return Info{}, err
	}
	defer file.Close()

	text := func(id string) string {
		value := file.GetTextFrame(id).Text
		return strings.ReplaceAll(value, "\x00", " / ")
	}
	info := Info{
		Title:    text("TIT2"),
		Artist:   text("TPE1"),
		Album:    text("TALB"),
		Genre:    text("TCON"),
		Language: text("TLAN"),
		Composer: text("TCOM"),
		Lyricist: text("TEXT"),
	}
	info.Date = text("TDRC")
	if info.Date == "" {
		info.Date = text("TYER")
	}
	info.TrackNumber, info.TrackTotal = parseNumberPair(text("TRCK"))
	info.DiscNumber, info.DiscTotal = parseNumberPair(text("TPOS"))

	for _, frame := range file.GetFrames("USLT") {
		switch lyrics := frame.(type) {
		case id3v2.UnsynchronisedLyricsFrame:
			info.Lyrics = lyrics.Lyrics
		case *id3v2.UnsynchronisedLyricsFrame:
			info.Lyrics = lyrics.Lyrics
		}
		if info.Lyrics != "" {
			break
		}
	}
	var fallback *id3v2.PictureFrame
	for _, frame := range file.GetFrames("APIC") {
		var picture *id3v2.PictureFrame
		switch value := frame.(type) {
		case id3v2.PictureFrame:
			picture = &value
		case *id3v2.PictureFrame:
			picture = value
		}
		if picture == nil {
			continue
		}
		if picture.PictureType != id3v2.PTFrontCover {
			if fallback == nil {
				fallback = picture
			}
			continue
		}
		info.Cover = append([]byte(nil), picture.Picture...)
		info.CoverMIME = picture.MimeType
		break
	}
	if info.Cover == nil && fallback != nil {
		info.Cover = append([]byte(nil), fallback.Picture...)
		info.CoverMIME = fallback.MimeType
	}
	return info, nil
}

func readFLAC(path string) (Info, error) {
	file, err := flac.ParseFile(path)
	if err != nil {
		return Info{}, err
	}
	defer file.Close()

	var info Info
	for _, block := range file.Meta {
		if block == nil {
			continue
		}
		switch block.Type {
		case flac.VorbisComment:
			comments, err := flacvorbis.ParseFromMetaDataBlock(*block)
			if err != nil {
				return Info{}, err
			}
			for _, comment := range comments.Comments {
				if !strings.Contains(comment, "=") {
					return Info{}, errors.New("标签格式损坏")
				}
			}
			value := func(key string) string {
				values, _ := comments.Get(key)
				if len(values) == 0 {
					return ""
				}
				return values[0]
			}
			info.Title = value(flacvorbis.FIELD_TITLE)
			info.Artist = value(flacvorbis.FIELD_ARTIST)
			info.Album = value(flacvorbis.FIELD_ALBUM)
			info.Date = value(flacvorbis.FIELD_DATE)
			if info.Date == "" {
				info.Date = value("YEAR")
			}
			info.Genre = value(flacvorbis.FIELD_GENRE)
			info.Language = value("LANGUAGE")
			info.Composer = value("COMPOSER")
			info.Lyricist = value("LYRICIST")
			info.Lyrics = value("LYRICS")
			if info.Lyrics == "" {
				info.Lyrics = value("UNSYNCEDLYRICS")
			}
			info.TrackNumber, info.TrackTotal = parseNumberPair(value(flacvorbis.FIELD_TRACKNUMBER))
			if info.TrackTotal == 0 {
				info.TrackTotal = intValue(value("TRACKTOTAL"))
				if info.TrackTotal == 0 {
					info.TrackTotal = intValue(value("TOTALTRACKS"))
				}
			}
			info.DiscNumber, info.DiscTotal = parseNumberPair(value("DISCNUMBER"))
			if info.DiscTotal == 0 {
				info.DiscTotal = intValue(value("DISCTOTAL"))
				if info.DiscTotal == 0 {
					info.DiscTotal = intValue(value("TOTALDISCS"))
				}
			}
		case flac.Picture:
			if info.Cover != nil {
				continue
			}
			picture, err := flacpicture.ParseFromMetaDataBlock(*block)
			if err != nil || picture.PictureType != flacpicture.PictureTypeFrontCover {
				continue
			}
			info.Cover = picture.ImageData
			info.CoverMIME = picture.MIME
		}
	}
	return info, nil
}

func number(value int) string {
	if value <= 0 {
		return ""
	}
	return strconv.Itoa(value)
}

func numberPair(value int, total int) string {
	if value <= 0 {
		return ""
	}
	if total > 0 {
		return strconv.Itoa(value) + "/" + strconv.Itoa(total)
	}
	return strconv.Itoa(value)
}

func intValue(value string) int {
	value = strings.TrimSpace(value)
	if index := strings.IndexByte(value, '/'); index >= 0 {
		value = strings.TrimSpace(value[:index])
	}
	number, _ := strconv.Atoi(value)
	return number
}

// parseNumberPair reads the leading number and the total of "3/12" or "3".
func parseNumberPair(value string) (int, int) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, 0
	}
	parts := strings.SplitN(value, "/", 2)
	first, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
	if len(parts) == 1 {
		return first, 0
	}
	second, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
	return first, second
}

func Write(path string, info Info) error {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".flac":
		return writeFLAC(path, info)
	case ".mp3":
		return writeMP3(path, info)
	default:
		return ErrUnsupported
	}
}

// writeAtomic replaces path with the file fn writes to the temp path it
// receives, so a failure partway leaves the original audio untouched.
func writeAtomic(path string, fn func(tmp string) error) (err error) {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tag-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	success := false
	defer func() {
		if !success {
			_ = os.Remove(tmpName)
		}
	}()
	if err := fn(tmpName); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	success = true
	return nil
}

// ownedComments are the Vorbis fields the editor rewrites. Everything else
// in the file is carried over untouched.
var ownedComments = map[string]bool{
	"TITLE":          true,
	"ARTIST":         true,
	"ALBUM":          true,
	"TRACKNUMBER":    true,
	"TRACKTOTAL":     true,
	"TOTALTRACKS":    true,
	"DISCNUMBER":     true,
	"DISCTOTAL":      true,
	"TOTALDISCS":     true,
	"DATE":           true,
	"YEAR":           true,
	"GENRE":          true,
	"LANGUAGE":       true,
	"COMPOSER":       true,
	"LYRICIST":       true,
	"LYRICS":         true,
	"UNSYNCEDLYRICS": true,
}

func commentKey(comment string) string {
	index := strings.IndexByte(comment, '=')
	if index < 0 {
		return ""
	}
	return strings.ToUpper(strings.TrimSpace(comment[:index]))
}

// writeFLAC keeps the existing comments and pictures, replacing only the
// fields the editor owns and the front cover.
func writeFLAC(path string, info Info) error {
	return writeAtomic(path, func(tmp string) error {
		file, err := flac.ParseFile(path)
		if err != nil {
			return err
		}
		defer file.Close()

		meta := make([]*flac.MetaDataBlock, 0, len(file.Meta)+2)
		var comments *flacvorbis.MetaDataBlockVorbisComment
		for _, block := range file.Meta {
			if block == nil {
				continue
			}
			switch block.Type {
			case flac.VorbisComment:
				if comments == nil {
					parsed, err := flacvorbis.ParseFromMetaDataBlock(*block)
					if err != nil {
						return err
					}
					comments = parsed
				}
			case flac.Picture:
				picture, err := flacpicture.ParseFromMetaDataBlock(*block)
				if err == nil && picture.PictureType == flacpicture.PictureTypeFrontCover {
					continue
				}
				meta = append(meta, block)
			default:
				meta = append(meta, block)
			}
		}
		if comments == nil {
			comments = flacvorbis.New()
		}
		kept := comments.Comments[:0]
		for _, comment := range comments.Comments {
			if ownedComments[commentKey(comment)] {
				continue
			}
			kept = append(kept, comment)
		}
		comments.Comments = kept

		for _, field := range []struct {
			key   string
			value string
		}{
			{flacvorbis.FIELD_TITLE, info.Title},
			{flacvorbis.FIELD_ARTIST, info.Artist},
			{flacvorbis.FIELD_ALBUM, info.Album},
			{flacvorbis.FIELD_TRACKNUMBER, number(info.TrackNumber)},
			{"TRACKTOTAL", number(info.TrackTotal)},
			{"DISCNUMBER", number(info.DiscNumber)},
			{"DISCTOTAL", number(info.DiscTotal)},
			{flacvorbis.FIELD_DATE, info.Date},
			{flacvorbis.FIELD_GENRE, info.Genre},
			{"LANGUAGE", info.Language},
			{"COMPOSER", info.Composer},
			{"LYRICIST", info.Lyricist},
			{"LYRICS", info.Lyrics},
		} {
			if strings.TrimSpace(field.value) == "" {
				continue
			}
			if err := comments.Add(field.key, field.value); err != nil {
				return err
			}
		}
		commentBlock := comments.Marshal()
		meta = append(meta, &commentBlock)

		if len(info.Cover) > 0 {
			mime := info.CoverMIME
			if mime == "" {
				mime = ImageMIME(info.Cover)
			}
			if mime != "" {
				picture, err := flacpicture.NewFromImageData(flacpicture.PictureTypeFrontCover, "Cover", info.Cover, mime)
				if err == nil {
					pictureBlock := picture.Marshal()
					meta = append(meta, &pictureBlock)
				}
			}
		}

		file.Meta = meta

		// flac.File.Save leaks the handle it opens to inspect an existing
		// target, and Windows cannot rename a file that is still open, so
		// write the tagged stream ourselves.
		out, err := os.Create(tmp)
		if err != nil {
			return err
		}
		if _, err := file.WriteTo(out); err != nil {
			_ = out.Close()
			return err
		}
		return out.Close()
	})
}

func writeMP3(path string, info Info) error {
	return writeAtomic(path, func(tmp string) (err error) {
		// id3v2 saves in place, so tag a copy of the audio and swap it in.
		if err := copyFile(path, tmp); err != nil {
			return err
		}
		file, err := id3v2.Open(tmp, id3v2.Options{Parse: true})
		if err != nil {
			return err
		}
		defer func() {
			if closeErr := file.Close(); err == nil && closeErr != nil {
				err = closeErr
			}
		}()

		file.SetVersion(4)

		for _, id := range []string{
			"TIT2", "TPE1", "TALB", "APIC", "USLT",
			"TRCK", "TPOS", "TDRC", "TYER", "TCON", "TLAN", "TCOM", "TEXT",
		} {
			file.DeleteFrames(id)
		}

		if strings.TrimSpace(info.Title) != "" {
			file.SetTitle(info.Title)
		}
		if strings.TrimSpace(info.Artist) != "" {
			file.SetArtist(info.Artist)
		}
		if strings.TrimSpace(info.Album) != "" {
			file.SetAlbum(info.Album)
		}
		if len(info.Cover) > 0 {
			mime := info.CoverMIME
			if mime == "" {
				mime = ImageMIME(info.Cover)
			}
			if mime != "" {
				file.AddAttachedPicture(id3v2.PictureFrame{
					Encoding:    id3v2.EncodingUTF8,
					MimeType:    mime,
					PictureType: id3v2.PTFrontCover,
					Description: "Cover",
					Picture:     info.Cover,
				})
			}
		}
		if strings.TrimSpace(info.Lyrics) != "" {
			file.AddUnsynchronisedLyricsFrame(id3v2.UnsynchronisedLyricsFrame{
				Encoding:          id3v2.EncodingUTF8,
				Language:          "eng",
				ContentDescriptor: "Lyrics",
				Lyrics:            info.Lyrics,
			})
		}
		for id, value := range map[string]string{
			"TRCK": numberPair(info.TrackNumber, info.TrackTotal),
			"TPOS": numberPair(info.DiscNumber, info.DiscTotal),
			"TDRC": info.Date,
			"TCON": info.Genre,
			"TLAN": info.Language,
			"TCOM": info.Composer,
			"TEXT": info.Lyricist,
		} {
			if strings.TrimSpace(value) != "" {
				file.AddTextFrame(id, id3v2.EncodingUTF8, value)
			}
		}
		return file.Save()
	})
}

func copyFile(src, dst string) error {
	from, err := os.Open(src)
	if err != nil {
		return err
	}
	defer from.Close()
	to, err := os.OpenFile(dst, os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(to, from); err != nil {
		_ = to.Close()
		return err
	}
	return to.Close()
}

// ImageMIME names the image format embedded in data, or returns an empty
// string when the format is not one the taggers can declare.
func ImageMIME(data []byte) string {
	switch {
	case len(data) >= 8 && string(data[:8]) == "\x89PNG\r\n\x1a\n":
		return "image/png"
	case len(data) >= 2 && data[0] == 0xff && data[1] == 0xd8:
		return "image/jpeg"
	default:
		return ""
	}
}
