package qqmusic

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Chart is a toplist with its songs. UpdateTime is the day the list was
// last refreshed, as the server writes it.
type Chart struct {
	Title      string
	UpdateTime string
	Total      int
	Songs      []Song
}

// Chart reads the first songs of a toplist, such as 热歌榜 (26).
func (c *Client) Chart(ctx context.Context, topID, limit int) (Chart, error) {
	if topID <= 0 {
		return Chart{}, errors.New("榜单 ID 无效")
	}
	if limit <= 0 || limit > 50 {
		limit = 5
	}
	var data struct {
		Data struct {
			Title      string `json:"title"`
			UpdateTime string `json:"updateTime"`
			Total      int    `json:"totalNum"`
		} `json:"data"`
		Songs []songItem `json:"songInfoList"`
	}
	params := map[string]any{
		"topId":    topID,
		"offset":   0,
		"num":      limit,
		"withTags": true,
	}
	if err := c.call(ctx, "music.musicToplist.Toplist", "GetDetail", params, &data); err != nil {
		return Chart{}, err
	}
	songs := make([]Song, 0, len(data.Songs))
	for _, item := range data.Songs {
		songs = append(songs, item.song())
	}
	return Chart{
		Title:      cleanText(data.Data.Title),
		UpdateTime: data.Data.UpdateTime,
		Total:      data.Data.Total,
		Songs:      songs,
	}, nil
}

type hotKeyItem struct {
	Title string `json:"title"`
	Query string `json:"query"`
}

// HotSearches reads the hot search terms, as the mobile search page shows
// them. Terms repeat between the two lists, so the result is deduplicated.
func (c *Client) HotSearches(ctx context.Context, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 8
	}
	var data struct {
		Hot    []hotKeyItem `json:"vec_hotkey"`
		Reckey []hotKeyItem `json:"vec_reckey"`
	}
	params := map[string]any{"search_id": fmt.Sprintf("%d", time.Now().UnixMilli())}
	if err := c.call(ctx, "music.musicsearch.HotkeyService", "GetHotkeyForQQMusicMobile", params, &data); err != nil {
		return nil, err
	}
	out := make([]string, 0, limit)
	seen := make(map[string]bool, len(data.Hot)+len(data.Reckey))
	add := func(item hotKeyItem) {
		if len(out) >= limit {
			return
		}
		text := strings.TrimSpace(cleanText(item.Title))
		if text == "" {
			text = strings.TrimSpace(cleanText(item.Query))
		}
		if text == "" || seen[text] {
			return
		}
		seen[text] = true
		out = append(out, text)
	}
	for _, item := range data.Hot {
		add(item)
	}
	for _, item := range data.Reckey {
		add(item)
	}
	return out, nil
}
