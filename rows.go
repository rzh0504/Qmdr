package main

import (
	"time"

	"github.com/egoist/mygo/ui"
)

func statusIcon(c *ui.Context, col colors, status QueueStatus) ui.Element {
	icon := iconClock
	fg := col.muted
	switch status {
	case StatusDownloading:
		icon, fg = iconDownload, col.accent
	case StatusDone:
		icon, fg = iconCheck, col.green
	case StatusFailed:
		icon, fg = iconAlert, col.red
	case StatusCancelled:
		icon, fg = iconX, col.gold
	case StatusWaiting:
		icon, fg = iconClock, col.muted
	}
	return ui.Icon(c, icon).Size(14, 14).TextColor(fg)
}

func playlistIcon(c *ui.Context, col colors, playlist Playlist, size float32) ui.Element {
	icon := iconPlaylist
	tint := col.faint
	if playlist.DirID == 201 {
		icon, tint = iconHeartFilled, col.red
	}
	box := ui.Box(c).Size(size, size)
	box.Children(func() {
		ui.Row(c).Fill().Center().Children(func() {
			ui.Icon(c, icon).Size(size*0.46, size*0.46).TextColor(tint)
		})
	})
	return box
}

type songRowOptions struct {
	index       int
	selected    bool
	selectMode  bool
	onSelect    func()
	onLongPress func()
	onDownload  func()
}

const songLongPressDelay = 500 * time.Millisecond

func (a *app) trackSongPress(row ui.Element, song Song, onLongPress func()) {
	key := song.Mid
	if key == "" {
		return
	}
	if row.Pressed() {
		if a.playlists.pressKey == key {
			return
		}
		a.stopSongPress()
		a.playlists.pressKey = key
		a.playlists.pressSeq++
		seq := a.playlists.pressSeq
		a.playlists.suppressClick = false
		a.playlists.pressTimer = time.AfterFunc(songLongPressDelay, func() {
			a.update(func() {
				if a.playlists.pressKey != key || a.playlists.pressSeq != seq {
					return
				}
				a.playlists.pressTimer = nil
				a.playlists.suppressClick = true
				onLongPress()
			})
		})
		return
	}
	if a.playlists.pressKey == key {
		a.stopSongPress()
	}
}

func (a *app) stopSongPress() {
	if a.playlists.pressTimer != nil {
		a.playlists.pressTimer.Stop()
		a.playlists.pressTimer = nil
	}
	a.playlists.pressKey = ""
}

func songSelectionMark(c *ui.Context, col colors, selected bool) ui.Element {
	if selected {
		box := ui.Box(c).Size(16, 16).Radius(5).Background(col.accent).Center()
		box.Children(func() {
			ui.Icon(c, iconCheck).Size(10, 10).TextColor(col.accentText)
		})
		return box
	}
	return ui.Box(c).Size(16, 16).Radius(5).Border(1, col.fieldStroke)
}

func (a *app) songRow(c *ui.Context, col colors, song Song, opts songRowOptions) ui.Element {
	row := ui.Row(c).Gap(10).Padding(7, 10).Height(52).Radius(8).AlignItems(ui.Center)
	switch {
	case opts.selectMode && opts.selected:
		fill := col.accent.Alpha(0.06)
		if col.dark {
			fill = col.accent.Alpha(0.12)
		}
		row.Background(fill).Cursor(ui.CursorPointer)
	case opts.selectMode:
		row.Cursor(ui.CursorPointer)
		if row.Hovered() {
			row.Background(col.rowSelected)
			row = chosen(row, col)
		}
	case row.Hovered():
		row.Background(col.rowSelected)
		row = chosen(row, col)
	}
	if opts.onLongPress != nil {
		a.trackSongPress(row, song, opts.onLongPress)
	}
	if opts.selectMode && opts.onSelect != nil && row.Clicked() {
		if a.playlists.suppressClick && song.Mid == a.playlists.pressKey {
			a.playlists.suppressClick = false
			a.playlists.pressKey = ""
		} else {
			a.playlists.suppressClick = false
			a.playlists.pressKey = ""
			opts.onSelect()
		}
	}
	row.Children(func() {
		if opts.selectMode {
			ui.Row(c).Width(24).Center().Children(func() {
				songSelectionMark(c, col, opts.selected)
			})
		} else {
			ui.Textf(c, "%02d", opts.index+1).Width(24).FontSize(11).
				TextColor(col.faint).SingleLine()
		}
		a.cover(c, col, song.AlbumMid, 34)
		ui.Column(c).Gap(2).Grow(1).MinWidth(0).Children(func() {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Text(c, song.Title).FontSize(13).FontWeight(400).TextColor(col.text).SingleLine()
				if song.VIP {
					ui.Text(c, "VIP").FontSize(9).FontWeight(600).TextColor(col.gold).SingleLine()
				}
				if song.Quality != "" {
					ui.Text(c, song.Quality).FontSize(10).TextColor(col.muted).SingleLine()
				}
			})
			ui.Textf(c, "%s · %s", song.Artist, song.Album).FontSize(12).
				TextColor(col.muted).SingleLine()
		})
		ui.Text(c, song.Duration).Width(38).FontSize(11).TextColor(col.faint).SingleLine()
		if !opts.selectMode && opts.onDownload != nil {
			iconButton(c, col, iconDownload, "下载 "+song.Title, opts.onDownload)
		}
	})
	return row
}
