package explorecache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/rzh0504/qmdr/internal/fsutil"
	"github.com/rzh0504/qmdr/internal/model"
)

const cacheVersion = 1

// Chart is one cached toplist shown on the search page.
type Chart struct {
	ID    int          `json:"id"`
	Name  string       `json:"name"`
	Songs []model.Song `json:"songs"`
}

// Cache is the explore content fetched on a given local day.
type Cache struct {
	Version     int       `json:"version"`
	Day         string    `json:"day"`
	HotSearches []string  `json:"hot_searches"`
	Charts      []Chart   `json:"charts"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Day formats t's local calendar day, the cache's freshness key.
func Day(t time.Time) string { return t.Format("2006-01-02") }

func Path() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "Qmdr", "explore.json")
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
	if err := json.Unmarshal(data, &cache); err != nil ||
		cache.Version != cacheVersion || cache.Day == "" || len(cache.Charts) == 0 {
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
