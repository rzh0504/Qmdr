package playlistcache

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/rzh0504/qmdr/internal/model"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "playlists.json")
	cache := Cache{
		UserID:    "123",
		Playlists: []model.Playlist{{TID: 1, DirID: 201, Name: "我喜欢", Count: 2}},
		Selected:  0,
		Songs:     []model.Song{{Mid: "a", Title: "A"}, {Mid: "b", Title: "B"}},
		UpdatedAt: time.Unix(1700000000, 0),
	}
	if err := Save(path, cache); err != nil {
		t.Fatal(err)
	}
	loaded, ok := LoadFile(path)
	if !ok {
		t.Fatal("cache did not load")
	}
	if loaded.UserID != cache.UserID || len(loaded.Playlists) != 1 || len(loaded.Songs) != 2 || loaded.Version != cacheVersion {
		t.Fatalf("loaded cache = %+v", loaded)
	}
}
