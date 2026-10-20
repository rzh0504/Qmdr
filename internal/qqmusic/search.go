package qqmusic

import (
	"context"
	"fmt"
	"html"
	"strings"
	"time"
)

type Song struct {
	ID              int64
	Mid             string
	Title           string
	Artists         []string
	Album           string
	DurationSeconds int
	VIP             bool
	Quality         string
}

type searchItem struct {
	ID       int64  `json:"id"`
	Mid      string `json:"mid"`
	Title    string `json:"title"`
	Interval int    `json:"interval"`
	Singer   []struct {
		Name string `json:"name"`
	} `json:"singer"`
	Album struct {
		Name string `json:"name"`
	} `json:"album"`
	Pay struct {
		PayPlay int `json:"pay_play"`
	} `json:"pay"`
	File struct {
		SizeFLAC   int64 `json:"size_flac"`
		Size320MP3 int64 `json:"size_320mp3"`
		Size128MP3 int64 `json:"size_128mp3"`
	} `json:"file"`
}

func (c *Client) Search(ctx context.Context, query string, limit int) ([]Song, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("搜索关键词不能为空")
	}
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	params := map[string]any{
		"searchid":     fmt.Sprintf("%d", time.Now().UnixMilli()),
		"query":        query,
		"search_type":  0,
		"num_per_page": limit,
		"page_num":     1,
		"highlight":    false,
		"grp":          true,
	}
	var data struct {
		Body struct {
			ItemSong []searchItem `json:"item_song"`
		} `json:"body"`
	}
	module, method := "music.search.SearchCgiService", "DoSearchForQQMusicMobile"
	if err := c.call(ctx, module, method, params, &data); err != nil {
		return nil, err
	}

	songs := make([]Song, 0, len(data.Body.ItemSong))
	for _, item := range data.Body.ItemSong {
		artists := make([]string, 0, len(item.Singer))
		for _, singer := range item.Singer {
			if name := cleanText(singer.Name); name != "" {
				artists = append(artists, name)
			}
		}
		quality := "128K"
		switch {
		case item.File.SizeFLAC > 0:
			quality = "无损"
		case item.File.Size320MP3 > 0:
			quality = "320K"
		}
		songs = append(songs, Song{
			ID:              item.ID,
			Mid:             item.Mid,
			Title:           cleanText(item.Title),
			Artists:         artists,
			Album:           cleanText(item.Album.Name),
			DurationSeconds: item.Interval,
			VIP:             item.Pay.PayPlay != 0,
			Quality:         quality,
		})
	}
	return songs, nil
}

func cleanText(s string) string {
	s = strings.NewReplacer("<em>", "", "</em>", "", "<EM>", "", "</EM>", "").Replace(s)
	return strings.TrimSpace(html.UnescapeString(s))
}
