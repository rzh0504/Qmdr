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

func TestTagMatchQuery(t *testing.T) {
	a := &app{}
	tagged := &tagEntry{title: "花开忘忧", artist: "周深"}
	if q := a.tagMatchQuery(tagged); q != "花开忘忧 周深" {
		t.Fatalf("tagMatchQuery(tagged) = %q, want 花开忘忧 周深", q)
	}
	noisy := &tagEntry{title: "【无损】花开忘忧 [HQ]"}
	if q := a.tagMatchQuery(noisy); q != "花开忘忧" {
		t.Fatalf("tagMatchQuery(noisy) = %q, want 花开忘忧", q)
	}
	untagged := &tagEntry{path: "E:\\Music\\r1cky - Demo.flac"}
	if q := a.tagMatchQuery(untagged); q != "r1cky - Demo" {
		t.Fatalf("tagMatchQuery(untagged) = %q, want r1cky - Demo", q)
	}
}

func TestApplyTagBatchFill(t *testing.T) {
	info := tag.Info{Title: "已有标题", Artist: "   "}
	filled := applyTagBatchFill(&info,
		qqmusic.SongMeta{ReleaseDate: "2024-01-01", Genre: "流行"},
		qqmusic.SongCredits{Composer: "某人"},
		"", "周深", "专辑")
	if filled != 5 {
		t.Fatalf("filled = %d, want 5", filled)
	}
	if info.Title != "已有标题" {
		t.Fatalf("Title = %q, want unchanged", info.Title)
	}
	if info.Artist != "周深" || info.Album != "专辑" || info.Date != "2024-01-01" ||
		info.Genre != "流行" || info.Composer != "某人" {
		t.Fatalf("info = %+v, want filled fields", info)
	}
	if info.Language != "" || info.Lyricist != "" {
		t.Fatalf("empty sources should leave fields empty: %+v", info)
	}
}
