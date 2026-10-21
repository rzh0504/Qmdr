package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type appSettings struct {
	DownloadDir string  `json:"download_dir"`
	Quality     string  `json:"quality"`
	CoverSize   string  `json:"cover_size"`
	Overwrite   bool    `json:"overwrite"`
	Concurrency float64 `json:"concurrency"`
	Appearance  string  `json:"appearance"`
	Background  string  `json:"background"`
}

func settingsPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "Qmdr", "settings.json")
}

func loadSettings() appSettings {
	settings := appSettings{
		DownloadDir: defaultDownloadDir(),
		Quality:     "FLAC 无损",
		CoverSize:   "800 px",
		Concurrency: 3,
		Appearance:  "light",
		Background:  "rex",
	}
	path := settingsPath()
	if path == "" {
		return settings
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return settings
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		return appSettings{
			DownloadDir: defaultDownloadDir(),
			Quality:     "FLAC 无损",
			CoverSize:   "800 px",
			Concurrency: 3,
			Appearance:  "light",
			Background:  "rex",
		}
	}
	if settings.DownloadDir == "" {
		settings.DownloadDir = defaultDownloadDir()
	}
	if !validQuality(settings.Quality) {
		settings.Quality = "FLAC 无损"
	}
	if !validCoverSize(settings.CoverSize) {
		settings.CoverSize = "800 px"
	}
	if settings.Concurrency < 1 || settings.Concurrency > 8 {
		settings.Concurrency = 3
	}
	if !validAppearance(settings.Appearance) {
		settings.Appearance = "light"
	}
	if !validBackground(settings.Background) {
		settings.Background = "rex"
	}
	return settings
}

func (a *app) saveSettings() {
	path := settingsPath()
	if path == "" {
		return
	}
	settings := appSettings{
		DownloadDir: a.downloadDir,
		Quality:     a.quality,
		CoverSize:   a.coverSize,
		Overwrite:   a.overwrite,
		Concurrency: a.concurrency,
		Appearance:  a.appearance,
		Background:  a.background,
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0o644)
}

func validQuality(value string) bool {
	for _, option := range qualityOptions {
		if value == option {
			return true
		}
	}
	return false
}

func validCoverSize(value string) bool {
	for _, option := range coverSizeOptions {
		if value == option {
			return true
		}
	}
	return false
}
