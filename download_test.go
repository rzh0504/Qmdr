package main

import (
	"os"
	"path/filepath"
	"testing"
)

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
