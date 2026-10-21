package main

import (
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/rzh0504/qmdr/internal/qqmusic"
)

type viewID int

const (
	viewSearch viewID = iota
	viewPlaylists
	viewQueue
	viewHistory
	viewSettings
)

const (
	barMid  = 30
	topBar  = barMid * 2
	cardGap = 8
	headerH = (barMid - cardGap) * 2
)

type app struct {
	win       *mygo.Window
	api       *qqmusic.Client
	brandIcon *ui.Bitmap

	view       viewID
	closed     atomic.Bool
	toast      string
	updating   bool
	aboutOpen  bool
	updateInfo updatePrompt

	diskJobs chan func()

	search      searchState
	playlists   playlistState
	queue       queueState
	history     historyState
	covers      coverState
	login       loginState
	contextMenu contextMenuState

	quality     string
	downloadDir string
	coverSize   string
	concurrency float64
	overwrite   bool
	appearance  string
	background  string
	historyKeep string
	showExplore bool
}

func newApp() *app {
	api := qqmusic.New()
	userID := api.UserID()
	loginType := "QQ"
	if api.Credential().LoginType == 1 {
		loginType = "微信"
	}
	displayUser := userID
	if displayUser == "" {
		displayUser = "未登录"
	}
	settings := loadSettings()
	brandIcon, _ := ui.DecodeBitmap(appIconPNG)
	a := &app{
		api:         api,
		brandIcon:   brandIcon,
		view:        viewSearch,
		covers:      coverState{cache: map[string]coverEntry{}, loading: map[string]bool{}, sem: make(chan struct{}, coverFetchLimit)},
		queue:       queueState{pending: loadPending()},
		playlists:   playlistState{owner: userID, selected: -1, selectedMids: map[string]bool{}},
		history:     historyState{loading: true},
		login:       loginState{loggedIn: api.HasCredential(), userID: displayUser, loginType: loginType},
		quality:     settings.Quality,
		downloadDir: settings.DownloadDir,
		coverSize:   settings.CoverSize,
		concurrency: settings.Concurrency,
		overwrite:   settings.Overwrite,
		appearance:  settings.Appearance,
		background:  settings.Background,
		historyKeep: settings.HistoryKeep,
		showExplore: settings.ShowExplore,
	}
	a.startDiskWorker()
	a.restorePlaylistCache()
	return a
}

func defaultDownloadDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(home, "Music", "Qmdr")
}

func (a *app) update(fn func()) {
	if a.closed.Load() || a.win == nil {
		return
	}
	a.win.Update(fn)
}

func (a *app) refresh() {
	a.update(func() {})
}

func (a *app) syncWindowBackground() {
	if a.win != nil {
		_ = a.win.SetBackgroundColor(a.backgroundCSS())
	}
}

func (a *app) setToast(message string) {
	a.toast = message
	a.refresh()
}

func (a *app) appendLog(message, level string) {
	entry := LogEntry{At: time.Now(), Level: level, Message: message}
	if a.history.store == nil {
		a.history.logBacklog = append(a.history.logBacklog, entry)
		if len(a.history.logBacklog) > 500 {
			a.history.logBacklog = a.history.logBacklog[len(a.history.logBacklog)-500:]
		}
		return
	}
	a.storeLog(entry)
}

func (a *app) queueCounts() (total, done, failed, active int) {
	total = len(a.queue.items)
	for _, item := range a.queue.items {
		switch item.Status {
		case StatusDone:
			done++
		case StatusFailed, StatusCancelled:
			failed++
		case StatusWaiting, StatusDownloading:
			active++
		}
	}
	return
}

func (a *app) pageKey() string {
	switch a.view {
	case viewPlaylists:
		return "playlists"
	case viewQueue:
		return "queue"
	case viewHistory:
		return "history"
	case viewSettings:
		return "settings"
	default:
		return "search"
	}
}

func (a *app) setPage(page string) {
	switch page {
	case "playlists":
		a.view = viewPlaylists
	case "queue":
		a.view = viewQueue
	case "history":
		a.view = viewHistory
		a.pruneHistoryNow()
	case "settings":
		a.view = viewSettings
	default:
		a.view = viewSearch
	}
}

// applyAppIcon gives each platform its best icon: Windows reads the
// multi-size ICO embedded with rsrc, macOS uses the PNG for the Dock and
// ships an ICNS with the bundle, and Linux takes the PNG on the window.
func applyAppIcon(win *mygo.Window) {
	switch runtime.GOOS {
	case "windows":
		// The ICO resource gives the crispest taskbar icon; scaling a large
		// PNG into a HICON here is blurrier.
	case "darwin":
		if err := mygo.App.Dock.SetIcon(appIconPNG); err != nil {
			log.Printf("set dock icon: %v", err)
		}
	default:
		if err := win.SetIcon(appIconPNG); err != nil {
			log.Printf("set window icon: %v", err)
		}
	}
}
