package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/rzh0504/qmdr/internal/history"
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
	win *mygo.Window
	api *qqmusic.Client

	covers       map[string]*ui.Bitmap
	coverLoading map[string]bool
	coverFailed  map[string]bool
	coverSem     chan struct{}

	view   viewID
	closed atomic.Bool

	query       string
	quality     string
	searching   bool
	hasSearched bool
	searchErr   string
	searchID    int
	results     []Song

	playlistOwner       string
	loadingPlaylists    bool
	refreshingPlaylists bool
	loadingSongs        bool
	playlistErr         string
	playlistNotice      string
	playlistReq         int
	playlistSongsID     int
	playlists           []Playlist
	selectedPlaylist    int
	playlistSongs       []Song
	selectingSongs      bool
	selectedSongMids    map[string]bool
	songPressKey        string
	songPressSeq        int
	songPressTimer      *time.Timer
	suppressSongClick   bool

	musicIDOpen  bool
	musicIDInput string
	musicIDErr   string

	pending     []Song
	queue       []QueueItem
	queueFilter int
	queueSeq    int
	downloading bool
	runID       int
	cancel      context.CancelFunc
	logBacklog  []LogEntry

	// taskCancels cancels one running download; the run-level cancel above
	// still cancels every task at once.
	taskCancelsMu sync.Mutex
	taskCancels   map[int]context.CancelFunc

	historyStore   *history.Store
	historyItems   []history.Record
	historyBacklog []history.Record
	historyCount   int
	historyBytes   int64
	historyErr     string
	historyLoading bool

	historyKeep string

	downloadDir string
	coverSize   string
	concurrency float64
	overwrite   bool
	appearance  string
	background  string

	loggedIn  bool
	userID    string
	loginType string

	qrOpen   bool
	qrFailed bool
	qrSeq    int
	qrType   string
	qrStatus string
	qrBitmap *ui.Bitmap
	qrCancel context.CancelFunc

	toast    string
	updating bool
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
	a := &app{
		api:              api,
		covers:           make(map[string]*ui.Bitmap),
		coverLoading:     make(map[string]bool),
		coverFailed:      make(map[string]bool),
		coverSem:         make(chan struct{}, coverFetchLimit),
		view:             viewSearch,
		quality:          settings.Quality,
		playlistOwner:    userID,
		selectedPlaylist: -1,
		selectedSongMids: make(map[string]bool),
		pending:          loadPending(),
		historyLoading:   true,
		downloadDir:      settings.DownloadDir,
		coverSize:        settings.CoverSize,
		concurrency:      settings.Concurrency,
		overwrite:        settings.Overwrite,
		appearance:       settings.Appearance,
		background:       settings.Background,
		historyKeep:      settings.HistoryKeep,
		loggedIn:         api.HasCredential(),
		userID:           displayUser,
		loginType:        loginType,
	}
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

func (a *app) viewRoot(c *ui.Context) {
	col := a.themeColors(c)
	if a.toast != "" {
		msg := a.toast
		a.toast = ""
		c.Toast(msg)
	}

	base, glow := a.backdrop(col.dark)
	ui.Column(c).Fill().LinearGradient(base).Children(func() {
		row := ui.Row(c).Fill().AlignItems(ui.Stretch).LinearGradient(glow)
		row.Children(func() {
			a.sidebar(c, col)
			main := ui.Column(c).Grow(1).MinWidth(0).Padding(cardGap, cardGap, cardGap, cardGap)
			main.Children(func() {
				content := ui.Column(c).Fill()
				smoothPanel(content, col, 14, col.card, true)
				content.Children(func() {
					a.cardHeader(c, col)
					if a.view == viewSearch && !a.hasSearched {
						ui.Column(c).Grow(1).MinHeight(0).
							Padding(12, 20, 20, 18).Children(func() {
							a.searchView(c, col)
						})
						return
					}
					if a.view == viewPlaylists {
						ui.Column(c).Grow(1).MinHeight(0).
							Padding(12, 20, 20, 18).Children(func() {
							a.playlistsView(c, col)
						})
						return
					}
					ui.Scroll(c).Grow(1).MinHeight(0).
						Padding(12, 20, 20, 18).Gap(18).Children(func() {
						switch a.view {
						case viewSearch:
							a.searchView(c, col)
						case viewPlaylists:
							a.playlistsView(c, col)
						case viewQueue:
							a.queueView(c, col)
						case viewHistory:
							a.historyView(c, col)
						case viewSettings:
							a.settingsView(c, col)
						}
					})
				})
			})
		})
	})

	a.qrModal(c, col)
	a.musicIDModal(c, col)
}

