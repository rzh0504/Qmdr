package tag

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	id3v2 "github.com/bogem/id3v2/v2"
	"github.com/go-flac/flacpicture/v2"
	"github.com/go-flac/flacvorbis/v2"
	flac "github.com/go-flac/go-flac/v2"
)

type Info struct {
	Title     string
	Artist    string
	Album     string
	Lyrics    string
	Cover     []byte
	CoverMIME string
}

func Write(path string, info Info) error {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".flac":
		return writeFLAC(path, info)
	case ".mp3":
		return writeMP3(path, info)
	default:
		return nil
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

func writeFLAC(path string, info Info) error {
	return writeAtomic(path, func(tmp string) error {
		file, err := flac.ParseFile(path)
		if err != nil {
			return err
		}

		meta := make([]*flac.MetaDataBlock, 0, len(file.Meta)+2)
		for _, block := range file.Meta {
			if block == nil {
				continue
			}
			if block.Type == flac.VorbisComment || block.Type == flac.Picture {
				continue
			}
			meta = append(meta, block)
		}

		comments := flacvorbis.New()
		for _, field := range []struct {
			key   string
			value string
		}{
			{flacvorbis.FIELD_TITLE, info.Title},
			{flacvorbis.FIELD_ARTIST, info.Artist},
			{flacvorbis.FIELD_ALBUM, info.Album},
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
				mime = imageMIME(info.Cover)
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
		return file.Save(tmp)
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

		for _, id := range []string{"TIT2", "TPE1", "TALB", "APIC", "USLT"} {
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
				mime = imageMIME(info.Cover)
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

func imageMIME(data []byte) string {
	switch {
	case len(data) >= 8 && string(data[:8]) == "\x89PNG\r\n\x1a\n":
		return "image/png"
	case len(data) >= 2 && data[0] == 0xff && data[1] == 0xd8:
		return "image/jpeg"
	default:
		return ""
	}
}
