package main

import (
	"log"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/rzh0504/qmdr/internal/pending"
	"github.com/rzh0504/qmdr/internal/qqmusic"
)

type viewID int

const (
	viewSearch viewID = iota
	viewPlaylists
	viewQueue
	viewHistory
	viewTagStudio
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

	view         viewID
	downloadsTab viewID
	closed       atomic.Bool
	toast        string
	updating     bool
	aboutOpen    bool
	updateInfo   updatePrompt

	diskJobs chan func()

	search      searchState
	playlists   playlistState
	queue       queueState
	history     historyState
	tagStudio   tagStudioState
	covers      coverState
	metaCovers  metadataCoverState
	login       loginState
	contextMenu contextMenuState

	quality      string
	downloadDir  string
	coverSize    string
	concurrency  float64
	overwrite    bool
	appearance   string
	background   string
	historyKeep  string
	showExplore  bool
	nameTemplate string
	lyricsTrans  bool

	// sidebarCollapsed keeps the nav as an icon rail; persisted in settings.
	sidebarCollapsed bool

	// settingsSaveTimer debounces the settings write while the user types
	// in a settings field; it is only touched on the UI thread.
	settingsSaveTimer *time.Timer
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
	cfg := loadSettings()
	brandIcon, _ := ui.DecodeBitmap(appIconPNG)
	a := &app{
		api:          api,
		brandIcon:    brandIcon,
		view:         viewPlaylists,
		downloadsTab: viewQueue,
		covers:       coverState{cache: map[string]coverEntry{}, loading: map[string]bool{}, sem: make(chan struct{}, coverFetchLimit)},
		metaCovers:   metadataCoverState{cache: map[string]metadataCover{}, pending: map[string]*metadataCoverCall{}},
		queue:        queueState{pending: pending.Load()},
		playlists:    playlistState{owner: userID, selected: -1, selectedMids: map[string]bool{}},
		history:      historyState{loading: true},
		login:        loginState{loggedIn: api.HasCredential(), userID: displayUser, loginType: loginType},
		quality:      cfg.Quality,
		downloadDir:  cfg.DownloadDir,
		coverSize:    cfg.CoverSize,
		concurrency:  cfg.Concurrency,
		overwrite:    cfg.Overwrite,
		appearance:   cfg.Appearance,
		background:   cfg.Background,
		historyKeep:  cfg.HistoryKeep,
		showExplore:  cfg.ShowExplore,
		nameTemplate: cfg.NameTemplate,
		lyricsTrans:  cfg.LyricsTrans,

		sidebarCollapsed: cfg.SidebarCollapsed,
	}
	a.startDiskWorker()
	a.restorePlaylistCache()
	a.restoreExploreCache()
	api.SetCredentialExpiredHandler(a.handleCredentialExpired)
	return a
}

// handleCredentialExpired resets the signed-in UI after the server rejects
// the saved credential, so every page stops showing stale account data.
func (a *app) handleCredentialExpired() {
	a.update(func() {
		a.clearAccountState()
		a.toast = "登录已过期，请重新登录"
	})
}

// clearAccountState drops the account and everything fetched for it.
func (a *app) clearAccountState() {
	a.login.loggedIn = false
	a.login.userID = "未登录"
	a.login.loginType = "QQ"
	a.resetUserProfile()
	a.playlists.owner = ""
	a.playlists.resetFetches()
	a.playlists.loading = false
	a.playlists.refreshing = false
	a.playlists.loadingSongs = false
	a.playlists.musicIDOpen = false
	a.playlists.musicIDInput = ""
	a.playlists.musicIDErr = ""
	a.playlists.err = ""
	a.playlists.notice = ""
	a.playlists.items = nil
	a.playlists.selected = -1
	a.playlists.songs = nil
	a.resetSongSelection()
}

// logout signs out after the user confirms; the saved credential is deleted
// so a restart does not sign back in.
func (a *app) logout() {
	err := a.api.Logout()
	a.clearAccountState()
	if err != nil {
		a.toast = "已退出登录，但本地凭证删除失败：" + friendlyError(err)
	} else {
		a.toast = "已退出登录并清除本地凭据"
	}
	a.refresh()
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

func (a *app) queueCounts() (total, done, skipped, failed, active int) {
	total = len(a.queue.items)
	for _, item := range a.queue.items {
		switch item.Status {
		case StatusDone:
			done++
		case StatusSkipped:
			skipped++
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
	case viewQueue, viewHistory:
		return "downloads"
	case viewTagStudio:
		return "tagstudio"
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
		if a.view == viewHistory {
			a.resetHistorySelection()
		}
		a.view = viewQueue
		a.downloadsTab = viewQueue
	case "history":
		a.view = viewHistory
		a.downloadsTab = viewHistory
		a.pruneHistoryNow()
	case "downloads":
		if a.downloadsTab != viewQueue && a.downloadsTab != viewHistory {
			a.downloadsTab = viewQueue
		}
		a.view = a.downloadsTab
		if a.view == viewHistory {
			a.pruneHistoryNow()
		}
	case "tagstudio":
		a.view = viewTagStudio
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
