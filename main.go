package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/rzh0504/qmdr/internal/qqmusic"
	"github.com/skip2/go-qrcode"
)

type viewID int

const (
	viewSearch viewID = iota
	viewPlaylists
	viewQueue
	viewSettings
)

type app struct {
	win *mygo.Window
	api *qqmusic.Client

	view   viewID
	closed bool

	query       string
	quality     string
	searching   bool
	hasSearched bool
	searchErr   string
	searchID    int
	results     []Song

	musicID          string
	loadingPlaylists bool
	playlists        []Playlist
	selectedPlaylist int
	playlistSongs    []Song

	queue           []QueueItem
	queueFilter     int
	queueSeq        int
	downloading     bool
	overallProgress float64
	currentTaskID   int
	runID           int
	cancel          context.CancelFunc
	logs            []LogEntry

	downloadDir string
	coverSize   string
	overwrite   bool
	externalAPI string

	loggedIn  bool
	userID    string
	loginType string

	qrOpen   bool
	qrType   string
	qrStatus string
	qrBitmap *ui.Bitmap

	toast string
}

func newApp() *app {
	api := qqmusic.New()
	userID := api.UserID()
	loginType := "QQ"
	if api.Credential.LoginType == 1 {
		loginType = "微信"
	}
	if userID == "" {
		userID = "未登录"
	}
	a := &app{
		api:              api,
		view:             viewSearch,
		quality:          "FLAC 无损",
		musicID:          userID,
		playlists:        samplePlaylists(),
		selectedPlaylist: 0,
		downloadDir:      defaultDownloadDir(),
		coverSize:        "800 px",
		loggedIn:         api.HasCredential(),
		userID:           userID,
		loginType:        loginType,
	}
	a.playlistSongs = a.playlists[0].Songs
	a.recalcOverallProgress()
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
	col := themeColors(c)
	if a.toast != "" {
		msg := a.toast
		a.toast = ""
		c.Toast(msg)
	}

	ui.Row(c).Fill().AlignItems(ui.Stretch).Background(col.bg).Children(func() {
		a.sidebar(c, col)
		ui.Column(c).Grow(1).MinWidth(0).Children(func() {
			ui.Scroll(c).Grow(1).Padding(16, 24, 24).Gap(16).Children(func() {
				switch a.view {
				case viewSearch:
					a.searchView(c, col)
				case viewPlaylists:
					a.playlistsView(c, col)
				case viewQueue:
					a.queueView(c, col)
				case viewSettings:
					a.settingsView(c, col)
				}
			})
		})
	})

	a.qrModal(c, col)
}

func (a *app) update(fn func()) {
	if a.closed || a.win == nil {
		return
	}
	a.win.Update(fn)
}

func (a *app) refresh() {
	a.update(func() {})
}

func (a *app) setToast(message string) {
	a.toast = message
	a.refresh()
}

func (a *app) appendLog(message, level string) {
	a.logs = append(a.logs, LogEntry{
		Time:    time.Now().Format("15:04:05"),
		Message: message,
		Level:   level,
	})
	if len(a.logs) > 120 {
		a.logs = a.logs[len(a.logs)-120:]
	}
}

func (a *app) recalcOverallProgress() {
	total := len(a.queue)
	if total == 0 {
		a.overallProgress = 0
		return
	}
	sum := 0.0
	for _, item := range a.queue {
		switch item.Status {
		case StatusDone, StatusSkipped, StatusFailed, StatusCancelled:
			sum++
		case StatusDownloading:
			sum += item.Progress
		}
	}
	a.overallProgress = sum / float64(total)
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
	case "settings":
		a.view = viewSettings
	default:
		a.view = viewSearch
	}
}

