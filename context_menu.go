package main

import (
	"github.com/egoist/mygo/ui"
)

const (
	contextMenuItemH     = 30
	contextMenuSeparator = 9
	contextMenuPadding   = 4
	contextMenuRadius    = 10
)

// songMenuItems lists the actions a song row's context menu offers.
// selecting is passed by the playlists page only, so a hidden selection
// mode there never changes the menu of another list.
func (a *app) songMenuItems(song Song, selecting bool) []menuAction {
	if selecting {
		return []menuAction{
			{label: "下载选中歌曲", disabled: a.selectedSongCount() == 0, run: a.downloadSelectedSongs},
			{label: "选中项加入待下载队列", run: a.addSelectedToPending},
			{separator: true},
			{label: "全选", run: a.selectAllSongs},
			{label: "退出多选", run: a.endSongSelection},
		}
	}
	return []menuAction{
		{label: "下载这首歌", run: func() { a.downloadSongNow(song) }},
		{label: "加入待下载队列", run: func() { a.addToPending(song) }},
	}
}

// openSongMenu opens the menu at the pointer when row was right-clicked.
func (a *app) openSongMenu(row ui.Element, song Song, selecting bool) {
	if !row.RightClicked() {
		return
	}
	b := row.Bounds()
	x, y, over := row.PointerPosition()
	if !over {
		x, y = b.W/2, b.H/2
	}
	a.contextMenu = contextMenuState{open: true, x: b.X + x, y: b.Y + y, items: a.songMenuItems(song, selecting)}
	a.refresh()
}

// contextMenuWidthFor fits the menu to its longest label, so short menus
// do not carry the width of the longest selection item.
func contextMenuWidthFor(items []menuAction) float32 {
	width := float32(0)
	for _, item := range items {
		if item.separator {
			continue
		}
		width = max(width, 40+13*float32(len([]rune(item.label))))
	}
	return max(140, min(width, 220))
}

func (a *app) closeMenu() {
	if !a.contextMenu.open {
		return
	}
	a.contextMenu.open = false
	a.refresh()
}

// menuView draws the context menu above the window with a continuous
// rounded rectangle; the system's menu only takes the corner Windows draws.
func (a *app) menuView(c *ui.Context, col colors) {
	if !a.contextMenu.open {
		return
	}
	menu := a.contextMenu
	height := float32(2 * contextMenuPadding)
	for _, item := range menu.items {
		if item.separator {
			height += contextMenuSeparator
		} else {
			height += contextMenuItemH
		}
	}
	width := contextMenuWidthFor(menu.items)
	winW, winH := c.Size()
	x := min(max(menu.x, 8), max(8, winW-width-8))
	y := min(max(menu.y, 8), max(8, winH-height-8))

	ui.Overlay(c, func() {
		back := ui.Box(c).Absolute().Left(0).Top(0).Right(0).Bottom(0)
		if back.Clicked() || back.RightClicked() {
			a.closeMenu()
			return
		}
		panel := ui.Column(c).Absolute().Left(x).Top(y).Width(width).Padding(contextMenuPadding)
		// The opaque base keeps the surface solid even if a repaint skips
		// the smooth path fill; the stroke makes the edge unmistakable.
		panel.Background(col.card)
		smoothPanel(panel, col, contextMenuRadius, col.card, false)
		panel.Shadow(0, 12, 32, 0, col.floatShadow)
		if !col.dark {
			panel.DrawOver(func(p *ui.Painter, r ui.Rect) {
				path := smoothRectPath(snapRect(p, r), contextMenuRadius)
				p.StrokePath(&path, 1, col.border)
			})
		}
		panel.Children(func() {
			for _, item := range menu.items {
				item := item
				if item.separator {
					ui.Row(c).Height(contextMenuSeparator).FillWidth().
						AlignItems(ui.Center).Padding(0, 8).Children(func() {
						ui.Box(c).Height(1).FillWidth().Background(col.border)
					})
					continue
				}
				row := ui.ButtonBase(c).Height(contextMenuItemH).FillWidth().Radius(7).
					Padding(0, 10).Justify(ui.Start).AlignItems(ui.Center)
				if !item.disabled {
					row.Cursor(ui.CursorPointer)
					if row.Hovered() {
						smoothFlat(row, col, 7, col.rowSelected)
					}
					if row.Clicked() {
						a.contextMenu.open = false
						item.run()
						a.refresh()
					}
				}
				textColor := col.text
				if item.disabled {
					textColor = col.faint
				}
				row.Children(func() {
					ui.Text(c, item.label).FontSize(13).TextColor(textColor).SingleLine()
				})
			}
		})
		if panel.OverlayShortcut(0, ui.KeyEscape) {
			a.closeMenu()
		}
	})
}
