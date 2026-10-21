package settings

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFileDefaultsAndValidation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if got := LoadFile(path); got != Defaults() {
		t.Fatalf("missing file = %+v, want defaults", got)
	}

	if err := os.WriteFile(path, []byte(`{"quality":"bad","cover_size":"1 px","concurrency":99,"history_keep":"never"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	got := LoadFile(path)
	if got.Quality != "FLAC 无损" || got.CoverSize != "800 px" || got.Concurrency != 3 || got.HistoryKeep != "90 天" {
		t.Fatalf("invalid values were not normalized: %+v", got)
	}
	if got.DownloadDir == "" {
		t.Fatal("DownloadDir was left empty")
	}
}

func TestHelpers(t *testing.T) {
	if !ValidQuality("FLAC 无损") || ValidQuality("bad") {
		t.Fatal("ValidQuality is wrong")
	}
	if !ValidCoverSize("300 px") || ValidCoverSize("1 px") {
		t.Fatal("ValidCoverSize is wrong")
	}
	if !ValidHistoryKeep("永久") || ValidHistoryKeep("never") {
		t.Fatal("ValidHistoryKeep is wrong")
	}
	if HistoryKeepDays("30 天") != 30 || HistoryKeepDays("永久") != 0 {
		t.Fatal("HistoryKeepDays is wrong")
	}
	if CoverSizeValue("500 px") != 500 || CoverSizeValue("bad") != 800 {
		t.Fatal("CoverSizeValue is wrong")
	}
}