func (a *app) sidebar(c *ui.Context, col colors) {
	side := ui.Column(c).Width(200).FillHeight().Background(col.sidebar).
		BorderWidth(0, 1, 0, 0).BorderColor(col.border)

	side.Children(func() {
		ui.Row(c).Padding(12, 14).Gap(8).AlignItems(ui.Center).Children(func() {
			a.logoMark(c, col, 26)
			ui.Text(c, "Qmdr").FontSize(14).Bold().TextColor(col.text).SingleLine()
		})

		page := a.pageKey()
		_, _, _, active := a.queueCounts()
		nav := ui.Sidebar(c, &page, func() {
			ui.SidebarItem(c, "search", iconSearch, "搜索下载")
			ui.SidebarItem(c, "playlists", iconLibrary, "歌单下载")
			item := ui.SidebarItem(c, "queue", iconQueue, "下载队列")
			if active > 0 {
				item.Children(func() {
					ui.Badge(c, fmt.Sprint(active))
				})
			}
			ui.SidebarItem(c, "settings", iconSettings, "设置")
		}).Grow(1).Label("导航")
		if nav.Changed() {
			a.setPage(page)
			a.refresh()
		}

		divider(c, col)

		account := ui.ButtonBase(c).Padding(8, 12).Gap(8).
			AlignItems(ui.Center).Label("账号设置")
		if account.Hovered() {
			account = account.Background(col.cardHover)
		}
		if account.Clicked() {
			a.view = viewSettings
			a.refresh()
		}
		account.Children(func() {
			a.avatar(c, col, 24)
			ui.Column(c).Gap(1).Grow(1).Children(func() {
				status := "未登录"
				if a.loggedIn {
					status = "已登录"
				}
				ui.Text(c, status).FontSize(10).Bold().TextColor(col.text).SingleLine()
				ui.Text(c, a.userID).FontSize(9).TextColor(col.muted).SingleLine()
			})
			ui.Icon(c, iconChevronRight).FontSize(12).TextColor(col.faint)
		})
	})
}

func (a *app) logoMark(c *ui.Context, col colors, size float32) {
	ui.Box(c).Size(size, size).Radius(size * 0.24).Background(col.accent).
		Children(func() {
			ui.Row(c).Fill().Center().Children(func() {
				ui.Icon(c, iconMusic).FontSize(size * 0.46).TextColor(col.accentText)
			})
		})
}

func (a *app) avatar(c *ui.Context, col colors, size float32) {
	ui.Box(c).Size(size, size).Radius(size/2).
		Background(col.cardAlt).Border(1, col.border).
		Children(func() {
			ui.Row(c).Fill().Center().Children(func() {
				ui.Text(c, "Q").FontSize(size * 0.38).Bold().TextColor(col.text)
			})
		})
}

func (a *app) openQR(kind string) {
	a.qrType = kind
	a.qrOpen = true
	a.qrStatus = "请使用手机" + kind + "扫码"
	a.qrBitmap = nil
	png, err := qrcode.Encode("Qmdr UI demo · "+strings.ToUpper(kind), qrcode.Medium, 512)
	if err == nil {
		if bmp, err := ui.DecodeBitmap(png); err == nil {
			a.qrBitmap = bmp
		}
	}
	a.refresh()
}

func (a *app) qrModal(c *ui.Context, col colors) {
	ui.Modal(c, &a.qrOpen, func() {
		panel := ui.Column(c).Width(340).Padding(20).Gap(14).Radius(12).
			Background(col.card).Border(1, col.border).
			Shadow(0, 16, 48, 0, ui.RGBA(0, 0, 0, 0.18))
		panel.Children(func() {
			ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
				ui.Column(c).Gap(3).Grow(1).Children(func() {
					ui.Textf(c, "%s登录", a.qrType).FontSize(18).Bold().TextColor(col.text)
					ui.Text(c, "真实扫码登录将在下一步接入，这里是 UI 演示").FontSize(11).TextColor(col.muted)
				})
				iconButton(c, col, iconX, "关闭", func() {
					a.qrOpen = false
					a.refresh()
				})
			})

			qr := ui.Box(c).Size(204, 204).Radius(8).Padding(10).
				Background(ui.RGB(255, 255, 255)).Center().Border(1, col.border)
			qr.Children(func() {
				if a.qrBitmap != nil {
					ui.Image(c, a.qrBitmap).Size(184, 184)
				} else {
					ui.Spinner(c).Size(24, 24)
				}
			})

			ui.Text(c, a.qrStatus).FontSize(12).TextColor(col.muted)
			ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
				secondaryButton(c, col, nil, "关闭", func() {
					a.qrOpen = false
					a.refresh()
				})
			})
		})
	})
}

func main() {
	mygo.App.SetName("Qmdr")
	mygo.Theme.SetSource(mygo.ThemeLight)
	a := newApp()

	mygo.App.WhenReady(func() {
		if err := os.MkdirAll(a.downloadDir, 0o755); err != nil {
			a.downloadDir = defaultDownloadDir()
		}
		a.win = mygo.NewWindow(mygo.WindowOptions{
			Title:           "Qmdr · QQ 音乐下载器",
			Width:           1200,
			Height:          800,
			MinWidth:        1000,
			MinHeight:       680,
			BackgroundColor: "#ffffff",
			Content:         ui.View(a.viewRoot),
		})
		a.win.OnClosed(func() {
			a.closed = true
			if a.cancel != nil {
				a.cancel()
			}
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
