package main

import "testing"

func TestAppendSongsSkipsRepeats(t *testing.T) {
	base := []Song{{Mid: "a", Title: "A"}, {Mid: "b", Title: "B"}}
	more := []Song{
		{Mid: "b", Title: "B again"},
		{Mid: "c", Title: "C"},
		{Mid: "", Title: "unknown"},
	}
	got := appendSongs(base, more)
	if len(got) != 4 {
		t.Fatalf("appendSongs returned %d songs, want 4: %+v", len(got), got)
	}
	if got[2].Mid != "c" || got[3].Title != "unknown" {
		t.Fatalf("appendSongs = %+v, want C then the song without a mid", got)
	}
}
