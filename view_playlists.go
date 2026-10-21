package main

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

func (a *app) playlistsView(c *ui.Context, col colors) {
	ui.Row(c).Fill().AlignItems(ui.Stretch).Gap(16).Children(func() {
		a.playlistSidebar(c, col)
		a.playlistPreview(c, col)
	})
}

func (a *app) viewingOtherPlaylists() bool {
	owner := strings.TrimSpace(a.playlistOwner)
	return owner != "" && owner != a.api.UserID()
}

func listPanel(c *ui.Context, col colors, key string, fn func()) ui.Element {
	panel := ui.Column(c).Grow(1).MinHeight(0).FillWidth().Padding(4)
	smoothFlat(panel, col, 12, col.chip)
	panel.Children(func() {
		ui.Scroll(c).Key(key).Grow(1).MinHeight(0).FillWidth().Gap(1).Children(fn)
	})
	return panel
}

func (a *app) playlistSidebar(c *ui.Context, col colors) {
	ui.Column(c).Width(264).FillHeight().MinHeight(0).Gap(12).Children(func() {
		title := "我的歌单"
		if a.viewingOtherPlaylists() {
			title = "其他用户的歌单"
		}
		ui.Row(c).Height(44).FillWidth().AlignItems(ui.Center).Padding(0, 4).Children(func() {
			ui.Column(c).Grow(1).MinWidth(0).Gap(1).Children(func() {
				ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
					ui.Text(c, title).FontSize(12).FontWeight(600).TextColor(col.muted)
					if len(a.playlists) > 0 {
						ui.Textf(c, "%d", len(a.playlists)).FontSize(12).TextColor(col.faint)
					}
				})
				if a.viewingOtherPlaylists() {
					ui.Text(c, a.playlistOwner).FontSize(11).TextColor(col.faint).
						SingleLine().Ellipsis("…")
				}
			})
			if a.viewingOtherPlaylists() {
				chipButton(c, col, iconChevronLeft, "返回我的", a.loadOwnPlaylists)
			} else {
				chipButton(c, col, iconUser, "按 ID 查看", a.openMusicIDModal)
			}
		})

		if len(a.playlists) == 0 {
			sheet(c, col, func() {
				title := "还没有歌单"
				hint := "点击右上角刷新读取歌单。"
				switch {
				case !a.loggedIn:
					title = "未登录"
					hint = "请先在设置页登录。"
				case a.viewingOtherPlaylists():
					title = "没有公开歌单"
					hint = "该用户没有可读取的公开歌单，或 ID 无效。"
				}
				emptyState(c, col, iconLibrary, title, hint)
			})
			if a.playlistErr != "" {
				ui.Text(c, a.playlistErr).FontSize(11).TextColor(col.red).MaxLines(2).Padding(0, 4)
			}
			return
		}

		listPanel(c, col, "playlists-"+a.playlistOwner, func() {
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
						ui.Text(c, playlist.Name).FontSize(13).FontWeight(400).
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
	ui.Column(c).Grow(1).MinWidth(0).FillHeight().MinHeight(0).Gap(12).Children(func() {
		if len(a.playlists) == 0 || a.selectedPlaylist < 0 || a.selectedPlaylist >= len(a.playlists) {
			sheet(c, col, func() {
				emptyState(c, col, iconPlay, "选择一个歌单", "左侧选择歌单后，这里会显示歌曲。")
			})
			return
		}
		playlist := a.playlists[a.selectedPlaylist]

		ui.Row(c).Height(44).FillWidth().AlignItems(ui.Center).Children(func() {
			playlistCover(c, col, a.selectedPlaylist+1, 44)
			ui.Column(c).Gap(2).Grow(1).MinWidth(0).Children(func() {
				ui.Text(c, playlist.Name).FontSize(16).FontWeight(600).
					TextColor(col.text).SingleLine()
				ui.Textf(c, "%d 首", playlist.Count).FontSize(12).TextColor(col.muted)
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
			listPanel(c, col, "songs-"+a.playlistOwner+"-"+strconv.Itoa(a.selectedPlaylist), func() {
				for i, song := range a.playlistSongs {
					a.songRow(c, col, song, i, true, nil)
				}
			})
		}
	})
}

func (a *app) loadPlaylists() {
	owner := strings.TrimSpace(a.playlistOwner)
	if owner == "" || owner == "未登录" {
		owner = a.api.UserID()
	}
	a.loadPlaylistsFor(owner)
}

func (a *app) loadOwnPlaylists() {
	a.musicIDOpen = false
	a.musicIDErr = ""
	owner := a.api.UserID()
	a.playlistReq++
	a.playlistSongsID++
	a.loadingPlaylists = false
	a.loadingSongs = false
	a.playlistOwner = owner
	a.playlistErr = ""
	a.playlists = nil
	a.selectedPlaylist = -1
	a.playlistSongs = nil
	a.restorePlaylistCache()
	if len(a.playlists) > 0 {
		a.refresh()
		return
	}
	a.loadPlaylistsFor(owner)
}

func (a *app) loadPlaylistsFor(owner string) {
	if !a.api.HasCredential() {
		a.setToast("请先登录后读取歌单")
		return
	}
	owner = strings.TrimSpace(owner)
	if owner == "" || owner == "未登录" {
		owner = a.api.UserID()
	}
	if owner == "" {
		a.setToast("缺少 music ID")
		return
	}
	a.playlistReq++
	requestID := a.playlistReq
	a.playlistSongsID++
	a.playlistOwner = owner
	a.loadingPlaylists = true
	a.loadingSongs = false
	a.playlistErr = ""
	a.playlists = nil
	a.selectedPlaylist = -1
	a.playlistSongs = nil
	a.musicIDOpen = false
	a.musicIDErr = ""
	a.refresh()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		items, err := a.api.UserPlaylists(ctx, owner)
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
			if requestID != a.playlistReq {
				return
			}
			a.loadingPlaylists = false
			if err != nil {
				a.playlistErr = friendlyError(err)
				a.toast = "读取歌单失败"
				return
			}
			a.playlists = mapped
			a.selectedPlaylist = -1
			a.playlistSongs = nil
			if !a.viewingOtherPlaylists() {
				a.savePlaylistCache()
			}
			if len(mapped) == 0 {
				if a.viewingOtherPlaylists() {
					a.toast = "该用户没有公开歌单"
				} else {
					a.toast = "没有找到歌单"
				}
				return
			}
			a.loadPlaylistSongs(0)
		})
	}()
}

func (a *app) openMusicIDModal() {
	a.musicIDInput = ""
	a.musicIDErr = ""
	a.musicIDOpen = true
	a.refresh()
}

func (a *app) closeMusicIDModal() {
	a.musicIDOpen = false
	a.musicIDErr = ""
	a.refresh()
}

func (a *app) submitMusicID() {
	id := strings.TrimSpace(a.musicIDInput)
	if !validMusicID(id) {
		a.musicIDErr = "请输入数字 music ID"
		a.refresh()
		return
	}
	if !a.api.HasCredential() {
		a.musicIDErr = "请先登录后再查看歌单"
		a.refresh()
		return
	}
	a.musicIDOpen = false
	a.musicIDErr = ""
	if id == a.api.UserID() {
		a.loadOwnPlaylists()
		return
	}
	a.loadPlaylistsFor(id)
}

func validMusicID(id string) bool {
	if id == "" {
		return false
	}
	for _, r := range id {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func (a *app) musicIDModal(c *ui.Context, col colors) {
	ui.DialogBase(c, &a.musicIDOpen, func(backdrop, panel ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, 0.4))
		panel.Width(376).Padding(18)
		smoothPanel(panel, col, 16, col.card, false)
		panel.Shadow(0, 16, 40, 0, col.floatShadow)
		panel.Children(func() {
			ui.Column(c).FillWidth().Gap(14).Children(func() {
				ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
					ui.Text(c, "查看用户歌单").FontSize(15).FontWeight(600).
						TextColor(col.text).Grow(1)
					iconButton(c, col, iconX, "关闭", a.closeMusicIDModal)
				})
				ui.Text(c, "输入 QQ 音乐的数字 music ID，查看该用户公开的歌单。其他用户的「我喜欢」不可读取。").
					FontSize(12).TextColor(col.muted).MaxWidth(340).LineHeight(1.45)
				a.textField(c, col, &a.musicIDInput, "QQ 音乐数字 ID", 340, 34, nil, a.submitMusicID)
				if a.musicIDErr != "" {
					ui.Text(c, a.musicIDErr).FontSize(12).TextColor(col.red)
				}
				ui.Row(c).Gap(6).Justify(ui.End).Children(func() {
					secondaryButton(c, col, nil, "取消", a.closeMusicIDModal)
					primaryButton(c, col, iconUser, "查看", a.submitMusicID)
				})
			})
		})
	})
}

func (a *app) loadPlaylistSongs(index int) {
	if index < 0 || index >= len(a.playlists) {
		return
	}
	playlist := a.playlists[index]
	a.selectedPlaylist = index
	if playlist.DirID == 201 && a.viewingOtherPlaylists() {
		a.playlistSongsID++
		a.loadingSongs = false
		a.playlistErr = "其他用户的「我喜欢」不公开"
		a.playlistSongs = nil
		a.refresh()
		return
	}
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
