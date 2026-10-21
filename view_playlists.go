package main

import (
	"context"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

func (a *app) playlistsView(c *ui.Context, col colors) {
	a.playlistLoader(c, col)

	ui.Row(c).Gap(16).AlignItems(ui.Start).Children(func() {
		a.playlistSidebar(c, col)
		a.playlistPreview(c, col)
	})
}

func (a *app) playlistLoader(c *ui.Context, col colors) {
	ui.Row(c).Gap(8).AlignItems(ui.Center).Wrap().Children(func() {
		ui.Text(c, "QQ 号 / 微信 ID").FontSize(12).TextColor(col.muted)
		ui.TextInput(c, &a.musicID).
			Placeholder("QQ 号 / 微信 ID").
			Label("QQ 号或微信 ID").
			Width(190).Height(34).FontSize(13).
			Background(col.control).InsetShadow(0, 0, 0, 1, col.fieldStroke).
			Radius(8)
		if a.loadingPlaylists {
			secondaryButton(c, col, nil, "正在加载…", nil).Disabled(true)
		} else {
			secondaryButton(c, col, iconRefresh, "加载歌单", func() {
				a.loadPlaylists()
			})
		}
		ui.Spacer(c)
		if len(a.playlists) > 0 {
			ui.Textf(c, "%d 个歌单", len(a.playlists)).FontSize(11.5).TextColor(col.faint)
		}
	})
}

func (a *app) playlistSidebar(c *ui.Context, col colors) {
	ui.Column(c).Width(264).Gap(8).Children(func() {
		ui.Row(c).FillWidth().AlignItems(ui.Center).Padding(0, 4).Children(func() {
			ui.Text(c, "我的歌单").FontSize(12).FontWeight(600).TextColor(col.muted).Grow(1)
			if len(a.playlists) > 0 {
				ui.Textf(c, "%d", len(a.playlists)).FontSize(11.5).TextColor(col.faint)
			}
		})

		if len(a.playlists) == 0 {
			sheet(c, col, func() {
				title := "还没有歌单"
				hint := "登录后点击“加载歌单”。"
				if !a.loggedIn {
					title = "未登录"
					hint = "请先在设置页登录。"
				}
				emptyState(c, col, iconLibrary, title, hint)
			})
			if a.playlistErr != "" {
				ui.Text(c, a.playlistErr).FontSize(11).TextColor(col.red).MaxLines(2).Padding(0, 4)
			}
			return
		}

		sheet(c, col, func() {
			for i, playlist := range a.playlists {
				i, playlist := i, playlist
				selected := i == a.selectedPlaylist
				btn := ui.ButtonBase(c).Gap(9).Padding(6, 8).Height(46).Radius(8).
					AlignItems(ui.Center).Cursor(ui.CursorPointer).Label(playlist.Name)
				if selected {
					btn.Background(col.rowSelected)
					chosen(btn, col)
				} else if btn.Hovered() {
					btn.Background(col.sheetHover)
				}
				if btn.Clicked() {
					a.loadPlaylistSongs(i)
				}
				btn.Children(func() {
					playlistCover(c, col, i+1, 30)
					ui.Column(c).Gap(2).Grow(1).MinWidth(0).Children(func() {
						ui.Text(c, playlist.Name).FontSize(12.5).FontWeight(500).
							TextColor(col.text).SingleLine()
						ui.Textf(c, "%d 首", playlist.Count).FontSize(11).
							TextColor(col.muted).SingleLine()
					})
				})
			}
		})
	})
}

func (a *app) playlistPreview(c *ui.Context, col colors) {
	ui.Column(c).Grow(1).MinWidth(0).Gap(12).Children(func() {
		if len(a.playlists) == 0 || a.selectedPlaylist < 0 || a.selectedPlaylist >= len(a.playlists) {
			sheet(c, col, func() {
				emptyState(c, col, iconPlay, "选择一个歌单", "左侧选择歌单后，这里会显示歌曲。")
			})
			return
		}
		playlist := a.playlists[a.selectedPlaylist]

		ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
			playlistCover(c, col, a.selectedPlaylist+1, 44)
			ui.Column(c).Gap(2).Grow(1).MinWidth(0).Children(func() {
				ui.Text(c, playlist.Name).FontSize(16).FontWeight(650).
					TextColor(col.text).SingleLine()
				ui.Textf(c, "%d 首", playlist.Count).FontSize(11.5).TextColor(col.muted)
			})
			if a.loadingSongs {
				ui.Spinner(c).Size(15, 15)
			} else if len(a.playlistSongs) > 0 {
				primaryButton(c, col, iconDownload, "下载歌单", func() {
					a.enqueueSongs(a.playlistSongs)
				})
			}
		})

		switch {
		case a.loadingSongs:
			softCard(c, col, func() {
				ui.Row(c).Gap(9).AlignItems(ui.Center).Children(func() {
					ui.Spinner(c).Size(15, 15)
					ui.Text(c, "正在加载歌曲…").FontSize(12).TextColor(col.muted)
				})
			})
		case a.playlistErr != "":
			softCard(c, col, func() {
				ui.Row(c).Gap(9).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, iconAlert).Size(15, 15).TextColor(col.red)
					ui.Text(c, a.playlistErr).FontSize(12).TextColor(col.red).Grow(1).MaxLines(2)
					secondaryButton(c, col, iconRefresh, "重试", func() {
						a.loadPlaylistSongs(a.selectedPlaylist)
					})
				})
			})
		case len(a.playlistSongs) == 0:
			sheet(c, col, func() {
				emptyState(c, col, iconMusic, "歌单为空", "这个歌单暂时没有可下载的歌曲。")
			})
		default:
			sheet(c, col, func() {
				for i, song := range a.playlistSongs {
					a.songRow(c, col, song, i, true, nil)
				}
			})
		}
	})
}

