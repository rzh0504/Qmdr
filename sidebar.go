package main

import (
	"time"

	"github.com/egoist/mygo/ui"
)

// sidebarItems is the nav the sidebar offers; one renderer serves both the
// expanded rows and the collapsed rail.
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
	sidebarRailWidth = 64
	// The labels fade out over the fade range and the tree switches to the
	// rail at the split, so a toggle glides instead of snapping.
	sidebarSplit = (sidebarWidth + sidebarRailWidth) / 2
	sidebarFade  = 55

	// The nav icon is 20px in both trees and glides from the expanded row's
	// left edge to the rail's center; the rail rows grow taller and spread
	// apart on the way.
	sidebarIconPad  = float32(9)
	sidebarRailIcon = float32(sidebarRailWidth-20-24) / 2
	sidebarChipPad  = float32(10)
	sidebarRailChip = float32(sidebarRailWidth-20-26) / 2
)

func (a *app) sidebar(c *ui.Context, col colors) {
	bar := c.TitleBar()
	// A host element in front of the column carries the width animation: it
	// renders at the same tree spot every frame, so its animated value
	// survives across frames and sizes the column that follows it.
	host := ui.Box(c).Size(1, 1).Shrink(0)
	target := float32(sidebarRailWidth)
	if !a.sidebarCollapsed {
		target = sidebarWidth
	}
	width := host.Animate("sidebar-width", target, 200*time.Millisecond)
	rail := width < sidebarSplit
	alpha := min(max((width-sidebarSplit)/sidebarFade, 0), 1)
	glide := min(max((sidebarWidth-width)/(sidebarWidth-sidebarRailWidth), 0), 1)

	side := ui.Column(c).Width(width).Shrink(0).FillHeight().Padding(0, 10, 10, 10)

	side.Children(func() {
		top := ui.Row(c).Height(topBar).Shrink(0).AlignItems(ui.Center).
			Padding(0, bar.Left+10, 0, 10).Gap(8).DragWindow()
		if rail {
			// The rail's top slot is the expand button alone, centered on
			// the nav icons' axis below.
			top = top.Justify(ui.Center)
		}
		top.Children(func() {
			if rail {
				a.sidebarToggle(c, col, rail)
				return
			}
			a.logoMark(c, col, 22)
			ui.Text(c, "Qmdr").FontSize(13).FontWeight(600).
				TextColor(col.text.Alpha(alpha))
			ui.Spacer(c)
			a.sidebarToggle(c, col, rail)
		})

		ui.Column(c).Gap(2+4*glide).Children(func() {
			for _, item := range sidebarItems {
				a.navItem(c, col, item.key, item.icon, item.label, alpha, glide)
			}
		})

		ui.Spacer(c)

		a.accountChip(c, col, alpha, glide)
	})
}

// sidebarToggle draws the collapse affordance: a 20px chevron in the title
// bar, at its far corner when expanded and centered on the rail's icon axis
// when collapsed.
func (a *app) sidebarToggle(c *ui.Context, col colors, rail bool) {
	icon, tip := iconChevronLeft, "收起侧栏"
	if rail {
		icon, tip = iconChevronRight, "展开侧栏"
	}
	btn := ui.ButtonBase(c).Size(32, 30).Radius(8).Justify(ui.Center).
		AlignItems(ui.Center).Cursor(ui.CursorPointer).Label(tip).Tooltip(tip)
	if btn.Hovered() {
		btn.Background(col.cardHover)
	}
	if btn.Clicked() {
		a.toggleSidebar()
	}
	fg := col.muted
	if btn.Hovered() {
		fg = col.text
	}
	btn.Children(func() {
		ui.Icon(c, icon).Size(20, 20).TextColor(fg)
	})
}

// navItem renders one nav entry in both trees: the icon glides from the row's
// left edge to the rail's center while the label fades out, so a collapse
// reads as one continuous motion and the resting rail is centered.
func (a *app) navItem(c *ui.Context, col colors, key string, icon *ui.SVG, label string, alpha, glide float32) {
	active := a.pageKey() == key
	pad := sidebarIconPad + (sidebarRailIcon-sidebarIconPad)*glide
	height := 34 + 4*glide
	row := ui.ButtonBase(c).FillWidth().Height(height).Padding(0, 9, 0, pad).Gap(12).
		Radius(8).Justify(ui.Start).AlignItems(ui.Center).
		Cursor(ui.CursorPointer).Label(label)
	if alpha < 1 {
		row = row.Tooltip(label)
	}
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
		ui.Box(c).Size(24, 24).Children(func() {
			ui.Row(c).Fill().Center().Children(func() {
				ui.Icon(c, icon).Size(20, 20).TextColor(fg)
			})
			if key == "downloads" && glide > 0.5 && a.sidebarBusy() {
				ui.Box(c).Absolute().Top(0).Right(0).Size(6, 6).Radius(3).Background(col.accent)
			}
		})
		ui.Text(c, label).FontSize(14).FontWeight(400).TextColor(fg.Alpha(alpha)).
			Grow(1).SingleLine().Ellipsis("…")
		if key == "downloads" {
			_, _, _, _, active := a.queueCounts()
			if count := active + len(a.queue.pending); count > 0 && alpha > 0 {
				ui.Textf(c, "%d", count).FontSize(12).TextColor(col.muted.Alpha(alpha))
			}
		}
	})
}

// sidebarBusy reports whether the downloads entry deserves its badge dot.
func (a *app) sidebarBusy() bool {
	_, _, _, _, active := a.queueCounts()
	return active+len(a.queue.pending) > 0
}

// toggleSidebar flips the sidebar between the full width and the icon rail;
// the choice is persisted with the rest of the settings. The width itself
// eases there through the sidebar animation.
func (a *app) toggleSidebar() {
	a.sidebarCollapsed = !a.sidebarCollapsed
	a.saveSettings()
	a.refresh()
}

// accountChip renders the account entry in both trees: the avatar glides to
// the rail's center while the name, membership and chevron fade out. There
// is no card fill in either state — only a hover tint for the click target.
func (a *app) accountChip(c *ui.Context, col colors, alpha, glide float32) {
	pad := sidebarChipPad + (sidebarRailChip-sidebarChipPad)*glide
	chip := ui.ButtonBase(c).FillWidth().Height(50).Padding(0, 10*(1-glide), 0, pad).Gap(12).
		Radius(8).Justify(ui.Start).AlignItems(ui.Center).
		Cursor(ui.CursorPointer).Label("账号设置").Tooltip("账号设置")
	if chip.Hovered() {
		chip.Background(col.cardHover)
	}
	if chip.Clicked() {
		a.view = viewSettings
		a.refresh()
	}
	chip.Children(func() {
		a.avatar(c, col, 26)
		if alpha > 0 {
			ui.Column(c).Gap(1).Grow(1).MinWidth(0).Children(func() {
				name := a.login.userID
				if a.login.nick != "" {
					name = a.login.nick
				}
				ui.Text(c, name).FontSize(12).FontWeight(400).
					TextColor(col.text.Alpha(alpha)).SingleLine().Ellipsis("…")
				if a.login.loggedIn {
					detail, fg := "已登录", col.muted
					if label := a.vipLabel(); label != "" {
						detail, fg = label, a.vipColor(col)
					}
					ui.Text(c, detail).FontSize(11).TextColor(fg.Alpha(alpha)).
						SingleLine().Ellipsis("…")
				}
			})
			ui.Icon(c, iconChevronRight).Size(13, 13).TextColor(col.faint.Alpha(alpha))
		}
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
