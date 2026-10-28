package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rzh0504/qmdr/internal/history"
)

func TestRestoreQueueConvertsDownloading(t *testing.T) {
	ctx := context.Background()
	store, err := history.Open(ctx, filepath.Join(t.TempDir(), "qmdr.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	now := time.Now()
	records := []history.QueueRecord{
		{ID: 3, SongMID: "mid-a", Status: int(StatusDownloading), Progress: 0.6, CreatedAt: now},
		{ID: 5, SongMID: "mid-b", Status: int(StatusDone), Progress: 1, CreatedAt: now},
	}
	if err := store.SaveQueueItems(ctx, records); err != nil {
		t.Fatal(err)
	}
	a := &app{history: historyState{store: store}}
	a.restoreQueue(records)

	if len(a.queue.items) != 2 {
		t.Fatalf("restored %d queue items, want 2", len(a.queue.items))
	}
	if a.queue.items[0].Status != StatusWaiting || a.queue.items[0].Progress != 0 {
		t.Fatalf("restored downloading item = %+v, want waiting at 0", a.queue.items[0])
	}
	if a.queue.items[1].Status != StatusDone {
		t.Fatalf("restored done item = %+v, want done", a.queue.items[1])
	}
	if a.queue.seq != 5 {
		t.Fatalf("queueSeq = %d, want 5", a.queue.seq)
	}

	stored, err := store.LoadQueue(ctx, now.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(stored) != 2 || stored[0].ID != 3 || stored[0].Status != int(StatusWaiting) {
		t.Fatalf("stored queue = %+v, want item 3 as waiting and item 5", stored)
	}
	if stored[1].ID != 5 || stored[1].Status != int(StatusDone) {
		t.Fatalf("stored done item = %+v, want done", stored[1])
	}
}

func TestQueueFilters(t *testing.T) {
	a := &app{queue: queueState{items: []QueueItem{
		{ID: 1, Status: StatusWaiting},
		{ID: 2, Status: StatusDownloading},
		{ID: 3, Status: StatusDone},
		{ID: 4, Status: StatusSkipped},
		{ID: 5, Status: StatusFailed},
		{ID: 6, Status: StatusCancelled},
	}}}

	if all := a.filteredQueue(); len(all) != 6 {
		t.Fatalf("filteredQueue(all) returned %d items, want 6", len(all))
	}

	a.queue.filter = 1
	if active := a.filteredQueue(); len(active) != 2 {
		t.Fatalf("filteredQueue(active) returned %d items, want 2", len(active))
	}

	a.queue.filter = 2
	completed := a.filteredQueue()
	if len(completed) != 2 || completed[0].ID != 4 || completed[1].ID != 3 {
		t.Fatalf("filteredQueue(completed) = %+v, want skipped then done, newest first", completed)
	}

	a.queue.filter = 3
	if failed := a.filteredQueue(); len(failed) != 2 {
		t.Fatalf("filteredQueue(failed) returned %d items, want 2", len(failed))
	}
}

func TestClearFailedQueue(t *testing.T) {
	a := &app{queue: queueState{items: []QueueItem{
		{ID: 1, Status: StatusWaiting},
		{ID: 2, Status: StatusFailed, Error: "boom"},
		{ID: 3, Status: StatusCancelled},
		{ID: 4, Status: StatusDone},
	}}}
	a.clearFailed()
	if len(a.queue.items) != 2 || a.queue.items[0].ID != 1 || a.queue.items[1].ID != 4 {
		t.Fatalf("queue after clear = %+v, want waiting item 1 and done item 4", a.queue.items)
	}
	if a.toast != "已清理 2 个失败任务" {
		t.Fatalf("toast = %q", a.toast)
	}

	a = &app{queue: queueState{items: []QueueItem{{ID: 1, Status: StatusDone}}}}
	a.clearFailed()
	if len(a.queue.items) != 1 || a.toast != "没有失败的任务" {
		t.Fatalf("queue = %+v, toast = %q", a.queue.items, a.toast)
	}
}

func TestDownloadClaims(t *testing.T) {
	dir := t.TempDir()
	claims := newDownloadClaims()

	first, result := claims.reserve(dir, "Song", ".flac", false)
	if result != claimOK || first != filepath.Join(dir, "Song.flac") {
		t.Fatalf("first claim = %q, %v", first, result)
	}
	second, result := claims.reserve(dir, "Song", ".flac", false)
	if result != claimOK || second != filepath.Join(dir, "Song (2).flac") {
		t.Fatalf("second claim = %q, %v", second, result)
	}

	existing := filepath.Join(dir, "Existing.flac")
	if err := os.WriteFile(existing, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	claims = newDownloadClaims()
	skipped, result := claims.reserve(dir, "Existing", ".flac", false)
	if result != claimSkip || skipped != existing {
		t.Fatalf("existing claim = %q, %v", skipped, result)
	}

	claims = newDownloadClaims()
	overwritten, result := claims.reserve(dir, "Existing", ".flac", true)
	if result != claimOK || overwritten != existing {
		t.Fatalf("overwrite claim = %q, %v", overwritten, result)
	}

	claims = newDownloadClaims()
	_, result = claims.reserve(dir, "Case", ".flac", false)
	if result != claimOK {
		t.Fatalf("case claim = %v", result)
	}
	lower, result := claims.reserve(dir, "case", ".flac", false)
	if result != claimOK || lower != filepath.Join(dir, "case (2).flac") {
		t.Fatalf("case-insensitive claim = %q, %v", lower, result)
	}
}
