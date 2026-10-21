package history

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
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

	removed, err := store.Prune(ctx, base.Add(30*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if removed != 1 {
		t.Fatalf("Prune removed %d records, want 1", removed)
	}
	stats, err = store.Stats(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Count != 1 || stats.Bytes != 2048 {
		t.Fatalf("stats after prune = %+v, want count 1 bytes 2048", stats)
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

func TestQueueStorage(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "qmdr.db")
	store, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	now := time.Now()
	old := QueueRecord{
		ID:          1,
		SongMID:     "mid-old",
		Title:       "Old",
		Artist:      "Artist",
		Album:       "Album",
		AlbumMID:    "album",
		Duration:    "03:00",
		SongQuality: "无损",
		VIP:         true,
		Status:      5,
		Progress:    0.5,
		Quality:     "FLAC",
		Error:       "cancelled",
		FilePath:    `C:\Music\old.flac`,
		FileSize:    4096,
		CreatedAt:   now.Add(-2 * time.Hour),
	}
	fresh := QueueRecord{
		ID:        2,
		SongMID:   "mid-new",
		Title:     "New",
		Status:    2,
		Progress:  1,
		Quality:   "MP3 320K",
		FilePath:  `C:\Music\new.mp3`,
		FileSize:  2048,
		CreatedAt: now,
	}
	if err := store.SaveQueueItems(ctx, []QueueRecord{old, fresh}); err != nil {
		t.Fatal(err)
	}

	loaded, err := store.LoadQueue(ctx, now.Add(-24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 2 || loaded[0].ID != 1 || loaded[1].ID != 2 {
		t.Fatalf("LoadQueue = %+v, want ids 1 and 2", loaded)
	}
	if !loaded[0].VIP || loaded[0].Progress != 0.5 || loaded[0].FileSize != 4096 {
		t.Fatalf("unexpected first record: %+v", loaded[0])
	}

	old.Status = 0
	old.Error = ""
	if err := store.SaveQueueItem(ctx, old); err != nil {
		t.Fatal(err)
	}
	loaded, err = store.LoadQueue(ctx, now.Add(-24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if loaded[0].Status != 0 || loaded[0].Error != "" {
		t.Fatalf("updated record = %+v, want waiting with no error", loaded[0])
	}

	if err := store.PruneQueue(ctx, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	loaded, err = store.LoadQueue(ctx, now.Add(-24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 1 || loaded[0].ID != 2 {
		t.Fatalf("after prune = %+v, want only id 2", loaded)
	}

	if err := store.DeleteQueueItems(ctx, []int{2}); err != nil {
		t.Fatal(err)
	}
	loaded, err = store.LoadQueue(ctx, now.Add(-24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 0 {
		t.Fatalf("after delete = %+v, want empty", loaded)
	}

	if err := store.ReplaceQueue(ctx, []QueueRecord{old}); err != nil {
		t.Fatal(err)
	}
	loaded, err = store.LoadQueue(ctx, now.Add(-24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 1 || loaded[0].SongMID != "mid-old" {
		t.Fatalf("after replace = %+v, want old record", loaded)
	}
}

func TestLogStorage(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "qmdr.db")
	store, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	base := time.Now().Add(-2 * time.Hour)
	if err := store.AddLog(ctx, LogRecord{At: base, Level: "info", Message: "first"}); err != nil {
		t.Fatal(err)
	}
	if err := store.AddLog(ctx, LogRecord{At: base.Add(time.Minute), Level: "warning", Message: "second"}); err != nil {
		t.Fatal(err)
	}

	records, err := store.LogsSince(ctx, base.Add(-time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 || records[0].Message != "first" || records[1].Level != "warning" {
		t.Fatalf("LogsSince = %+v, want first then warning second", records)
	}

	if err := store.PruneLogs(ctx, base.Add(30*time.Second)); err != nil {
		t.Fatal(err)
	}
	records, err = store.LogsSince(ctx, base.Add(-time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0].Message != "second" {
		t.Fatalf("after prune = %+v, want only second", records)
	}
}

func TestStoreCloseDuringUse(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "qmdr.db"))
	if err != nil {
		t.Fatal(err)
	}

	// Operations and a Close never touch the database at the same time:
	// each operation either completes or reports ErrStoreClosed.
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				_, err := store.Add(ctx, Record{SongMID: "mid", Title: "T", FilePath: "x.flac"})
				cancel()
				if err != nil && !errors.Is(err, ErrStoreClosed) {
					t.Errorf("Add failed with %v, want nil or ErrStoreClosed", err)
					return
				}
			}
		}()
	}
	for i := 0; i < 25; i++ {
		time.Sleep(time.Millisecond)
		// Repeated closes are safe and wait for running operations.
		if err := store.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	}
	wg.Wait()

	// Operations after Close report ErrStoreClosed instead of touching a
	// closed database.
	if _, err := store.Add(ctx, Record{}); !errors.Is(err, ErrStoreClosed) {
		t.Fatalf("Add after Close = %v, want ErrStoreClosed", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}
