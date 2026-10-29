package tag

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-flac/flacvorbis/v2"
	flac "github.com/go-flac/go-flac/v2"
)

var sampleFrames = []byte{0xFF, 0xF8, 0x69, 0x00, 0x00, 0x00}

// writeMinimalFLAC writes a FLAC file holding an empty STREAMINFO block and a
// frame sync, which is enough for the tagger to parse and rewrite.
func writeMinimalFLAC(t *testing.T, path string) {
	t.Helper()
	data := append([]byte("fLaC"), 0x80, 0x00, 0x00, 0x22)
	data = append(data, make([]byte, 34)...)
	data = append(data, sampleFrames...)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestWriteFLACReplacesFileWithTags(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "song.flac")
	writeMinimalFLAC(t, path)

	if err := Write(path, Info{Title: "Title", Artist: "Artist", Album: "Album"}); err != nil {
		t.Fatal(err)
	}
	assertNoTempFiles(t, dir)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasSuffix(data, sampleFrames) {
		t.Fatal("audio frames missing after tagging")
	}

	file, err := flac.ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	for _, block := range file.Meta {
		if block == nil || block.Type != flac.VorbisComment {
			continue
		}
		comments, err := flacvorbis.ParseFromMetaDataBlock(*block)
		if err != nil {
			t.Fatal(err)
		}
		if got, _ := comments.Get(flacvorbis.FIELD_TITLE); len(got) != 1 || got[0] != "Title" {
			t.Fatalf("title = %v, want [Title]", got)
		}
		return
	}
	t.Fatal("no vorbis comment block in tagged file")
}

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

func TestCleanTempFiles(t *testing.T) {
	dir := t.TempDir()
	stale := []string{".song.flac.tag-123456", ".a.b.mp3.tag-7"}
	keep := []string{"song.flac", ".song.flac.tag-abc", ".song.flac.part", ".tag-1"}
	for _, name := range append(stale, keep...) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	CleanTempFiles(dir)
	for _, name := range stale {
		if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
			t.Fatalf("%q survived cleanup", name)
		}
	}
	for _, name := range keep {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("%q was removed: %v", name, err)
		}
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
