package main

import (
	"context"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/rzh0504/qmdr/internal/qqmusic"
	"github.com/rzh0504/qmdr/internal/settings"
)

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
					if a.view == viewSearch && !a.search.searched && !a.showExplore {
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
					if a.fullPageLoading() {
						ui.Column(c).Grow(1).MinHeight(0).
							Padding(12, 20, 20, 18).Children(func() {
							a.viewBody(c, col)
						})
						return
					}
					scroll := ui.Scroll(c).Grow(1).MinHeight(0).
						Padding(12, 20, 20, 18).Gap(18)
					if a.view == viewSearch {
						scroll = scroll.TrackScroll(&a.search.scroll)
					}
					scroll.Children(func() {
						a.viewBody(c, col)
					})
				})
			})
		})
	})

	a.qrModal(c, col)
	a.musicIDModal(c, col)
	a.clearHistoryModal(c, col)
	a.logoutModal(c, col)
	a.aboutModal(c, col)
	a.updateModal(c, col)
	a.menuView(c, col)
}

func (a *app) viewBody(c *ui.Context, col colors) {
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
}

// fullPageLoading reports whether the view shows nothing but a loading state,
// which is centered rather than scrolled from the top.
func (a *app) fullPageLoading() bool {
	switch a.view {
	case viewSearch:
		if !a.search.searched {
			return a.showExplore && len(a.search.charts) == 0 &&
				(!a.search.exploreLoaded || a.search.exploreLoading)
		}
		return a.search.loading
	case viewHistory:
		return a.history.loading && a.history.store == nil
	}
	return false
}

func (a *app) openQR(kind string) {
	if a.login.qrCancel != nil {
		a.login.qrCancel()
		a.login.qrCancel = nil
	}
	a.login.qrType = kind
	a.login.qrOpen = true
	a.login.qrFailed = false
	a.login.qrStatus = "正在获取二维码…"
	a.login.qrBitmap = nil
	a.login.qrSeq++
	session := a.login.qrSeq
	ctx, cancel := context.WithCancel(context.Background())
	a.login.qrCancel = cancel
	a.refresh()

	go func() {
		current := func() bool { return a.login.qrOpen && a.login.qrSeq == session }
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("qr login panic: %v", recovered)
				a.update(func() {
					if current() {
						a.login.qrBitmap = nil
						a.login.qrFailed = true
						a.login.qrStatus = "登录过程出现异常，请重试"
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
					a.login.qrFailed = true
					a.login.qrStatus = "获取二维码失败：" + friendlyError(err)
				}
			})
			return
		}
		bitmap, decodeErr := ui.DecodeBitmap(qr.Image)
		if decodeErr != nil {
			a.update(func() {
				if current() {
					a.login.qrFailed = true
					a.login.qrStatus = "二维码解析失败，请重新获取"
				}
			})
			return
		}
		a.update(func() {
			if !current() {
				return
			}
			a.login.qrBitmap = bitmap
			a.login.qrStatus = "请使用手机扫码"
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
							a.login.qrBitmap = nil
							a.login.qrFailed = true
							a.login.qrStatus = "登录检查失败：" + friendlyError(err)
						}
					})
					return
				}
				if failures == 1 {
					a.update(func() {
						if current() {
							a.login.qrStatus = "网络波动，正在重试…"
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
				a.login.qrStatus = result.Message
				if done {
					a.finishLogin()
					return
				}
				if stop {
					a.login.qrBitmap = nil
					a.login.qrFailed = true
				}
			})
			if stop {
				return
			}
		}
	}()
}

// resetUserProfile forgets the fetched profile when the account changes.
func (a *app) resetUserProfile() {
	a.login.profileSlot.reset()
	a.login.nick = ""
	a.login.avatar = nil
	a.login.profileFor = ""
	a.login.profileLoading = false
	a.login.profileDone = false
}

// loadUserProfile fetches the signed-in user's nickname and avatar once
// per account, quietly keeping the placeholder if it cannot.
func (a *app) loadUserProfile() {
	userID := a.api.UserID()
	if !a.api.HasCredential() || userID == "" {
		return
	}
	if a.login.profileLoading || (a.login.profileDone && a.login.profileFor == userID) {
		return
	}
	a.login.profileLoading = true
	a.login.profileFor = userID
	var nick string
	var avatar *ui.Bitmap
	a.async(&a.login.profileSlot, 20*time.Second,
		func(ctx context.Context) error {
			profile, err := a.api.UserProfile(ctx)
			if err != nil {
				return err
			}
			nick = profile.Nick
			if profile.AvatarURL != "" {
				data, err := a.api.FetchImage(ctx, profile.AvatarURL)
				if err == nil {
					avatar, _ = ui.DecodeBitmap(data)
				}
			}
			return nil
		},
		func(err error) {
			a.login.profileLoading = false
			a.login.profileDone = true
			if a.api.UserID() != userID {
				return
			}
			if err != nil {
				log.Printf("load user profile: %v", err)
				return
			}
			a.login.nick = nick
			if avatar != nil {
				a.login.avatar = avatar
			}
		})
}

