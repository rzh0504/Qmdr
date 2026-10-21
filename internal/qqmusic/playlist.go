package qqmusic

import (
	"context"
	"errors"
)

type Playlist struct {
	DirID     int64
	TID       int64
	Name      string
	SongCount int
}

func (c *Client) UserPlaylists(ctx context.Context, uin string) ([]Playlist, error) {
	if !c.HasCredential() {
		return nil, errors.New("需要登录后才能读取歌单")
	}
	if uin == "" {
		uin = c.UserID()
	}
	if uin == "" {
		return nil, errors.New("缺少用户 ID")
	}
	var data struct {
		Playlists []struct {
			Name      string `json:"dirName"`
			DirID     int64  `json:"dirId"`
			TID       int64  `json:"tid"`
			SongCount int    `json:"songNum"`
		} `json:"v_playlist"`
	}
	if err := c.call(ctx, "music.musicasset.PlaylistBaseRead", "GetPlaylistByUin",
		map[string]any{"uin": uin}, &data); err != nil {
		return nil, err
	}
	out := make([]Playlist, 0, len(data.Playlists))
	for _, item := range data.Playlists {
		name := cleanText(item.Name)
		if name == "" {
			name = "未命名歌单"
		}
		out = append(out, Playlist{
			DirID:     item.DirID,
			TID:       item.TID,
			Name:      name,
			SongCount: item.SongCount,
		})
	}
	return out, nil
}

// PlaylistPage is the songs of one playlist. Total is the count the
// server reported; Truncated says the playlist holds more songs than the
// read cap, so Songs is only a prefix of it.
type PlaylistPage struct {
	Songs     []Song
	Total     int
	Truncated bool
}

func (c *Client) PlaylistSongs(ctx context.Context, tid, dirID int64) (PlaylistPage, error) {
	if !c.HasCredential() {
		return PlaylistPage{}, errors.New("需要登录后才能读取歌单")
	}
	if tid == 0 {
		return PlaylistPage{}, errors.New("歌单缺少 TID")
	}
	const pageSize = 100
	const maxSongs = 2000

	var songs []Song
	var total int
	for offset := 0; offset < maxSongs; offset += pageSize {
		var data struct {
			Total    int        `json:"total_song_num"`
			Songlist []songItem `json:"songlist"`
		}
		params := map[string]any{
			"disstid":      tid,
			"dirid":        dirID,
			"tag":          false,
			"song_begin":   offset,
			"song_num":     pageSize,
			"userinfo":     false,
			"orderlist":    false,
			"onlysonglist": true,
		}
		if err := c.call(ctx, "music.srfDissInfo.DissInfo", "CgiGetDiss", params, &data); err != nil {
			return PlaylistPage{}, err
		}
		total = data.Total
		for _, item := range data.Songlist {
			songs = append(songs, item.song())
		}
		if len(data.Songlist) < pageSize {
			break
		}
		if total > 0 && len(songs) >= total {
			break
		}
	}
	if len(songs) > maxSongs {
		songs = songs[:maxSongs]
	}
	return PlaylistPage{
		Songs:     songs,
		Total:     total,
		Truncated: len(songs) >= maxSongs && (total <= 0 || total > maxSongs),
	}, nil
}
