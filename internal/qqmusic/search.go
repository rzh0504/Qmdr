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
	AlbumMid        string
	DurationSeconds int
	VIP             bool
	Quality         string
}

type songItem struct {
	ID       int64  `json:"id"`
	Mid      string `json:"mid"`
	Title    string `json:"title"`
	Name     string `json:"name"`
	Interval int    `json:"interval"`
	Singer   []struct {
		Name string `json:"name"`
	} `json:"singer"`
	Album struct {
		Name string `json:"name"`
		Mid  string `json:"mid"`
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

func (item songItem) song() Song {
	artists := make([]string, 0, len(item.Singer))
	for _, singer := range item.Singer {
		if name := cleanText(singer.Name); name != "" {
			artists = append(artists, name)
		}
	}
	title := item.Title
	if title == "" {
		title = item.Name
	}
	quality := "128K"
	switch {
	case item.File.SizeFLAC > 0:
		quality = "无损"
	case item.File.Size320MP3 > 0:
		quality = "320K"
	}
	return Song{
		ID:              item.ID,
		Mid:             item.Mid,
		Title:           cleanText(title),
		Artists:         artists,
		Album:           cleanText(item.Album.Name),
		AlbumMid:        item.Album.Mid,
		DurationSeconds: item.Interval,
		VIP:             item.Pay.PayPlay != 0,
		Quality:         quality,
	}
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
			ItemSong []songItem `json:"item_song"`
		} `json:"body"`
	}
	module, method := "music.search.SearchCgiService", "DoSearchForQQMusicMobile"
	if err := c.call(ctx, module, method, params, &data); err != nil {
		return nil, err
	}

	songs := make([]Song, 0, len(data.Body.ItemSong))
	for _, item := range data.Body.ItemSong {
		songs = append(songs, item.song())
	}
	return songs, nil
}

func cleanText(s string) string {
	s = strings.NewReplacer("<em>", "", "</em>", "", "<EM>", "", "</EM>", "").Replace(s)
	return strings.TrimSpace(html.UnescapeString(s))
}