func (a *app) cardHeader(c *ui.Context, col colors) {
	ui.Row(c).FillWidth().Height(headerH).Shrink(0).AlignItems(ui.Center).
		Padding(0, 8, 0, 16).DragWindow().Children(func() {
		ui.Text(c, a.pageTitle()).FontSize(15).FontWeight(600).TextColor(col.text)
		ui.Spacer(c)
		a.headerActions(c, col)
		if runtime.GOOS != "darwin" {
			ui.Box(c).Width(6)
			a.windowControls(c, col)
		}
	})
}

func (a *app) pageTitle() string {
	switch a.view {
	case viewPlaylists:
		return "歌单"
	case viewQueue:
		return "队列"
	case viewHistory:
		return "历史"
	case viewSettings:
		return "设置"
	default:
		return "搜索"
	}
}

func (a *app) headerActions(c *ui.Context, col colors) {
	switch a.view {
	case viewSearch, viewHistory, viewSettings:
		iconButton(c, col, iconFolder, "打开下载目录", a.openDownloadDir)
	case viewQueue:
		_, done, _, active := a.queueCounts()
		skipped := 0
		for _, item := range a.queue {
			if item.Status == StatusSkipped {
				skipped++
			}
		}
		if a.downloading {
			ghostDangerButton(c, col, iconX, "取消", a.cancelQueue)
		}
		if !a.downloading && done+skipped > 0 {
			iconButton(c, col, iconTrash, "清理已完成", a.clearCompleted)
		}
		if !a.downloading && active > 0 {
			primaryButton(c, col, iconPlay, "开始", a.startQueue)
		}
	}
}

func (a *app) windowControls(c *ui.Context, col colors) {
	if a.win == nil {
		return
	}
	ui.Row(c).Gap(2).AlignItems(ui.Center).Shrink(0).Children(func() {
		minimize := ui.ButtonBase(c).Size(28, 26).Radius(6).Justify(ui.Center).
			AlignItems(ui.Center).Label("最小化").Tooltip("最小化").TextColor(col.muted)
		if minimize.Pressed() {
			minimize.Background(col.cardHover.Alpha(0.12))
		} else if minimize.Hovered() {
			minimize.Background(col.cardHover).TextColor(col.text)
		}
		minimize.Children(func() {
			ui.Icon(c, iconWinMinimize).Size(15, 15)
		})
		if minimize.Clicked() {
			a.win.Minimize()
		}

		maximized := a.win.IsMaximized()
		label, icon := "最大化", iconWinMaximize
		if maximized {
			label, icon = "还原", iconWinRestore
		}
		maximize := ui.ButtonBase(c).Size(28, 26).Radius(6).Justify(ui.Center).
			AlignItems(ui.Center).Label(label).TextColor(col.muted).MaximizeButton()
		if maximize.Pressed() {
			maximize.Background(col.cardHover.Alpha(0.12))
		} else if maximize.Hovered() {
			maximize.Background(col.cardHover).TextColor(col.text)
		}
		maximize.Children(func() {
			ui.Icon(c, icon).Size(15, 15)
		})
		if maximize.Clicked() {
			if maximized {
				a.win.Unmaximize()
			} else {
				a.win.Maximize()
			}
		}

		closeButton := ui.ButtonBase(c).Size(28, 26).Radius(6).Justify(ui.Center).
			AlignItems(ui.Center).Label("关闭").Tooltip("关闭").TextColor(col.muted)
		if closeButton.Hovered() || closeButton.Pressed() {
			closeButton.Background(col.red).TextColor(ui.RGB(255, 255, 255))
		}
		closeButton.Children(func() {
			ui.Icon(c, iconX).Size(15, 15)
		})
		if closeButton.Clicked() {
			a.win.Close()
		}
	})
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
	if a.historyStore == nil {
		a.logBacklog = append(a.logBacklog, entry)
		if len(a.logBacklog) > 500 {
			a.logBacklog = a.logBacklog[len(a.logBacklog)-500:]
		}
		return
	}
	a.storeLog(entry)
}

