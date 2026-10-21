package history

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestStoreRecordsStatsAndClear(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "qmdr.db")
	store, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	base := time.Unix(1700000000, 0)
	first, err := store.Add(ctx, Record{
		SongMID:     "mid-a",
		Title:       "A",
		Artist:      "Artist",
		Album:       "Album",
		AlbumMID:    "album",
		Quality:     "FLAC",
		FilePath:    `C:\Music\a.flac`,
		FileSize:    1024,
		Status:      StatusDone,
		CompletedAt: base,
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.Add(ctx, Record{
		SongMID:     "mid-b",
		Title:       "B",
		Quality:     "MP3 320K",
		FilePath:    `C:\Music\b.mp3`,
		FileSize:    2048,
		Status:      StatusSkipped,
		CompletedAt: base.Add(time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}

	records, err := store.Recent(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 {
		t.Fatalf("Recent returned %d records, want 2", len(records))
	}
	if records[0].ID != second || records[1].ID != first {
		t.Fatalf("records are not newest first: %+v", records)
	}
	if records[0].Status != StatusSkipped || records[1].FileSize != 1024 {
		t.Fatalf("unexpected records: %+v", records)
	}

	stats, err := store.Stats(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Count != 2 || stats.Bytes != 3072 {
		t.Fatalf("stats = %+v, want count 2 bytes 3072", stats)
	}

	if err := store.Clear(ctx); err != nil {
		t.Fatal(err)
	}
	stats, err = store.Stats(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Count != 0 || stats.Bytes != 0 {
		t.Fatalf("stats after clear = %+v, want empty", stats)
	}

	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	stats, err = reopened.Stats(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Count != 0 {
		t.Fatalf("reopened history has %d records, want 0", stats.Count)
	}
}
