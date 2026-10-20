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

func (c *Client) ResolveURL(ctx context.Context, mid, quality string) (ResolvedURL, error) {
	if mid == "" {
		return ResolvedURL{}, errors.New("歌曲缺少 MID")
	}
	for _, ft := range fileTypesForQuality(quality) {
		params := map[string]any{
			"filename": []string{ft.Code + mid + mid + ft.Ext},
			"guid":     randomHex(16),
			"songmid":  []string{mid},
			"songtype": []int{0},
		}
		var data struct {
			Midurlinfo []struct {
				Songmid string `json:"songmid"`
				Wifiurl string `json:"wifiurl"`
				Purl    string `json:"purl"`
			} `json:"midurlinfo"`
		}
		if err := c.call(ctx, "music.vkey.GetVkey", "UrlGetVkey", params, &data); err != nil {
			return ResolvedURL{}, err
		}
		for _, info := range data.Midurlinfo {
			if info.Songmid != "" && info.Songmid != mid {
				continue
			}
			raw := info.Wifiurl
			if raw == "" {
				raw = info.Purl
			}
			if raw == "" {
				continue
			}
			if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
				return ResolvedURL{URL: raw, Ext: ft.Ext, Quality: ft.Name}, nil
			}
			return ResolvedURL{URL: cdnBase + strings.TrimPrefix(raw, "/"), Ext: ft.Ext, Quality: ft.Name}, nil
		}
	}
	if !c.HasCredential() {
		return ResolvedURL{}, errors.New("需要登录后才能获取该歌曲的下载链接")
	}
	return ResolvedURL{}, fmt.Errorf("没有可用的音质，请尝试其他音质")
}
