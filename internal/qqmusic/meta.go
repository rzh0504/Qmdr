package qqmusic

import (
	"context"
	"errors"
	"strings"
)

// SongMeta is the extra tagging detail the song detail page carries.
type SongMeta struct {
	Language    string
	Genre       string
	ReleaseDate string
}

type detailContent struct {
	Value string `json:"value"`
}

// SongMeta reads the language, genre and release date of a song. Fields the
// service does not know stay empty.
func (c *Client) SongMeta(ctx context.Context, mid string) (SongMeta, error) {
	if mid == "" {
		return SongMeta{}, errors.New("歌曲缺少 MID")
	}
	var data struct {
		Info struct {
			Genre struct {
				Content []detailContent `json:"content"`
			} `json:"genre"`
			Lan struct {
				Content []detailContent `json:"content"`
			} `json:"lan"`
			PubTime struct {
				Content []detailContent `json:"content"`
			} `json:"pub_time"`
		} `json:"info"`
	}
	if err := c.call(ctx, "music.pf_song_detail_svr", "get_song_detail_yqq",
		map[string]any{"song_mid": mid}, &data); err != nil {
		return SongMeta{}, err
	}
	return SongMeta{
		Language:    firstDetailValue(data.Info.Lan.Content),
		Genre:       firstDetailValue(data.Info.Genre.Content),
		ReleaseDate: dateOnly(firstDetailValue(data.Info.PubTime.Content)),
	}, nil
}

func firstDetailValue(items []detailContent) string {
	for _, item := range items {
		if value := cleanText(item.Value); value != "" {
			return value
		}
	}
	return ""
}

// SongCredits is who wrote the song, as the producer credits list them.
type SongCredits struct {
	Lyricist string
	Composer string
}

// SongCredits reads the lyricist and composer of a song from its producer
// credits. Missing groups stay empty.
func (c *Client) SongCredits(ctx context.Context, mid string) (SongCredits, error) {
	if mid == "" {
		return SongCredits{}, errors.New("歌曲缺少 MID")
	}
	var data struct {
		Groups []struct {
			Title     string `json:"Title"`
			Producers []struct {
				Name string `json:"Name"`
			} `json:"Producers"`
		} `json:"Lst"`
	}
	if err := c.call(ctx, "music.sociality.KolWorksTag", "SongProducer",
		map[string]any{"songMid": mid}, &data); err != nil {
		return SongCredits{}, err
	}
	var credits SongCredits
	for _, group := range data.Groups {
		names := make([]string, 0, len(group.Producers))
		for _, producer := range group.Producers {
			if name := cleanText(producer.Name); name != "" {
				names = append(names, name)
			}
		}
		if len(names) == 0 {
			continue
		}
		value := strings.Join(names, " / ")
		switch strings.TrimSpace(group.Title) {
		case "作词":
			credits.Lyricist = value
		case "作曲":
			credits.Composer = value
		}
	}
	return credits, nil
}
