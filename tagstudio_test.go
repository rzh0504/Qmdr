package main

import (
	"testing"

	"github.com/rzh0504/qmdr/internal/qqmusic"
	"github.com/rzh0504/qmdr/internal/tag"
)

// The zero Client holds no credential, so the untouched-lyrics path of
// requestTagLyrics stops at the credential check without network or timers.
func TestRequestTagLyrics(t *testing.T) {
	a := &app{api: &qqmusic.Client{}}
	a.tagStudio = tagStudioState{
		source:         "history",
		selected:       0,
		historyEntries: []tagEntry{{path: "a.flac", songMID: "mid", loaded: true}},
		original:       tag.Info{Lyrics: "file lyrics"},
		draft:          tag.Info{Lyrics: "file lyrics"},
	}

	a.requestTagLyrics()
	if a.tagStudio.confirmLyricsOverwrite {
		t.Fatal("untouched lyrics should fetch without a confirm dialog")
	}

	a.tagStudio.draft.Lyrics = "my edited lyrics"
	a.requestTagLyrics()
	if !a.tagStudio.confirmLyricsOverwrite {
		t.Fatal("edited lyrics should open the confirm dialog first")
	}
}

func TestNumberPairText(t *testing.T) {
	cases := []struct {
		in            string
		number, total int
		hasTotal      bool
	}{
		{"3", 3, 0, false},
		{"3/12", 3, 12, true},
		{" 3 / 12 ", 3, 12, true},
		{"", 0, 0, false},
		{"/12", 0, 12, true},
		{"3/x", 3, 0, false},
		{"3/", 3, 0, false},
	}
	for _, tc := range cases {
		number, total, hasTotal := numberPairText(tc.in)
		if number != tc.number || total != tc.total || hasTotal != tc.hasTotal {
			t.Fatalf("numberPairText(%q) = %d, %d, %v; want %d, %d, %v",
				tc.in, number, total, hasTotal, tc.number, tc.total, tc.hasTotal)
		}
	}
}

func TestNumberText(t *testing.T) {
	if got := numberText(3, 12); got != "3/12" {
		t.Fatalf("numberText(3, 12) = %q, want 3/12", got)
	}
	if got := numberText(3, 0); got != "3" {
		t.Fatalf("numberText(3, 0) = %q, want 3", got)
	}
	if got := numberText(0, 12); got != "" {
		t.Fatalf("numberText(0, 12) = %q, want empty", got)
	}
}
