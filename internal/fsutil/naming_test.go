package fsutil

import (
	"path/filepath"
	"testing"
)

func TestRenderNameTemplate(t *testing.T) {
	cases := []struct {
		name     string
		template string
		artist   string
		album    string
		title    string
		want     string
	}{
		{"default", "{artist} - {title}", "周深", "花开忘忧", "花开忘忧", "周深 - 花开忘忧"},
		{"nested", "{artist}/{album}/{title}", "周深", "花开忘忧", "花开忘忧", filepath.Join("周深", "花开忘忧", "花开忘忧")},
		{"backslashes", `{artist}\{title}`, "周深", "", "花开忘忧", filepath.Join("周深", "花开忘忧")},
		{"value slash stays in segment", "{artist} - {title}", "AC/DC", "", "Thunder", "AC_DC - Thunder"},
		{"empty album drops segment", "{artist}/{album}/{title}", "周深", "", "花开忘忧", filepath.Join("周深", "花开忘忧")},
		{"empty artist trims separator", "{artist} - {title}", "", "", "花开忘忧", "花开忘忧"},
		{"reserved name escaped", "{title}", "x", "y", "CON", "_CON"},
		{"dot segment contained", "{artist}/../{title}", "周深", "", "花开忘忧", filepath.Join("周深", "未知歌曲", "花开忘忧")},
	}
	for _, tc := range cases {
		got, err := RenderNameTemplate(tc.template, tc.artist, tc.album, tc.title)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", tc.name, err)
		}
		if got != tc.want {
			t.Fatalf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestRenderNameTemplateRejectsEmpty(t *testing.T) {
	for _, template := range []string{"", "   ", "///", "{artist}/{album}"} {
		got, err := RenderNameTemplate(template, "", "", "")
		if err == nil || got != "" {
			t.Fatalf("RenderNameTemplate(%q) = %q, %v; want an error", template, got, err)
		}
	}
}

func TestRenderNameTemplateStaysInsideBase(t *testing.T) {
	base := t.TempDir()
	rel, err := RenderNameTemplate("{artist}/{album}/{title}", "..", "..", "..")
	if err != nil {
		t.Fatal(err)
	}
	path, err := SafePath(base, rel+".flac")
	if err != nil {
		t.Fatal(err)
	}
	if rel, err := filepath.Rel(base, path); err != nil || rel == ".." {
		t.Fatalf("rendered path escaped the download directory: %q", path)
	}
}
