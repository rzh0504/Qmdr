package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Settings is what the app stores between runs.
type Settings struct {
	DownloadDir string  `json:"download_dir"`
	Quality     string  `json:"quality"`
	CoverSize   string  `json:"cover_size"`
	Overwrite   bool    `json:"overwrite"`
	Concurrency float64 `json:"concurrency"`
	Appearance  string  `json:"appearance"`
	Background  string  `json:"background"`
	HistoryKeep string  `json:"history_keep"`
	ShowExplore bool    `json:"show_explore"`
}

var (
	QualityOptions     = []string{"臻品母带", "FLAC 无损", "MP3 320K", "MP3 128K"}
	CoverSizeOptions   = []string{"150 px", "300 px", "500 px", "800 px"}
	HistoryKeepOptions = []string{"30 天", "90 天", "180 天", "365 天", "永久"}
)

func Path() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "Qmdr", "settings.json")
}

func DefaultDownloadDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(home, "Music", "Qmdr")
}

func Defaults() Settings {
	return Settings{
		DownloadDir: DefaultDownloadDir(),
		Quality:     "FLAC 无损",
		CoverSize:   "800 px",
		Concurrency: 3,
		Appearance:  "light",
		Background:  "rex",
		HistoryKeep: "90 天",
	}
}

// Load reads the settings file at Path, falling back to values that make
// the app usable when it is missing or corrupt.
func Load() Settings { return LoadFile(Path()) }

// LoadFile reads path and normalizes the fields whose valid values this
// package knows. Appearance and Background are left to the UI layer.
func LoadFile(path string) Settings {
	s := Defaults()
	if path == "" {
		return s
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return s
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return Defaults()
	}
	if s.DownloadDir == "" {
		s.DownloadDir = DefaultDownloadDir()
	}
	if !ValidQuality(s.Quality) {
		s.Quality = "FLAC 无损"
	}
	if !ValidCoverSize(s.CoverSize) {
		s.CoverSize = "800 px"
	}
	if s.Concurrency < 1 || s.Concurrency > 8 {
		s.Concurrency = 3
	}
	if !ValidHistoryKeep(s.HistoryKeep) {
		s.HistoryKeep = "90 天"
	}
	return s
}

func ValidQuality(value string) bool     { return contains(QualityOptions, value) }
func ValidCoverSize(value string) bool   { return contains(CoverSizeOptions, value) }
func ValidHistoryKeep(value string) bool { return contains(HistoryKeepOptions, value) }

func contains(options []string, value string) bool {
	for _, option := range options {
		if value == option {
			return true
		}
	}
	return false
}

func HistoryKeepDays(value string) int {
	switch value {
	case "30 天":
		return 30
	case "90 天":
		return 90
	case "180 天":
		return 180
	case "365 天":
		return 365
	default:
		return 0
	}
}

func CoverSizeValue(value string) int {
	value = strings.TrimSpace(strings.TrimSuffix(strings.ToLower(value), "px"))
	size, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 800
	}
	switch size {
	case 150, 300, 500, 800:
		return size
	default:
		return 800
	}
}