func (a *app) queueCounts() (total, done, failed, active int) {
	total = len(a.queue)
	for _, item := range a.queue {
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

func (a *app) sidebar(c *ui.Context, col colors) {
	bar := c.TitleBar()
	side := ui.Column(c).Width(216).Shrink(0).FillHeight().Padding(0, 10, 10, 10)

	side.Children(func() {
		ui.Row(c).Height(topBar).Shrink(0).AlignItems(ui.Center).
			Padding(0, bar.Left+10, 0, 10).Gap(8).DragWindow().Children(func() {
			a.logoMark(c, col, 22)
			ui.Text(c, "Qmdr").FontSize(13).FontWeight(600).TextColor(col.text)
		})

		ui.Column(c).Gap(2).Children(func() {
			ui.Text(c, "资料库").FontSize(11).FontWeight(600).TextColor(col.faint).
				Padding(0, 9, 6, 9)
			a.navItem(c, col, "search", iconSearch, "搜索下载")
			a.navItem(c, col, "playlists", iconLibrary, "歌单下载")
			a.navItem(c, col, "queue", iconQueue, "下载队列")
			a.navItem(c, col, "history", iconClock, "下载历史")
			a.navItem(c, col, "settings", iconSettings, "设置")
		})

		ui.Spacer(c)

		a.accountChip(c, col)
	})
}

func (a *app) navItem(c *ui.Context, col colors, key string, icon *ui.SVG, label string) {
	active := a.pageKey() == key
	row := ui.ButtonBase(c).FillWidth().Height(30).Padding(0, 9).Gap(9).
		Radius(8).AlignItems(ui.Center).Cursor(ui.CursorPointer).Label(label)
	if active {
		row.Background(col.rowSelected)
		row = chosen(row, col)
	} else if row.Hovered() {
		row.Background(col.cardHover)
	}
	if row.Clicked() {
		a.setPage(key)
		a.refresh()
	}
	fg := col.muted
	if active {
		fg = col.text
	}
	row.Children(func() {
		ui.Icon(c, icon).Size(15, 15).TextColor(fg)
		ui.Text(c, label).FontSize(13).FontWeight(400).TextColor(fg).Grow(1)
		if key == "queue" {
			_, _, _, active := a.queueCounts()
			if count := active + len(a.pending); count > 0 {
				ui.Textf(c, "%d", count).FontSize(11).TextColor(col.muted)
			}
		}
	})
}

func (a *app) accountChip(c *ui.Context, col colors) {
	account := ui.Row(c).Height(50).Padding(0, 10).Gap(9).
		AlignItems(ui.Center).Cursor(ui.CursorPointer).Label("账号设置")
	fill := col.card
	if account.Hovered() {
		fill = col.controlHover
	}
	smoothPanel(account, col, 10, fill, false)
	if account.Clicked() {
		a.view = viewSettings
		a.refresh()
	}
	account.Children(func() {
		a.avatar(c, col, 26)
		ui.Column(c).Gap(1).Grow(1).MinWidth(0).Children(func() {
			status := "未登录"
			if a.loggedIn {
				status = "已登录"
			}
			ui.Text(c, status).FontSize(12).FontWeight(400).TextColor(col.text).SingleLine()
			ui.Text(c, a.userID).FontSize(11).TextColor(col.muted).SingleLine().Ellipsis("…")
		})
		ui.Icon(c, iconChevronRight).Size(13, 13).TextColor(col.faint)
	})
}

func (a *app) logoMark(c *ui.Context, col colors, size float32) {
	ui.Box(c).Size(size, size).Radius(size * 0.26).Background(col.accent).
		Children(func() {
			ui.Row(c).Fill().Center().Children(func() {
				ui.Icon(c, iconMusic).Size(size*0.5, size*0.5).TextColor(col.accentText)
			})
		})
}

func (a *app) avatar(c *ui.Context, col colors, size float32) {
	ui.Box(c).Size(size, size).Radius(size / 2).Background(col.chip).
		Children(func() {
			ui.Row(c).Fill().Center().Children(func() {
				ui.Text(c, "Q").FontSize(float32(int(size*0.4 + 0.5))).FontWeight(600).TextColor(col.text)
			})
		})
}

func (a *app) openQR(kind string) {
	if a.qrCancel != nil {
		a.qrCancel()
		a.qrCancel = nil
	}
	a.qrType = kind
	a.qrOpen = true
	a.qrFailed = false
	a.qrStatus = "正在获取二维码…"
	a.qrBitmap = nil
	a.qrSeq++
	session := a.qrSeq
	ctx, cancel := context.WithCancel(context.Background())
	a.qrCancel = cancel
	a.refresh()

	go func() {
		current := func() bool { return a.qrOpen && a.qrSeq == session }
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("qr login panic: %v", recovered)
				a.update(func() {
					if current() {
						a.qrBitmap = nil
						a.qrFailed = true
						a.qrStatus = "登录过程出现异常，请重试"
					}
				})
			}
		}()
		qr, err := func() (*qqmusic.QRLogin, error) {
			startCtx, startCancel := context.WithTimeout(ctx, 20*time.Second)
			defer startCancel()
			return a.api.StartQRLogin(startCtx, kind)
		}()
		if err != nil {
			a.update(func() {
				if current() {
					a.qrFailed = true
					a.qrStatus = "获取二维码失败：" + friendlyError(err)
				}
			})
			return
		}
		bitmap, decodeErr := ui.DecodeBitmap(qr.Image)
		if decodeErr != nil {
			a.update(func() {
				if current() {
					a.qrFailed = true
					a.qrStatus = "二维码解析失败，请重新获取"
				}
			})
			return
		}
		a.update(func() {
			if !current() {
				return
			}
			a.qrBitmap = bitmap
			a.qrStatus = "请使用手机扫码"
		})

		failures := 0
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
			result, err := func() (qqmusic.LoginResult, error) {
				pollCtx, pollCancel := context.WithTimeout(ctx, 15*time.Second)
				defer pollCancel()
				return a.api.PollQRLogin(pollCtx, qr)
			}()
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				failures++
				if failures >= 3 {
					a.update(func() {
						if current() {
							a.qrBitmap = nil
							a.qrFailed = true
							a.qrStatus = "登录检查失败：" + friendlyError(err)
						}
					})
					return
				}
				if failures == 1 {
					a.update(func() {
						if current() {
							a.qrStatus = "网络波动，正在重试…"
						}
					})
				}
				continue
			}
			failures = 0
			done := result.Event == qqmusic.LoginDone
			stop := done || result.Event == qqmusic.LoginExpired || result.Event == qqmusic.LoginRefused
			a.update(func() {
				if !current() {
					return
				}
				a.qrStatus = result.Message
				if done {
					a.finishLogin()
					return
				}
				if stop {
					a.qrBitmap = nil
					a.qrFailed = true
				}
			})
			if stop {
				return
			}
		}
	}()
}

