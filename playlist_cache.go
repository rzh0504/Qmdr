package main

import (
	"strings"
	"time"

	"github.com/rzh0504/qmdr/internal/playlistcache"
)

func (a *app) restorePlaylistCache() {
	cache, ok := playlistcache.Load()
	if !ok || !a.api.HasCredential() || cache.UserID != a.api.UserID() || len(cache.Playlists) == 0 {
		return
	}
	a.playlists.owner = a.api.UserID()
	a.playlists.items = cache.Playlists
	a.playlists.selected = cache.Selected
	a.playlists.songs = cache.Songs
	if a.playlists.selected < 0 || a.playlists.selected >= len(a.playlists.items) || len(a.playlists.songs) == 0 {
		a.playlists.selected = -1
		a.playlists.songs = nil
	}
}

func (a *app) savePlaylistCache() {
	path := playlistcache.Path()
	if path == "" {
		return
	}
	owner := strings.TrimSpace(a.playlists.owner)
	if owner == "" {
		owner = a.api.UserID()
	}
	if owner == "" || owner == "未登录" || owner != a.api.UserID() {
		return
	}
	selected := a.playlists.selected
	if selected < 0 || selected >= len(a.playlists.items) {
		selected = -1
	}
	cache := playlistcache.Cache{
		UserID:    owner,
		Playlists: a.playlists.items,
		Selected:  selected,
		Songs:     a.playlists.songs,
		UpdatedAt: time.Now(),
	}
	a.dispatchDisk(func() {
		if err := playlistcache.Save(path, cache); err != nil {
			a.update(func() {
				a.appendLog("歌单缓存保存失败："+friendlyError(err), "warning")
			})
		}
	})
}
