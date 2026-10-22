package explorecache

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/rzh0504/qmdr/internal/model"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "explore.json")
	cache := Cache{
		Day:         "2026-10-22",
		HotSearches: []string{"周杰伦", "华语流行"},
		Charts: []Chart{{
			ID:    26,
			Name:  "热歌榜",
			Songs: []model.Song{{Mid: "a", Title: "A"}},
		}},
		UpdatedAt: time.Unix(1700000000, 0),
	}
	if err := Save(path, cache); err != nil {
		t.Fatal(err)
	}
	loaded, ok := LoadFile(path)
	if !ok {
		t.Fatal("cache did not load")
	}
	if loaded.Day != cache.Day || len(loaded.HotSearches) != 2 ||
		len(loaded.Charts) != 1 || loaded.Charts[0].ID != 26 ||
		loaded.Charts[0].Songs[0].Mid != "a" || loaded.Version != cacheVersion {
		t.Fatalf("loaded cache = %+v", loaded)
	}
}

func TestLoadFileRejectsIncompleteCache(t *testing.T) {
	path := filepath.Join(t.TempDir(), "explore.json")
	if err := Save(path, Cache{Day: "2026-10-22"}); err != nil {
		t.Fatal(err)
	}
	if _, ok := LoadFile(path); ok {
		t.Fatal("cache without charts loaded")
	}
}

func TestDay(t *testing.T) {
	loc := time.FixedZone("UTC+8", 8*60*60)
	at := time.Date(2026, 10, 22, 23, 59, 0, 0, loc)
	if got := Day(at); got != "2026-10-22" {
		t.Fatalf("Day = %q", got)
	}
}