func (a *app) finishLogin() {
	a.loggedIn = a.api.HasCredential()
	a.userID = a.api.UserID()
	if a.userID == "" {
		a.userID = "未登录"
	}
	if a.api.Credential().LoginType == 1 {
		a.loginType = "微信"
	} else {
		a.loginType = "QQ"
	}
	a.playlistOwner = a.api.UserID()
	a.playlistReq++
	a.playlistSongsID++
	a.loadingPlaylists = false
	a.refreshingPlaylists = false
	a.loadingSongs = false
	a.musicIDOpen = false
	a.musicIDInput = ""
	a.musicIDErr = ""
	if a.qrCancel != nil {
		a.qrCancel()
		a.qrCancel = nil
	}
	a.qrOpen = false
	a.qrFailed = false
	a.qrBitmap = nil
	a.qrStatus = ""
	a.playlistErr = ""
	a.playlistNotice = ""
	a.playlists = nil
	a.selectedPlaylist = -1
	a.playlistSongs = nil
	a.resetSongSelection()
	a.restorePlaylistCache()
	if a.api.CredentialSaveError() != nil {
		a.toast = "登录成功，但凭证保存失败，重启后需重新登录"
	} else {
		a.toast = "登录成功"
	}
	a.refresh()
}

func (a *app) closeQR() {
	if a.qrCancel != nil {
		a.qrCancel()
		a.qrCancel = nil
	}
	a.qrOpen = false
	a.qrFailed = false
	a.qrBitmap = nil
	a.qrStatus = ""
	a.refresh()
}

