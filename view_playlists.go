package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/rzh0504/qmdr/internal/qqmusic"
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
			if a.loadingPlaylists || a.refreshingPlaylists {
				ui.Row(c).Gap(4).AlignItems(ui.Center).Padding(2, 6).Children(func() {
					ui.Spinner(c).Size(12, 12)
					ui.Text(c, "刷新").FontSize(12).TextColor(col.faint)
				})
			} else {
				chipButton(c, col, iconRefresh, "刷新", a.refreshPlaylists)
			}
			if a.viewingOtherPlaylists() {
				chipButton(c, col, iconChevronLeft, "返回我的", a.loadOwnPlaylists)
			} else {
				chipButton(c, col, iconUser, "按 ID 查看", a.openMusicIDModal)
			}
		})

		if len(a.playlists) == 0 {
			sheet(c, col, func() {
				title := "还没有歌单"
				hint := "点击刷新按钮读取歌单。"
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
					playlistIcon(c, col, playlist, 30)
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
		selectedCount := a.selectedSongCount()

		ui.Row(c).Height(44).FillWidth().AlignItems(ui.Center).Children(func() {
			playlistIcon(c, col, playlist, 44)
			ui.Column(c).Gap(2).Grow(1).MinWidth(0).Children(func() {
				ui.Text(c, playlist.Name).FontSize(16).FontWeight(600).
					TextColor(col.text).SingleLine()
				if a.selectingSongs {
					ui.Textf(c, "已选 %d / %d 首", selectedCount, len(a.playlistSongs)).
						FontSize(12).TextColor(col.accent)
				} else {
					ui.Textf(c, "%d 首", playlist.Count).FontSize(12).TextColor(col.muted)
				}
			})
			if a.loadingSongs {
				ui.Spinner(c).Size(15, 15)
			} else if len(a.playlistSongs) > 0 {
				ui.Row(c).Gap(6).Shrink(0).AlignItems(ui.Center).Children(func() {
					if a.selectingSongs {
						allLabel := "全选"
						if a.allSongsSelected() {
							allLabel = "取消全选"
						}
						ghostButton(c, col, nil, allLabel, a.toggleAllSongs)
						if selectedCount > 0 {
							primaryButton(c, col, iconDownload, "下载所选", a.downloadSelectedSongs)
						} else {
							primaryButtonDisabled(c, col, iconDownload, "下载所选")
						}
						secondaryButton(c, col, nil, "退出", a.endSongSelection)
					} else {
						primaryButton(c, col, iconDownload, "下载歌单", a.downloadCurrentPlaylist)
					}
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
			if a.playlistNotice != "" {
				ui.Row(c).FillWidth().Gap(7).AlignItems(ui.Center).Padding(0, 4).Children(func() {
					ui.Icon(c, iconAlert).Size(14, 14).TextColor(col.gold)
					ui.Text(c, a.playlistNotice).FontSize(12).TextColor(col.muted).
						Grow(1).MaxLines(2)
				})
			}
			listPanel(c, col, "songs-"+a.playlistOwner+"-"+strconv.FormatInt(playlist.TID, 10), func() {
				for i, song := range a.playlistSongs {
					i, song := i, song
					selected := a.selectingSongs && a.selectedSongMids[song.Mid]
					var longPress func()
					if !a.selectingSongs {
						longPress = func() { a.beginSongSelectionWith(song) }
					}
					row := a.songRow(c, col, song, songRowOptions{
						index:       i,
						selected:    selected,
						selectMode:  a.selectingSongs,
						onSelect:    func() { a.toggleSongSelection(song) },
						onLongPress: longPress,
					})
					row.ContextMenu(func(m *ui.Menu) {
						a.songContextMenu(m, song)
					})
				}
			})
		}
	})
}

func (a *app) songContextMenu(m *ui.Menu, song Song) {
	if a.selectingSongs {
		if m.Item("下载选中歌曲").Disabled(a.selectedSongCount() == 0).Chosen() {
			a.downloadSelectedSongs()
		}
		if m.Item("选中项加入待下载队列").Chosen() {
			a.addSelectedToPending()
		}
		m.Separator()
		if m.Item("全选").Chosen() {
			a.selectAllSongs()
		}
		if m.Item("退出多选").Chosen() {
			a.endSongSelection()
		}
		return
	}
	if m.Item("下载这首歌").Chosen() {
		a.downloadSongNow(song)
	}
	if m.Item("加入待下载队列").Chosen() {
		a.addToPending(song)
	}
}

func (a *app) beginSongSelectionWith(song Song) {
	if strings.TrimSpace(song.Mid) == "" {
		a.setToast("歌曲缺少 MID，无法下载")
		return
	}
	if !a.selectingSongs {
		a.selectingSongs = true
		a.selectedSongMids = make(map[string]bool)
	}
	a.selectedSongMids[song.Mid] = true
	if a.selectedSongCount() == 0 {
		a.resetSongSelection()
	}
	a.refresh()
}

func (a *app) endSongSelection() {
	a.resetSongSelection()
	a.refresh()
}

func (a *app) resetSongSelection() {
	a.stopSongPress()
	a.selectingSongs = false
	a.selectedSongMids = make(map[string]bool)
	a.suppressSongClick = false
}

func (a *app) toggleSongSelection(song Song) {
	if !a.selectingSongs {
		return
	}
	if strings.TrimSpace(song.Mid) == "" {
		a.setToast("歌曲缺少 MID，无法下载")
		return
	}
	if a.selectedSongMids[song.Mid] {
		delete(a.selectedSongMids, song.Mid)
		if a.selectedSongCount() == 0 {
			a.resetSongSelection()
		}
	} else {
		a.selectedSongMids[song.Mid] = true
	}
	a.refresh()
}

func (a *app) toggleAllSongs() {
	if a.allSongsSelected() {
		a.resetSongSelection()
	} else {
		a.selectAllSongs()
		return
	}
	a.refresh()
}

func (a *app) selectAllSongs() {
	next := make(map[string]bool, len(a.playlistSongs))
	for _, song := range a.playlistSongs {
		if strings.TrimSpace(song.Mid) != "" {
			next[song.Mid] = true
		}
	}
	a.selectedSongMids = next
	if len(next) == 0 {
		a.resetSongSelection()
	}
	a.refresh()
}

func (a *app) allSongsSelected() bool {
	if len(a.playlistSongs) == 0 {
		return false
	}
	for _, song := range a.playlistSongs {
		if !a.selectedSongMids[song.Mid] {
			return false
		}
	}
	return true
}

func (a *app) selectedSongCount() int {
	count := 0
	for _, song := range a.playlistSongs {
		if a.selectedSongMids[song.Mid] {
			count++
		}
	}
	return count
}

func (a *app) selectedSongsInOrder() []Song {
	songs := make([]Song, 0, len(a.selectedSongMids))
	for _, song := range a.playlistSongs {
		if a.selectedSongMids[song.Mid] {
			songs = append(songs, song)
		}
	}
	return songs
}

func (a *app) addSelectedToPending() {
	songs := a.selectedSongsInOrder()
	if len(songs) == 0 {
		a.setToast("请先选择歌曲")
		return
	}
	a.addToPending(songs...)
}

func (a *app) downloadSongNow(song Song) {
	if a.enqueueSongs([]Song{song}) {
		a.resetSongSelection()
	}
}

func (a *app) downloadSelectedSongs() {
	songs := a.selectedSongsInOrder()
	if len(songs) == 0 {
		a.setToast("请先选择要下载的歌曲")
		return
	}
	if a.enqueueSongs(songs) {
		a.resetSongSelection()
	}
}

func (a *app) downloadCurrentPlaylist() {
	if a.enqueueSongs(a.playlistSongs) {
		a.resetSongSelection()
	}
}

func (a *app) pruneSongSelection() {
	if !a.selectingSongs {
		return
	}
	if len(a.playlistSongs) == 0 {
		a.resetSongSelection()
		return
	}
	next := make(map[string]bool, len(a.selectedSongMids))
	for _, song := range a.playlistSongs {
		if a.selectedSongMids[song.Mid] {
			next[song.Mid] = true
		}
	}
	a.selectedSongMids = next
	if a.selectedSongCount() == 0 {
		a.resetSongSelection()
	}
}

func (a *app) refreshPlaylists() {
	owner := strings.TrimSpace(a.playlistOwner)
	if owner == "" || owner == "未登录" {
		owner = a.api.UserID()
	}
	if len(a.playlists) == 0 {
		a.loadPlaylistsFor(owner)
		return
	}
	a.refreshPlaylistsQuiet(owner)
}

func (a *app) refreshPlaylistsQuiet(owner string) {
	if a.loadingPlaylists || a.refreshingPlaylists || a.loadingSongs {
		return
	}
	if !a.api.HasCredential() {
		a.setToast("请先登录后读取歌单")
		return
	}
	owner = strings.TrimSpace(owner)
	if owner == "" || owner == "未登录" {
		owner = a.api.UserID()
	}
	if owner == "" {
		return
	}
	a.playlistReq++
	requestID := a.playlistReq
	a.playlistOwner = owner
	a.refreshingPlaylists = true
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
			a.refreshingPlaylists = false
			if err != nil {
				a.toast = "刷新失败：" + friendlyError(err)
				return
			}
			a.applyPlaylistsQuiet(mapped)
			if !a.loadingSongs {
				a.refreshSelectedSongsQuiet()
			}
		})
	}()
}

func (a *app) applyPlaylistsQuiet(playlists []Playlist) {
	if playlistsEqual(a.playlists, playlists) {
		return
	}
	var selectedTID, selectedDirID int64
	if a.selectedPlaylist >= 0 && a.selectedPlaylist < len(a.playlists) {
		selected := a.playlists[a.selectedPlaylist]
		selectedTID, selectedDirID = selected.TID, selected.DirID
	}
	a.playlists = playlists
	a.selectedPlaylist = -1
	if selectedTID != 0 || selectedDirID != 0 {
		for i, playlist := range playlists {
			if playlist.TID == selectedTID && playlist.DirID == selectedDirID {
				a.selectedPlaylist = i
				break
			}
		}
	}
	if a.selectedPlaylist < 0 {
		a.playlistSongs = nil
		a.playlistSongsID++
		a.loadingSongs = false
		a.playlistErr = ""
		a.playlistNotice = ""
		a.resetSongSelection()
	}
	if !a.viewingOtherPlaylists() {
		a.savePlaylistCache()
	}
}

func (a *app) refreshSelectedSongsQuiet() {
	if a.loadingSongs {
		return
	}
	index := a.selectedPlaylist
	if index < 0 || index >= len(a.playlists) {
		return
	}
	playlist := a.playlists[index]
	if playlist.DirID == 201 && a.viewingOtherPlaylists() {
		return
	}
	owner := a.playlistOwner
	tid, dirID := playlist.TID, playlist.DirID
	a.playlistSongsID++
	requestID := a.playlistSongsID

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		page, err := a.api.PlaylistSongs(ctx, tid, dirID)
		if err != nil {
			return
		}
		mapped := make([]Song, 0, len(page.Songs))
		for _, song := range page.Songs {
			mapped = append(mapped, songFromAPI(song))
		}
		a.update(func() {
			if requestID != a.playlistSongsID || a.playlistOwner != owner {
				return
			}
			if a.selectedPlaylist < 0 || a.selectedPlaylist >= len(a.playlists) {
				return
			}
			current := a.playlists[a.selectedPlaylist]
			if current.TID != tid || current.DirID != dirID {
				return
			}
			if a.playlistErr != "" {
				a.playlistErr = ""
			}
			a.playlistNotice = truncationNotice(page)
			if songsEqual(a.playlistSongs, mapped) {
				return
			}
			if songsEqual(a.playlistSongs, mapped) {
				return
			}
			a.playlistSongs = mapped
			a.pruneSongSelection()
			if !a.viewingOtherPlaylists() {
				a.savePlaylistCache()
			}
		})
	}()
}

