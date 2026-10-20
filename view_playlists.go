package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

func (a *app) playlistsView(c *ui.Context, col colors) {
	pageHeader(c, col, "歌单下载", func() {
		secondaryButton(c, col, iconRefresh, "刷新歌单", func() {
			a.loadPlaylists()
		})
	})

	a.playlistLoader(c, col)

	ui.Row(c).Gap(16).AlignItems(ui.Start).Children(func() {
		a.playlistSidebar(c, col)
		a.playlistPreview(c, col)
	})
}

func (a *app) playlistLoader(c *ui.Context, col colors) {
	ui.Row(c).Gap(8).AlignItems(ui.Center).Padding(4, 0).Children(func() {
		ui.Icon(c, iconLibrary).FontSize(14).TextColor(col.muted)
		ui.Text(c, "读取歌单").FontSize(12).Bold().TextColor(col.text)
		ui.Spacer(c)
		ui.TextInput(c, &a.musicID).
			Placeholder("QQ 号 / 微信 ID").
			Label("QQ 号或微信 ID").
			Width(200).Height(30).
			Background(col.cardAlt).Border(1, col.border).Radius(6)
		if a.loadingPlaylists {
			btn := ui.PrimaryButton(c, "").Label("正在加载").Disabled(true).
				Padding(0, 10).Height(30).Radius(6)
			btn.Children(func() {
				ui.Spinner(c).Size(12, 12)
				ui.Text(c, "正在加载…").FontSize(11).Bold()
			})
		} else {
			primaryButton(c, col, iconRefresh, "加载歌单", func() {
				a.loadPlaylists()
			})
		}
	})
	divider(c, col)
}

func (a *app) playlistSidebar(c *ui.Context, col colors) {
	box := ui.Column(c).Width(280).Padding(12).Gap(6).Radius(8).
		Background(col.card).Border(1, col.border)
	box.Children(func() {
		ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
			ui.Text(c, "我的歌单").FontSize(13).Bold().TextColor(col.text)
			ui.Box(c).Width(6)
			pill(c, fmt.Sprintf("%d", len(a.playlists)), col.muted, col.cardAlt)
			ui.Spacer(c)
			if a.loadingPlaylists {
				ui.Spinner(c).Size(14, 14)
			}
		})

		if len(a.playlists) == 0 {
			emptyState(c, col, iconLibrary, "还没有歌单", "输入账号后点击“加载歌单”。")
			return
		}

		for i, playlist := range a.playlists {
			i, playlist := i, playlist
			selected := i == a.selectedPlaylist
			btn := ui.ButtonBase(c).Gap(8).Padding(5).Radius(5).
				AlignItems(ui.Center).Label(playlist.Name)
			if selected {
				btn = btn.Background(col.cardHover)
			} else if btn.Hovered() {
				btn = btn.Background(col.cardHover)
			}
			if btn.Clicked() {
				a.selectedPlaylist = i
				a.playlistSongs = playlist.Songs
				a.refresh()
			}
			btn.Children(func() {
				cover(c, col, i+1, 32, iconLibrary)
				ui.Column(c).Gap(1).Grow(1).Children(func() {
					ui.Text(c, playlist.Name).FontSize(11).
						TextColor(col.text).SingleLine()
					ui.Textf(c, "%d 首", playlist.Count).FontSize(9).
						TextColor(col.muted).SingleLine()
				})
				if selected {
					ui.Icon(c, iconChevronRight).FontSize(13).TextColor(col.accent)
				}
			})
		}
	})
}

func (a *app) playlistPreview(c *ui.Context, col colors) {
	box := ui.Column(c).Grow(1).Padding(14).Gap(10).Radius(8).
		Background(col.card).Border(1, col.border)
	box.Children(func() {
		if len(a.playlists) == 0 || a.selectedPlaylist < 0 || a.selectedPlaylist >= len(a.playlists) {
			emptyState(c, col, iconPlay, "选择一个歌单", "左侧选择歌单后，这里会显示歌曲预览。")
			return
		}
		playlist := a.playlists[a.selectedPlaylist]

		ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
			cover(c, col, a.selectedPlaylist+1, 40, iconLibrary)
			ui.Column(c).Gap(1).Grow(1).Children(func() {
				ui.Text(c, playlist.Name).FontSize(15).Bold().TextColor(col.text).SingleLine()
				ui.Textf(c, "已加载 %d 首（演示数据）", len(playlist.Songs)).
					FontSize(10).TextColor(col.muted)
			})
			primaryButton(c, col, iconDownload, "下载歌单", func() {
				a.enqueueSongs(playlist.Songs)
			})
		})

		divider(c, col)

		ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
			ui.Text(c, "歌曲").FontSize(10).TextColor(col.muted)
			ui.Spacer(c)
			ui.Textf(c, "%d 首", len(playlist.Songs)).FontSize(10).TextColor(col.muted)
		})

		if len(playlist.Songs) == 0 {
			emptyState(c, col, iconMusic, "歌单为空", "这个歌单暂时没有可下载的歌曲。")
			return
		}

		ui.Column(c).Gap(0).Children(func() {
			for i, song := range playlist.Songs {
				songRow(c, col, song, i, true, nil)
				if i < len(playlist.Songs)-1 {
					divider(c, col)
				}
			}
		})
	})
}

func (a *app) loadPlaylists() {
	if strings.TrimSpace(a.musicID) == "" {
		a.setToast("请输入 QQ 号或微信 ID")
		return
	}
	if a.loadingPlaylists {
		return
	}
	a.loadingPlaylists = true
	a.refresh()

	go func() {
		time.Sleep(850 * time.Millisecond)
		a.update(func() {
			a.loadingPlaylists = false
			a.playlists = samplePlaylists()
			a.selectedPlaylist = 0
			a.playlistSongs = a.playlists[0].Songs
			a.appendLog("已加载 5 个歌单（演示数据）", "success")
			a.toast = "已加载 5 个歌单（演示数据）"
		})
	}()
}
