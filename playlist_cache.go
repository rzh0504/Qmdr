package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
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
	a.playlists = cache.Playlists
	a.selectedPlaylist = cache.Selected
	a.playlistSongs = cache.Songs
	if a.selectedPlaylist < 0 || a.selectedPlaylist >= len(a.playlists) || len(a.playlistSongs) == 0 {
		a.selectedPlaylist = -1
		a.playlistSongs = nil
	}
}

func (a *app) savePlaylistCache() {
	path := playlistCachePath()
	if path == "" {
		return
	}
	userID := strings.TrimSpace(a.musicID)
	if userID == "" || userID == "未登录" {
		userID = a.api.UserID()
	}
	if userID == "" {
		return
	}
	selected := a.selectedPlaylist
	if selected < 0 || selected >= len(a.playlists) {
		selected = -1
	}
	cache := playlistCache{
		Version:   playlistCacheVersion,
		UserID:    userID,
		Playlists: a.playlists,
		Selected:  selected,
		Songs:     a.playlistSongs,
		UpdatedAt: time.Now(),
	}
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0o644)
}
