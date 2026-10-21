package qqmusic

import (
	"context"
	"encoding/base64"
)

func (c *Client) Lyric(ctx context.Context, mid string) (string, error) {
	if mid == "" {
		return "", nil
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
		"trans":   0,
		"trans_t": 0,
		"type":    1,
		"songMid": mid,
	}
	var data struct {
		Lyric string `json:"lyric"`
	}
	if err := c.call(ctx, "music.musichallSong.PlayLyricInfo", "GetPlayLyricInfo", params, &data); err != nil {
		return "", err
	}
	if data.Lyric == "" {
		return "", nil
	}
	raw, err := base64.StdEncoding.DecodeString(data.Lyric)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
