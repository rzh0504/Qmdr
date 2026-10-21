package main

import (
	"encoding/json"

	"github.com/rzh0504/qmdr/internal/fsutil"
	"github.com/rzh0504/qmdr/internal/settings"
)

var qualityOptions = settings.QualityOptions
var coverSizeOptions = settings.CoverSizeOptions
var historyKeepOptions = settings.HistoryKeepOptions

func loadSettings() settings.Settings {
	s := settings.LoadFile(settings.Path())
	if !validAppearance(s.Appearance) {
		s.Appearance = "light"
	}
	if !validBackground(s.Background) {
		s.Background = "rex"
	}
	return s
}

func (a *app) saveSettings() {
	path := settings.Path()
	if path == "" {
		return
	}
	current := settings.Settings{
		DownloadDir: a.downloadDir,
		Quality:     a.quality,
		CoverSize:   a.coverSize,
		Overwrite:   a.overwrite,
		Concurrency: a.concurrency,
		Appearance:  a.appearance,
		Background:  a.background,
		HistoryKeep: a.historyKeep,
		ShowExplore: a.showExplore,
	}
	a.dispatchDisk(func() {
		data, err := json.MarshalIndent(current, "", "  ")
		if err != nil {
			a.update(func() { a.toast = "保存设置失败：" + friendlyError(err) })
			return
		}
		if err := fsutil.WriteFileAtomic(path, data, 0o644); err != nil {
			a.update(func() { a.toast = "保存设置失败：" + friendlyError(err) })
		}
	})
}
