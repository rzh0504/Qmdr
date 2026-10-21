package tag

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteAtomicReplacesOnSuccess(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "song.mp3")
	if err := os.WriteFile(path, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeAtomic(path, func(tmp string) error {
		return copyFile(path, tmp) // the MP3 flow: copy then rewrite
	}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "original" {
		t.Fatalf("file = %q, want the copied content", data)
	}
	assertNoTempFiles(t, dir)
}

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

func TestCopyFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	dst := filepath.Join(dir, "dst")
	want := []byte("some audio bytes")
	if err := os.WriteFile(src, want, 0o644); err != nil {
		t.Fatal(err)
	}
	// Copy over a non-empty target: the copy must truncate it.
	if err := os.WriteFile(dst, []byte("stale longer content"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(src, dst); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(want) {
		t.Fatalf("copied = %q, want %q", data, want)
	}
	if err := copyFile(filepath.Join(dir, "missing"), dst); err == nil {
		t.Fatal("copyFile of a missing source succeeded")
	}
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