package playlistcache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/rzh0504/qmdr/internal/fsutil"
	"github.com/rzh0504/qmdr/internal/model"
)

const cacheVersion = 1

// Cache is the last playlists page shown for the signed-in user.
type Cache struct {
	Version   int              `json:"version"`
	UserID    string           `json:"user_id"`
	Playlists []model.Playlist `json:"playlists"`
	Selected  int              `json:"selected"`
	Songs     []model.Song     `json:"songs"`
	UpdatedAt time.Time        `json:"updated_at"`
}

func Path() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "Qmdr", "playlists.json")
}

func Load() (Cache, bool) { return LoadFile(Path()) }

func LoadFile(path string) (Cache, bool) {
	if path == "" {
		return Cache{}, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Cache{}, false
	}
	var cache Cache
	if err := json.Unmarshal(data, &cache); err != nil || cache.Version != cacheVersion {
		return Cache{}, false
	}
	return cache, true
}

func Save(path string, cache Cache) error {
	cache.Version = cacheVersion
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(path, data, 0o644)
}
