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
