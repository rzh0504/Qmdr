package tag

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteAtomicKeepsOriginalOnError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "song.flac")
	want := []byte("original audio")
	if err := os.WriteFile(path, want, 0o644); err != nil {
		t.Fatal(err)
	}
	err := writeAtomic(path, func(tmp string) error {
		if err := os.WriteFile(tmp, []byte("half written"), 0o644); err != nil {
			return err
		}
		return errors.New("disk full")
	})
	if err == nil {
		t.Fatal("writeAtomic succeeded, want the injected error")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(want) {
		t.Fatalf("file = %q, want the original untouched", data)
	}
	assertNoTempFiles(t, dir)
}

func assertNoTempFiles(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".tag-") {
			t.Fatalf("leftover temp file %q", entry.Name())
		}
	}
}