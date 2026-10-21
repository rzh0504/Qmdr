package pending

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rzh0504/qmdr/internal/model"
)

func TestStateRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pending.json")
	songs := []model.Song{
		{Mid: "a", Title: "A"},
		{Mid: "a", Title: "duplicate"},
		{Mid: "", Title: "missing mid"},
		{Mid: "b", Title: "B"},
	}
	if err := SaveFile(path, songs); err != nil {
		t.Fatal(err)
	}
	got := LoadFile(path)
	if len(got) != 2 {
		t.Fatalf("loaded %d songs, want 2", len(got))
	}
	if got[0].Mid != "a" || got[1].Mid != "b" {
		t.Fatalf("loaded mids = %q, %q", got[0].Mid, got[1].Mid)
	}
}

func TestCorruptFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pending.json")
	if err := os.WriteFile(path, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := LoadFile(path); len(got) != 0 {
		t.Fatalf("corrupt file loaded %d songs, want none", len(got))
	}
}
