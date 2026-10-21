package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rzh0504/qmdr/internal/fsutil"
)

const playlistCacheVersion = 1

type playlistCache struct {
	Version   int        `json:"version"`
	UserID    string     `json:"user_id"`
	Playlists []Playlist `json:"playlists"`
	Selected  int        `json:"selected"`
	Songs     []Song     `json:"songs"`
	UpdatedAt time.Time  `json:"updated_at"`
}

func playlistCachePath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "Qmdr", "playlists.json")
}

func loadPlaylistCache() (playlistCache, bool) {
	path := playlistCachePath()
	if path == "" {
		return playlistCache{}, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return playlistCache{}, false
	}
	var cache playlistCache
	if err := json.Unmarshal(data, &cache); err != nil || cache.Version != playlistCacheVersion {
		return playlistCache{}, false
	}
	return cache, true
}

func (a *app) restorePlaylistCache() {
	cache, ok := loadPlaylistCache()
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
	path := playlistCachePath()
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
	// Snapshot on the caller's thread; the slices are replaced wholesale,
	// never mutated, so the worker can marshal them safely.
	cache := playlistCache{
		Version:   playlistCacheVersion,
		UserID:    owner,
		Playlists: a.playlists.items,
		Selected:  selected,
		Songs:     a.playlists.songs,
		UpdatedAt: time.Now(),
	}
	a.dispatchDisk(func() {
		data, err := json.MarshalIndent(cache, "", "  ")
		if err != nil {
			return
		}
		if err := fsutil.WriteFileAtomic(path, data, 0o644); err != nil {
			a.update(func() {
				a.appendLog("歌单缓存保存失败："+friendlyError(err), "warning")
			})
		}
	})
}
