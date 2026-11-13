package qqmusic

import (
	"context"
	"encoding/base64"
)

// Lyrics is a song's lyric with its official translation, when one exists.
type Lyrics struct {
	LRC         string
	Translation string
}

func (c *Client) Lyric(ctx context.Context, mid string) (Lyrics, error) {
	if mid == "" {
		return Lyrics{}, nil
	}
	params := map[string]any{
		"crypt":   0,
		"ct":      11,
		"cv":      versionCode,
		"lrc_t":   0,
		"qrc":     0,
		"qrc_t":   0,
		"roma":    0,
		"roma_t":  0,
		"trans":   1,
		"trans_t": 0,
		"type":    1,
		"songMid": mid,
	}
	var data struct {
		Lyric string `json:"lyric"`
		Trans string `json:"trans"`
	}
	if err := c.call(ctx, "music.musichallSong.PlayLyricInfo", "GetPlayLyricInfo", params, &data); err != nil {
		return Lyrics{}, err
	}
	lrc, err := decodeLyric(data.Lyric)
	if err != nil {
		return Lyrics{}, err
	}
	translation, err := decodeLyric(data.Trans)
	if err != nil {
		translation = ""
	}
	return Lyrics{LRC: lrc, Translation: translation}, nil
}

func decodeLyric(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	raw, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