func (a *app) loadPlaylists() {
	if a.loadingPlaylists {
		return
	}
	if !a.api.HasCredential() {
		a.setToast("请先登录后读取歌单")
		return
	}
	uin := strings.TrimSpace(a.musicID)
	if uin == "" || uin == "未登录" {
		uin = a.api.UserID()
	}
	if strings.TrimSpace(a.musicID) == "" || a.musicID == "未登录" {
		a.musicID = uin
	}
	a.loadingPlaylists = true
	a.playlistErr = ""
	a.refresh()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		items, err := a.api.UserPlaylists(ctx, uin)
		mapped := make([]Playlist, 0, len(items))
		for _, item := range items {
			mapped = append(mapped, Playlist{
				DirID: item.DirID,
				TID:   item.TID,
				Name:  item.Name,
				Count: item.SongCount,
			})
		}
		a.update(func() {
			a.loadingPlaylists = false
			if err != nil {
				a.playlistErr = friendlyError(err)
				a.toast = "读取歌单失败"
				return
			}
			a.playlists = mapped
			a.selectedPlaylist = -1
			a.playlistSongs = nil
			a.savePlaylistCache()
			if len(mapped) == 0 {
				a.toast = "没有找到歌单"
				return
			}
			a.loadPlaylistSongs(0)
		})
	}()
}

func (a *app) loadPlaylistSongs(index int) {
	if index < 0 || index >= len(a.playlists) {
		return
	}
	playlist := a.playlists[index]
	a.selectedPlaylist = index
	a.loadingSongs = true
	a.playlistErr = ""
	a.playlistSongs = nil
	a.playlistSongsID++
	requestID := a.playlistSongsID
	a.refresh()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		songs, err := a.api.PlaylistSongs(ctx, playlist.TID, playlist.DirID)
		mapped := make([]Song, 0, len(songs))
		for _, song := range songs {
			mapped = append(mapped, songFromAPI(song))
		}
		a.update(func() {
			if a.playlistSongsID != requestID {
				return
			}
			a.loadingSongs = false
			if err != nil {
				a.playlistErr = friendlyError(err)
				return
			}
			a.playlistSongs = mapped
			a.savePlaylistCache()
		})
	}()
}
