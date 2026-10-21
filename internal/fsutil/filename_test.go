package fsutil

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestSanitizeFilename(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"A<B>C:D\"/E\\F|G?H*I", "A_B_C_D__E_F_G_H_I"},
		{"name. ", "name"},
		{"name .", "name"},
		{"...", "未知歌曲"},
		{"CON", "_CON"},
		{"lpt4.txt", "_lpt4.txt"},
		{"normal.mp3", "normal.mp3"},
		{"  spaced  ", "spaced"},
	}
	for _, tc := range cases {
		if got := SanitizeFilename(tc.name); got != tc.want {
			t.Errorf("SanitizeFilename(%q) = %q, want %q", tc.name, got, tc.want)
		}
	}

	long := strings.Repeat("字", 200) + ".  "
	trimmed := []rune(SanitizeFilename(long))
	if len(trimmed) > 160 {
		t.Fatalf("sanitized length = %d, want at most 160", len(trimmed))
	}
	if strings.HasSuffix(SanitizeFilename(long), " ") || strings.HasSuffix(SanitizeFilename(long), ".") {
		t.Fatalf("sanitized %q keeps a trailing dot or space", SanitizeFilename(long))
	}
}

func TestSafePath(t *testing.T) {
	base := t.TempDir()
	got, err := SafePath(base, "song.mp3")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(base, "song.mp3"); got != want {
		t.Fatalf("SafePath = %q, want %q", got, want)
	}
	if _, err := SafePath(base, filepath.Join("..", "escape.mp3")); err == nil {
		t.Fatal("SafePath accepted a path outside the base")
	}
	if _, err := SafePath("", "song.mp3"); err == nil {
		t.Fatal("SafePath accepted an empty base")
	}
}
