package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestQueueSections(t *testing.T) {
	a := &app{queue: []QueueItem{
		{ID: 1, Status: StatusWaiting},
		{ID: 2, Status: StatusDownloading},
		{ID: 3, Status: StatusDone},
		{ID: 4, Status: StatusSkipped},
		{ID: 5, Status: StatusFailed},
		{ID: 6, Status: StatusCancelled},
	}}

	completed := a.completedQueue()
	if len(completed) != 2 || completed[0].ID != 3 || completed[1].ID != 4 {
		t.Fatalf("completedQueue = %+v, want done and skipped", completed)
	}

	all := a.filteredQueue()
	if len(all) != 4 {
		t.Fatalf("filteredQueue(all) returned %d items, want 4", len(all))
	}
	for _, item := range all {
		if item.Status == StatusDone || item.Status == StatusSkipped {
			t.Fatalf("filteredQueue(all) contains completed item %+v", item)
		}
	}

	a.queueFilter = 1
	if active := a.filteredQueue(); len(active) != 2 {
		t.Fatalf("filteredQueue(active) returned %d items, want 2", len(active))
	}

	a.queueFilter = 2
	if failed := a.filteredQueue(); len(failed) != 2 {
		t.Fatalf("filteredQueue(failed) returned %d items, want 2", len(failed))
	}
}

func TestFormatFileSize(t *testing.T) {
	cases := []struct {
		size int64
		want string
	}{
		{0, ""},
		{-1, ""},
		{512, "512 B"},
		{1 << 10, "1 KB"},
		{3 << 20, "3.0 MB"},
		{3 << 30, "3.0 GB"},
	}
	for _, tc := range cases {
		if got := formatFileSize(tc.size); got != tc.want {
			t.Errorf("formatFileSize(%d) = %q, want %q", tc.size, got, tc.want)
		}
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