func (a *app) qrModal(c *ui.Context, col colors) {
	ui.DialogBase(c, &a.qrOpen, func(backdrop, panel ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, 0.4))
		panel.Width(336).Padding(18)
		smoothPanel(panel, col, 16, col.card, false)
		panel.Shadow(0, 16, 40, 0, col.floatShadow)
		panel.Children(func() {
			ui.Column(c).FillWidth().Gap(14).Children(func() {
				ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
					ui.Textf(c, "%s登录", a.qrType).FontSize(15).FontWeight(600).
						TextColor(col.text).Grow(1)
					iconButton(c, col, iconX, "关闭", a.closeQR)
				})

				ui.Row(c).FillWidth().Justify(ui.Center).Children(func() {
					qr := ui.Box(c).Size(208, 208).Radius(10).Padding(10).
						Background(ui.RGB(255, 255, 255)).Center().
						Border(1, col.border)
					qr.Children(func() {
						switch {
						case a.qrBitmap != nil:
							ui.Image(c, a.qrBitmap).Size(188, 188)
						case a.qrFailed:
							ui.Icon(c, iconAlert).Size(22, 22).TextColor(col.red)
						default:
							ui.Spinner(c).Size(22, 22)
						}
					})
				})

				statusColor := col.muted
				if a.qrFailed {
					statusColor = col.red
				}
				ui.Text(c, a.qrStatus).FontSize(12).TextColor(statusColor)
				ui.Row(c).Gap(6).Justify(ui.End).Children(func() {
					secondaryButton(c, col, nil, "关闭", a.closeQR)
					if a.qrBitmap != nil || a.qrFailed {
						primaryButton(c, col, iconRefresh, "重新获取", func() {
							a.openQR(a.qrType)
						})
					}
				})
			})
		})
	})
}

func main() {
	mygo.App.SetName("Qmdr")
	mygo.App.SetVersion(appVersion)
	if !mygo.App.RequestSingleInstanceLock() {
		return
	}
	a := newApp()
	mygo.Theme.SetSource(themeSource(a.appearance))
	mygo.App.SetMenu(a.menu())
	mygo.App.OnSecondInstance(func(args []string, workingDir string) {
		if a.win == nil {
			return
		}
		if a.win.IsMinimized() {
			a.win.Restore()
		}
		a.win.Focus()
	})

	mygo.App.WhenReady(func() {
		if err := os.MkdirAll(a.downloadDir, 0o755); err != nil {
			a.downloadDir = defaultDownloadDir()
		}
		options := mygo.WindowOptions{
			Title:           "Qmdr · QQ 音乐下载器",
			Width:           1200,
			Height:          800,
			MinWidth:        1000,
			MinHeight:       680,
			BackgroundColor: a.backgroundCSS(),
			Content:         ui.View(a.viewRoot),
		}
		if runtime.GOOS == "darwin" {
			options.TitleBarStyle = mygo.TitleBarHiddenInset
			options.TitleBarHeight = topBar
			options.TrafficLightPosition = &mygo.Point{X: 18, Y: barMid - 7}
		} else {
			options.TitleBarStyle = mygo.TitleBarHidden
			options.HideWindowButtons = true
			options.AutoHideMenuBar = true
		}
		a.win = mygo.NewWindow(options)
		a.win.OnClosed(func() {
			a.closed.Store(true)
			a.stopSongPress()
			if a.cancel != nil {
				a.cancel()
			}
			if a.qrCancel != nil {
				a.qrCancel()
			}
			if a.historyStore != nil {
				_ = a.historyStore.Close()
			}
		})
		go a.initHistory()
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
