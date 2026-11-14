package main

import (
	"runtime"

	"github.com/egoist/mygo/ui"
)

// sidebarItems is the nav the sidebar offers, shared by the expanded rows
// and the collapsed icon rail.
var sidebarItems = []struct {
	key   string
	icon  *ui.SVG
	label string
}{
	{"search", iconSearch, "搜索"},
	{"playlists", iconLibrary, "歌单"},
	{"downloads", iconDownload, "下载"},
	{"tagstudio", iconTag, "Tag Studio"},
	{"settings", iconSettings, "设置"},
}

const (
	sidebarWidth     = 216
	sidebarRailWidth = 56
)

func (a *app) sidebar(c *ui.Context, col colors) {
	bar := c.TitleBar()
	width := float32(sidebarWidth)
	if a.sidebarCollapsed {
		width = sidebarRailWidth
	}
	side := ui.Column(c).Width(width).Shrink(0).FillHeight().Padding(0, 10, 10, 10)

	side.Children(func() {
		// The two states share one vertical skeleton: every section keeps
		// its height across the toggle, so nothing below jumps or resizes;
		// only the rail's width changes.
		//
		// On macOS the traffic lights sit over the rail's top, so the
		// collapsed rail leaves the brand mark out instead of overlapping.
		showLogo := !a.sidebarCollapsed || runtime.GOOS != "darwin"
		ui.Row(c).Height(topBar).Shrink(0).AlignItems(ui.Center).
			Padding(0, bar.Left+10, 0, 10).Gap(8).DragWindow().Children(func() {
			if showLogo {
				a.logoMark(c, col, 22)
			}
			if !a.sidebarCollapsed {
				ui.Text(c, "Qmdr").FontSize(13).FontWeight(600).TextColor(col.text)
			}
		})

		ui.Column(c).Gap(2).Children(func() {
			if a.sidebarCollapsed {
				ui.Row(c).FillWidth().Justify(ui.Center).Padding(0, 0, 6, 0).Children(func() {
					ui.Text(c, "资料库").FontSize(11).FontWeight(600).TextColor(col.faint)
				})
			} else {
				ui.Text(c, "资料库").FontSize(11).FontWeight(600).TextColor(col.faint).
					Padding(0, 9, 6, 9)
			}
			for _, item := range sidebarItems {
				if a.sidebarCollapsed {
					a.navIcon(c, col, item.key, item.icon, item.label)
				} else {
					a.navItem(c, col, item.key, item.icon, item.label)
				}
			}
		})

		ui.Spacer(c)

		a.sidebarToggle(c, col)
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
		if key == "downloads" {
			_, _, _, _, active := a.queueCounts()
			if count := active + len(a.queue.pending); count > 0 {
				ui.Textf(c, "%d", count).FontSize(11).TextColor(col.muted)
			}
		}
	})
}

// navIcon is the collapsed form of navItem: the icon alone, with the label
// as a tooltip, and a small dot when the download queue has work.
func (a *app) navIcon(c *ui.Context, col colors, key string, icon *ui.SVG, label string) {
	active := a.pageKey() == key
	row := ui.ButtonBase(c).Width(36).Height(30).Radius(8).Center().
		Cursor(ui.CursorPointer).Label(label).Tooltip(label)
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
		ui.Box(c).Size(16, 16).Children(func() {
			ui.Row(c).Fill().Center().Children(func() {
				ui.Icon(c, icon).Size(15, 15).TextColor(fg)
			})
			if key == "downloads" && a.sidebarBusy() {
				ui.Box(c).Absolute().Top(0).Right(0).Size(6, 6).Radius(3).Background(col.accent)
			}
		})
	})
}

// sidebarBusy reports whether the downloads entry deserves its badge dot.
func (a *app) sidebarBusy() bool {
	_, _, _, _, active := a.queueCounts()
	return active+len(a.queue.pending) > 0
}

// toggleSidebar flips the sidebar between the full width and the icon rail;
// the choice is persisted with the rest of the settings.
func (a *app) toggleSidebar() {
	a.sidebarCollapsed = !a.sidebarCollapsed
	a.saveSettings()
	a.refresh()
}

// sidebarToggle sits above the account chip as the collapse affordance.
// Both states keep the nav row's 30px height so the account chip below it
// never moves when the sidebar toggles.
func (a *app) sidebarToggle(c *ui.Context, col colors) {
	if a.sidebarCollapsed {
		row := ui.ButtonBase(c).Width(36).Height(30).Radius(8).Center().
			Cursor(ui.CursorPointer).Label("展开侧栏").Tooltip("展开侧栏")
		hovered := row.Hovered()
		if hovered {
			row.Background(col.cardHover)
		}
		if row.Clicked() {
			a.toggleSidebar()
		}
		fg := col.muted
		if hovered {
			fg = col.text
		}
		row.Children(func() {
			ui.Icon(c, iconChevronRight).Size(15, 15).TextColor(fg)
		})
		return
	}
	row := ui.ButtonBase(c).FillWidth().Height(30).Padding(0, 9).Gap(9).
		Radius(8).AlignItems(ui.Center).Cursor(ui.CursorPointer).Label("收起侧栏")
	hovered := row.Hovered()
	if hovered {
		row.Background(col.cardHover)
	}
	if row.Clicked() {
		a.toggleSidebar()
	}
	fg := col.muted
	if hovered {
		fg = col.text
	}
	row.Children(func() {
		ui.Icon(c, iconChevronLeft).Size(15, 15).TextColor(fg)
		ui.Text(c, "收起侧栏").FontSize(13).TextColor(fg).Grow(1)
	})
}

func (a *app) accountChip(c *ui.Context, col colors) {
	if a.sidebarCollapsed {
		chip := ui.ButtonBase(c).Width(36).Height(50).Radius(10).Center().
			Cursor(ui.CursorPointer).Label("账号设置").Tooltip("账号设置")
		fill := col.card
		if chip.Hovered() {
			fill = col.controlHover
		}
		smoothPanel(chip, col, 10, fill, false)
		if chip.Clicked() {
			a.view = viewSettings
			a.refresh()
		}
		chip.Children(func() {
			a.avatar(c, col, 26)
		})
		return
	}
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
				detail, fg := "已登录", col.muted
				if label := a.vipLabel(); label != "" {
					detail, fg = label, a.vipColor(col)
				}
				ui.Text(c, detail).FontSize(11).TextColor(fg).SingleLine().Ellipsis("…")
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
