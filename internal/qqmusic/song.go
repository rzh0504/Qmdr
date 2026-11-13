package qqmusic

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

type FileType struct {
	Code string
	Ext  string
	Name string
}

var (
	fileTypeMaster  = FileType{Code: "AI00", Ext: ".flac", Name: "臻品母带"}
	fileTypeAtmos2  = FileType{Code: "Q000", Ext: ".flac", Name: "臻品全景声"}
	fileTypeAtmos51 = FileType{Code: "Q001", Ext: ".flac", Name: "臻品音质"}
	fileTypeFLAC    = FileType{Code: "F000", Ext: ".flac", Name: "FLAC"}
	fileTypeMP3320  = FileType{Code: "M800", Ext: ".mp3", Name: "MP3 320K"}
	fileTypeMP3128  = FileType{Code: "M500", Ext: ".mp3", Name: "MP3 128K"}
)

type ResolvedURL struct {
	URL     string
	Ext     string
	Quality string
}

// SongFile is the audio file availability reported with a song: the media
// mid used in file names and the size of each quality, zero when absent.
type SongFile struct {
	MediaMid   string
	SizeMaster int64
	SizeAtmos2 int64
	SizeAtmos51 int64
	SizeFLAC   int64
	SizeMP3320 int64
	SizeMP3128 int64
}

func (f SongFile) SizeFor(ft FileType) int64 {
	switch ft.Code {
	case fileTypeMaster.Code:
		return f.SizeMaster
	case fileTypeAtmos2.Code:
		return f.SizeAtmos2
	case fileTypeAtmos51.Code:
		return f.SizeAtmos51
	case fileTypeFLAC.Code:
		return f.SizeFLAC
	case fileTypeMP3320.Code:
		return f.SizeMP3320
	case fileTypeMP3128.Code:
		return f.SizeMP3128
	}
	return 0
}

// HasSizes reports whether the server listed any file size, so a zero means
// "unavailable" rather than "unknown".
func (f SongFile) HasSizes() bool {
	return f.SizeMaster > 0 || f.SizeAtmos2 > 0 || f.SizeAtmos51 > 0 ||
		f.SizeFLAC > 0 || f.SizeMP3320 > 0 || f.SizeMP3128 > 0
}

// ExpectedSize is the size of the highest quality the preference can use.
func ExpectedSize(quality string, file SongFile) int64 {
	if !file.HasSizes() {
		return 0
	}
	for _, ft := range fileTypesForQuality(quality) {
		if size := file.SizeFor(ft); size > 0 {
			return size
		}
	}
	return 0
}

func fileTypesForQuality(quality string) []FileType {
	switch quality {
	case "臻品母带":
		return []FileType{fileTypeMaster, fileTypeAtmos2, fileTypeAtmos51, fileTypeFLAC, fileTypeMP3320, fileTypeMP3128}
	case "FLAC 无损":
		return []FileType{fileTypeFLAC, fileTypeMP3320, fileTypeMP3128}
	case "MP3 320K":
		return []FileType{fileTypeMP3320, fileTypeMP3128}
	case "MP3 128K":
		return []FileType{fileTypeMP3128}
	default:
		return []FileType{fileTypeFLAC, fileTypeMP3320, fileTypeMP3128}
	}
}

type vkeyEntry struct {
	Songmid  string `json:"songmid"`
	Filename string `json:"filename"`
	Wifiurl  string `json:"wifiurl"`
	Purl     string `json:"purl"`
}

// resolveEntry returns the first entry the server serves for the requested
// quality; entries for other songs and entries without a URL are skipped.
func resolveEntry(mid string, ft FileType, entries []vkeyEntry) (ResolvedURL, bool) {
	for _, entry := range entries {
		if entry.Songmid != "" && entry.Songmid != mid {
			continue
		}
		raw := entry.Wifiurl
		if raw == "" {
			raw = entry.Purl
		}
		if raw == "" {
			continue
		}
		if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
			raw = cdnBase + strings.TrimPrefix(raw, "/")
		}
		return ResolvedURL{URL: raw, Ext: ft.Ext, Quality: ft.Name}, true
	}
	return ResolvedURL{}, false
}

// vkeyNames lists the file names to request for one quality, preferring the
// media mid the service uses and keeping the song mid as a safety net.
func vkeyNames(mid string, ft FileType, mediaMid string) []string {
	names := []string{ft.Code + mid + mid + ft.Ext}
	if mediaMid != "" && mediaMid != mid {
		names = append([]string{ft.Code + mediaMid + ft.Ext}, names...)
	}
	return names
}

func (c *Client) ResolveURL(ctx context.Context, mid, quality string, file SongFile) (ResolvedURL, error) {
	if mid == "" {
		return ResolvedURL{}, errors.New("歌曲缺少 MID")
	}
	// The vkey service only resolves the first filename of a request, so try
	// the candidates one by one; a batch request would never fall back. When
	// the song lists file sizes, qualities of size zero are unavailable and
	// skipped without a request.
	guid := randomHex(16)
	known := file.HasSizes()
	for _, ft := range fileTypesForQuality(quality) {
		if known && file.SizeFor(ft) <= 0 {
			continue
		}
		for _, name := range vkeyNames(mid, ft, file.MediaMid) {
			params := map[string]any{
				"filename": []string{name},
				"guid":     guid,
				"songmid":  []string{mid},
				"songtype": []int{0},
			}
			var data struct {
				Midurlinfo []vkeyEntry `json:"midurlinfo"`
			}
			if err := c.call(ctx, "music.vkey.GetVkey", "UrlGetVkey", params, &data); err != nil {
				return ResolvedURL{}, err
			}
			if resolved, ok := resolveEntry(mid, ft, data.Midurlinfo); ok {
				return resolved, nil
			}
		}
	}
	if !c.HasCredential() {
		return ResolvedURL{}, errors.New("需要登录后才能获取该歌曲的下载链接")
	}
	return ResolvedURL{}, fmt.Errorf("没有可用的音质，请尝试其他音质")
}