func (a *app) finishLogin() {
	a.login.loggedIn = a.api.HasCredential()
	a.login.userID = a.api.UserID()
	if a.login.userID == "" {
		a.login.userID = "未登录"
	}
	if a.api.Credential().LoginType == 1 {
		a.login.loginType = "微信"
	} else {
		a.login.loginType = "QQ"
	}
	a.resetUserProfile()
	a.playlists.owner = a.api.UserID()
	a.playlists.resetFetches()
	a.playlists.loading = false
	a.playlists.refreshing = false
	a.playlists.loadingSongs = false
	a.playlists.musicIDOpen = false
	a.playlists.musicIDInput = ""
	a.playlists.musicIDErr = ""
	if a.login.qrCancel != nil {
		a.login.qrCancel()
		a.login.qrCancel = nil
	}
	a.login.qrOpen = false
	a.login.qrFailed = false
	a.login.qrBitmap = nil
	a.login.qrStatus = ""
	a.playlists.err = ""
	a.playlists.notice = ""
	a.playlists.items = nil
	a.playlists.selected = -1
	a.playlists.songs = nil
	a.resetSongSelection()
	a.restorePlaylistCache()
	a.loadUserProfile()
	if a.api.CredentialSaveError() != nil {
		a.toast = "登录成功，但凭证保存失败，重启后需重新登录"
	} else {
		a.toast = "登录成功"
	}
	a.refresh()
}

func (a *app) closeQR() {
	if a.login.qrCancel != nil {
		a.login.qrCancel()
		a.login.qrCancel = nil
	}
	a.login.qrOpen = false
	a.login.qrFailed = false
	a.login.qrBitmap = nil
	a.login.qrStatus = ""
	a.refresh()
}

func (a *app) qrModal(c *ui.Context, col colors) {
	ui.DialogBase(c, &a.login.qrOpen, func(backdrop, panel ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, 0.4))
		panel.Width(336).Padding(18)
		smoothPanel(panel, col, 16, col.card, false)
		panel.Shadow(0, 16, 40, 0, col.floatShadow)
		panel.Children(func() {
			ui.Column(c).FillWidth().Gap(14).Children(func() {
				ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
					ui.Textf(c, "%s登录", a.login.qrType).FontSize(15).FontWeight(600).
						TextColor(col.text).Grow(1)
					iconButton(c, col, iconX, "关闭", a.closeQR)
				})

				ui.Row(c).FillWidth().Justify(ui.Center).Children(func() {
					qr := ui.Box(c).Size(208, 208).Radius(10).Padding(10).
						Background(ui.RGB(255, 255, 255)).Center().
						Border(1, col.border)
					qr.Children(func() {
						switch {
						case a.login.qrBitmap != nil:
							ui.Image(c, a.login.qrBitmap).Size(188, 188)
						case a.login.qrFailed:
							ui.Icon(c, iconAlert).Size(22, 22).TextColor(col.red)
						default:
							ui.Spinner(c).Size(22, 22)
						}
					})
				})

				statusColor := col.muted
				if a.login.qrFailed {
					statusColor = col.red
				}
				ui.Text(c, a.login.qrStatus).FontSize(12).TextColor(statusColor)
				ui.Row(c).Gap(6).Justify(ui.End).Children(func() {
					secondaryButton(c, col, nil, "关闭", a.closeQR)
					if a.login.qrBitmap != nil || a.login.qrFailed {
						primaryButton(c, col, iconRefresh, "重新获取", func() {
							a.openQR(a.login.qrType)
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
			a.downloadDir = settings.DefaultDownloadDir()
		}
		cleanPartFiles(a.downloadDir)
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
		applyAppIcon(a.win)
		a.win.OnClosed(func() {
			a.closed.Store(true)
			a.stopSongPress()
			if a.queue.cancel != nil {
				a.queue.cancel()
			}
			if a.login.qrCancel != nil {
				a.login.qrCancel()
			}
			// Let queued writes finish before the store closes; a.closed
			// already stops their UI callbacks.
			a.drainDisk(2 * time.Second)
			if a.history.store != nil {
				_ = a.history.store.Close()
			}
		})
		a.loadUserProfile()
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			if err := a.api.EnsureCredential(ctx); err != nil {
				log.Printf("refresh credential: %v", err)
			}
			// A restored cache only shows the last session's playlists; pull
			// the list and its songs once so the page opens current.
			a.update(func() {
				if a.api.HasCredential() {
					a.refreshPlaylists()
				}
			})
		}()
		go a.initHistory()
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
