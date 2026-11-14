package tag

import (
	"bytes"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"reflect"
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

func TestWriteFLACExtraTags(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "song.flac")
	writeMinimalFLAC(t, path)

	if err := Write(path, Info{
		Title:       "Title",
		Artist:      "Artist",
		Album:       "Album",
		TrackNumber: 3,
		DiscNumber:  1,
		Date:        "2022-09-10",
		Genre:       "流行",
		Language:    "国语",
		Composer:    "钱雷",
		Lyricist:    "唐恬",
	}); err != nil {
		t.Fatal(err)
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
		want := map[string]string{
			flacvorbis.FIELD_TRACKNUMBER: "3",
			"DISCNUMBER":                 "1",
			flacvorbis.FIELD_DATE:        "2022-09-10",
			flacvorbis.FIELD_GENRE:       "流行",
			"LANGUAGE":                   "国语",
			"COMPOSER":                   "钱雷",
			"LYRICIST":                   "唐恬",
		}
		for key, value := range want {
			got, _ := comments.Get(key)
			if len(got) != 1 || got[0] != value {
				t.Fatalf("%s = %v, want [%s]", key, got, value)
			}
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

// tinyPNG is a 1x1 transparent image, small enough to embed in tests.
var tinyPNG, _ = base64.StdEncoding.DecodeString(
	"iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==")

func fullInfo() Info {
	return Info{
		Title:       "夜曲",
		Artist:      "周杰伦",
		Album:       "十一月的萧邦",
		Lyrics:      "[00:01.00]一群嗜血的蚂蚁",
		Cover:       append([]byte(nil), tinyPNG...),
		CoverMIME:   "image/png",
		TrackNumber: 3,
		TrackTotal:  12,
		DiscNumber:  1,
		DiscTotal:   2,
		Date:        "2005-11-01",
		Genre:       "流行",
		Language:    "国语",
		Composer:    "周杰伦",
		Lyricist:    "方文山",
	}
}

func TestFLACReadWriteRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "song.flac")
	writeMinimalFLAC(t, path)

	want := fullInfo()
	if err := Write(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("read = %+v, want %+v", got, want)
	}

	cleared := want
	cleared.Title = ""
	cleared.Cover = nil
	cleared.CoverMIME = ""
	cleared.Lyrics = ""
	if err := Write(path, cleared); err != nil {
		t.Fatal(err)
	}
	got, err = Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "" || got.Cover != nil || got.Lyrics != "" {
		t.Fatalf("cleared fields came back: %+v", got)
	}
}

func TestFLACWriteKeepsUnknownComments(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "song.flac")
	writeMinimalFLAC(t, path)
	if err := Write(path, Info{Title: "One", Artist: "Artist"}); err != nil {
		t.Fatal(err)
	}
	appendFLACComments(t, path, "ALBUMARTIST", "Various", "REPLAYGAIN_TRACK_GAIN", "-6.2 dB")

	if err := Write(path, Info{Title: "Two", Artist: "Artist"}); err != nil {
		t.Fatal(err)
	}
	comments := readFLACComments(t, path)
	if got := comments["TITLE"]; len(got) != 1 || got[0] != "Two" {
		t.Fatalf("title = %v, want [Two]", got)
	}
	if got := comments["ALBUMARTIST"]; len(got) != 1 || got[0] != "Various" {
		t.Fatalf("ALBUMARTIST = %v, want [Various]", got)
	}
	if got := comments["REPLAYGAIN_TRACK_GAIN"]; len(got) != 1 || got[0] != "-6.2 dB" {
		t.Fatalf("REPLAYGAIN_TRACK_GAIN = %v, want [-6.2 dB]", got)
	}
}

func appendFLACComments(t *testing.T, path string, pairs ...string) {
	t.Helper()
	file, err := flac.ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	block, err := flacvorbis.ParseFromMetaDataBlock(*file.Meta[len(file.Meta)-1])
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i+1 < len(pairs); i += 2 {
		if err := block.Add(pairs[i], pairs[i+1]); err != nil {
			t.Fatal(err)
		}
	}
	marshaled := block.Marshal()
	file.Meta[len(file.Meta)-1] = &marshaled
	tmp := path + ".rewrite"
	out, err := os.Create(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteTo(out); err != nil {
		_ = out.Close()
		t.Fatal(err)
	}
	if err := out.Close(); err != nil {
		t.Fatal(err)
	}
	_ = file.Close()
	if err := os.Rename(tmp, path); err != nil {
		t.Fatal(err)
	}
}

func readFLACComments(t *testing.T, path string) map[string][]string {
	t.Helper()
	file, err := flac.ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	for _, meta := range file.Meta {
		if meta == nil || meta.Type != flac.VorbisComment {
			continue
		}
		comments, err := flacvorbis.ParseFromMetaDataBlock(*meta)
		if err != nil {
			t.Fatal(err)
		}
		values := map[string][]string{}
		for _, comment := range comments.Comments {
			index := strings.IndexByte(comment, '=')
			if index < 0 {
				continue
			}
			key := strings.ToUpper(comment[:index])
			values[key] = append(values[key], comment[index+1:])
		}
		return values
	}
	t.Fatal("no vorbis comment block")
	return nil
}

func TestMP3ReadWriteRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "song.mp3")
	// The tagger only reads and prepends the ID3 tag, so the audio bytes
	// below merely need to survive the rewrite. They must span the ID3
	// header size or the library treats the file as a broken tag.
	audio := bytes.Repeat([]byte{0x00}, 64)
	copy(audio, []byte{0xFF, 0xFB, 0x90, 0x64})
	if err := os.WriteFile(path, audio, 0o644); err != nil {
		t.Fatal(err)
	}

	want := fullInfo()
	if err := Write(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("read = %+v, want %+v", got, want)
	}

	cleared := want
	cleared.Title = ""
	cleared.Cover = nil
	cleared.CoverMIME = ""
	cleared.Lyrics = ""
	if err := Write(path, cleared); err != nil {
		t.Fatal(err)
	}
	got, err = Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "" || got.Cover != nil || got.Lyrics != "" {
		t.Fatalf("cleared fields came back: %+v", got)
	}
}

func TestUnsupportedFormat(t *testing.T) {
	path := filepath.Join(t.TempDir(), "song.m4a")
	if _, err := Read(path); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("Read error = %v, want ErrUnsupported", err)
	}
	if err := Write(path, Info{Title: "Title"}); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("Write error = %v, want ErrUnsupported", err)
	}
	if Supported(path) {
		t.Fatal("Supported(m4a) = true")
	}
}
