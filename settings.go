package main

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/rzh0504/qmdr/internal/fsutil"
)

type appSettings struct {
	DownloadDir string  `json:"download_dir"`
	Quality     string  `json:"quality"`
	CoverSize   string  `json:"cover_size"`
	Overwrite   bool    `json:"overwrite"`
	Concurrency float64 `json:"concurrency"`
	Appearance  string  `json:"appearance"`
	Background  string  `json:"background"`
	HistoryKeep string  `json:"history_keep"`
}

func settingsPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "Qmdr", "settings.json")
}

func defaultSettings() appSettings {
	return appSettings{
		DownloadDir: defaultDownloadDir(),
		Quality:     "FLAC 无损",
		CoverSize:   "800 px",
		Concurrency: 3,
		Appearance:  "light",
		Background:  "rex",
		HistoryKeep: "90 天",
	}
}

func loadSettings() appSettings {
	settings := defaultSettings()
	path := settingsPath()
	if path == "" {
		return settings
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return settings
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		return defaultSettings()
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
	if !validHistoryKeep(settings.HistoryKeep) {
		settings.HistoryKeep = "90 天"
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
		HistoryKeep: a.historyKeep,
	}
	a.dispatchDisk(func() {
		data, err := json.MarshalIndent(settings, "", "  ")
		if err != nil {
			a.update(func() { a.toast = "保存设置失败：" + friendlyError(err) })
			return
		}
		if err := fsutil.WriteFileAtomic(path, data, 0o644); err != nil {
			a.update(func() { a.toast = "保存设置失败：" + friendlyError(err) })
		}
	})
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

func validHistoryKeep(value string) bool {
	for _, option := range historyKeepOptions {
		if value == option {
			return true
		}
	}
	return false
}

func historyKeepDays(value string) int {
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
