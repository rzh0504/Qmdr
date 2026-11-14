package main

import (
	"encoding/json"
	"time"

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
		DownloadDir:      a.downloadDir,
		Quality:          a.quality,
		CoverSize:        a.coverSize,
		Overwrite:        a.overwrite,
		Concurrency:      a.concurrency,
		Appearance:       a.appearance,
		Background:       a.background,
		HistoryKeep:      a.historyKeep,
		ShowExplore:      a.showExplore,
		NameTemplate:     a.nameTemplate,
		LyricsTrans:      a.lyricsTrans,
		SidebarCollapsed: a.sidebarCollapsed,
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

// settingsSaveDebounce is how long a settings field waits after the last
// keystroke before the settings are written.
const settingsSaveDebounce = 500 * time.Millisecond

// scheduleSettingsSave groups keystrokes in a settings field into one
// write: the save fires once the input settles instead of once per key.
// The timer only runs on the UI thread, like every other touch of the
// settings fields.
func (a *app) scheduleSettingsSave() {
	if a.settingsSaveTimer != nil {
		a.settingsSaveTimer.Stop()
	}
	a.settingsSaveTimer = time.AfterFunc(settingsSaveDebounce, func() {
		a.update(func() {
			a.settingsSaveTimer = nil
			a.saveSettings()
		})
	})
}

// saveSettingsNow cancels a pending debounced save and writes immediately,
// for explicit submissions like pressing Enter in a field.
func (a *app) saveSettingsNow() {
	if a.settingsSaveTimer != nil {
		a.settingsSaveTimer.Stop()
		a.settingsSaveTimer = nil
	}
	a.saveSettings()
}
