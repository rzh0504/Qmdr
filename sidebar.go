package main

import (
	"github.com/egoist/mygo/ui"
)

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
			if count := active + len(a.queue.pending); count > 0 {
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
			name := a.login.userID
			if a.login.nick != "" {
				name = a.login.nick
			}
			ui.Text(c, name).FontSize(12).FontWeight(400).TextColor(col.text).SingleLine().Ellipsis("…")
			if a.login.loggedIn {
				ui.Text(c, "已登录").FontSize(11).TextColor(col.muted).SingleLine()
			}
		})
		ui.Icon(c, iconChevronRight).Size(13, 13).TextColor(col.faint)
	})
}

func (a *app) logoMark(c *ui.Context, col colors, size float32) {
	if a.brandIcon != nil {
		ui.Image(c, a.brandIcon).Size(size, size)
		return
	}
	ui.Box(c).Size(size, size).Radius(size * 0.26).Background(col.accent).
		Children(func() {
			ui.Row(c).Fill().Center().Children(func() {
				ui.Icon(c, iconMusic).Size(size*0.5, size*0.5).TextColor(col.accentText)
			})
		})
}

func (a *app) avatar(c *ui.Context, col colors, size float32) {
	box := ui.Box(c).Size(size, size).Radius(size / 2).Background(col.chip).Clip()
	box.Children(func() {
		if a.login.avatar != nil {
			ui.Image(c, a.login.avatar).Size(size, size).Fit(ui.Cover)
			return
		}
		ui.Row(c).Fill().Center().Children(func() {
			ui.Text(c, "Q").FontSize(float32(int(size*0.4 + 0.5))).FontWeight(600).TextColor(col.text)
		})
	})
}