func truncationNotice(page qqmusic.PlaylistPage) string {
	if !page.Truncated {
		return ""
	}
	if page.Total > 0 {
		return fmt.Sprintf("歌单共 %d 首，已加载前 %d 首", page.Total, len(page.Songs))
	}
	return fmt.Sprintf("歌单较大，已加载前 %d 首", len(page.Songs))
}

func playlistsEqual(a, b []Playlist) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func songsEqual(a, b []Song) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (a *app) loadOwnPlaylists() {
	a.musicIDOpen = false
	a.musicIDErr = ""
	owner := a.api.UserID()
	a.playlistReq++
	a.playlistSongsID++
	a.loadingPlaylists = false
	a.refreshingPlaylists = false
	a.loadingSongs = false
	a.playlistOwner = owner
	a.playlistErr = ""
	a.playlistNotice = ""
	a.playlists = nil
	a.selectedPlaylist = -1
	a.playlistSongs = nil
	a.resetSongSelection()
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
	a.refreshingPlaylists = false
	a.loadingSongs = false
	a.playlistErr = ""
	a.playlistNotice = ""
	a.playlists = nil
	a.selectedPlaylist = -1
	a.playlistSongs = nil
	a.resetSongSelection()
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
	a.resetSongSelection()
	playlist := a.playlists[index]
	a.selectedPlaylist = index
	if playlist.DirID == 201 && a.viewingOtherPlaylists() {
		a.playlistSongsID++
		a.loadingSongs = false
		a.playlistErr = "其他用户的「我喜欢」不公开"
		a.playlistNotice = ""
		a.playlistSongs = nil
		a.refresh()
		return
	}
	a.loadingSongs = true
	a.playlistErr = ""
	a.playlistNotice = ""
	a.playlistSongs = nil
	a.playlistSongsID++
	requestID := a.playlistSongsID
	a.refresh()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		page, err := a.api.PlaylistSongs(ctx, playlist.TID, playlist.DirID)
		mapped := make([]Song, 0, len(page.Songs))
		for _, song := range page.Songs {
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
			a.playlistNotice = truncationNotice(page)
			a.savePlaylistCache()
		})
	}()
}
