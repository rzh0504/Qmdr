package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPendingStateRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pending.json")
	songs := []Song{
		{Mid: "a", Title: "A"},
		{Mid: "a", Title: "duplicate"},
		{Mid: "", Title: "missing mid"},
		{Mid: "b", Title: "B"},
	}
	if err := savePendingFile(path, songs); err != nil {
		t.Fatal(err)
	}
	got := loadPendingFile(path)
	if len(got) != 2 {
		t.Fatalf("loaded %d songs, want 2", len(got))
	}
	if got[0].Mid != "a" || got[1].Mid != "b" {
		t.Fatalf("loaded mids = %q, %q", got[0].Mid, got[1].Mid)
	}
}

func TestPendingStateCorruptFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pending.json")
	if err := os.WriteFile(path, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := loadPendingFile(path); len(got) != 0 {
		t.Fatalf("corrupt file loaded %d songs, want none", len(got))
	}
}
