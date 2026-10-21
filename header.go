package main

import (
	"runtime"

	"github.com/egoist/mygo/ui"
)

func (a *app) cardHeader(c *ui.Context, col colors) {
	ui.Row(c).FillWidth().Height(headerH).Shrink(0).AlignItems(ui.Center).
		Padding(0, 8, 0, 16).DragWindow().Children(func() {
		ui.Text(c, a.pageTitle()).FontSize(15).FontWeight(600).TextColor(col.text)
		ui.Spacer(c)
		a.headerActions(c, col)
		if runtime.GOOS != "darwin" {
			ui.Box(c).Width(6)
			a.windowControls(c, col)
		}
	})
}

func (a *app) pageTitle() string {
	switch a.view {
	case viewPlaylists:
		return "歌单"
	case viewQueue:
		return "队列"
	case viewHistory:
		return "历史"
	case viewSettings:
		return "设置"
	default:
		return "搜索"
	}
}

func (a *app) headerActions(c *ui.Context, col colors) {
	switch a.view {
	case viewSearch, viewHistory, viewSettings:
		iconButton(c, col, iconFolder, "打开下载目录", a.openDownloadDir)
	case viewQueue:
		_, done, _, active := a.queueCounts()
		skipped := 0
		for _, item := range a.queue.items {
			if item.Status == StatusSkipped {
				skipped++
			}
		}
		if a.queue.downloading {
			ghostDangerButton(c, col, iconX, "取消", a.cancelQueue)
		}
		if !a.queue.downloading && done+skipped > 0 {
			iconButton(c, col, iconTrash, "清理已完成", a.clearCompleted)
		}
		if !a.queue.downloading && active > 0 {
			primaryButton(c, col, iconPlay, "开始", a.startQueue)
		}
	}
}

func (a *app) windowControls(c *ui.Context, col colors) {
	if a.win == nil {
		return
	}
	ui.Row(c).Gap(2).AlignItems(ui.Center).Shrink(0).Children(func() {
		minimize := ui.ButtonBase(c).Size(28, 26).Radius(6).Justify(ui.Center).
			AlignItems(ui.Center).Label("最小化").Tooltip("最小化").TextColor(col.muted)
		if minimize.Pressed() {
			minimize.Background(col.cardHover.Alpha(0.12))
		} else if minimize.Hovered() {
			minimize.Background(col.cardHover).TextColor(col.text)
		}
		minimize.Children(func() {
			ui.Icon(c, iconWinMinimize).Size(15, 15)
		})
		if minimize.Clicked() {
			a.win.Minimize()
		}

		maximized := a.win.IsMaximized()
		label, icon := "最大化", iconWinMaximize
		if maximized {
			label, icon = "还原", iconWinRestore
		}
		maximize := ui.ButtonBase(c).Size(28, 26).Radius(6).Justify(ui.Center).
			AlignItems(ui.Center).Label(label).TextColor(col.muted).MaximizeButton()
		if maximize.Pressed() {
			maximize.Background(col.cardHover.Alpha(0.12))
		} else if maximize.Hovered() {
			maximize.Background(col.cardHover).TextColor(col.text)
		}
		maximize.Children(func() {
			ui.Icon(c, icon).Size(15, 15)
		})
		if maximize.Clicked() {
			if maximized {
				a.win.Unmaximize()
			} else {
				a.win.Maximize()
			}
		}

		closeButton := ui.ButtonBase(c).Size(28, 26).Radius(6).Justify(ui.Center).
			AlignItems(ui.Center).Label("关闭").Tooltip("关闭").TextColor(col.muted)
		if closeButton.Hovered() || closeButton.Pressed() {
			closeButton.Background(col.red).TextColor(ui.RGB(255, 255, 255))
		}
		closeButton.Children(func() {
			ui.Icon(c, iconX).Size(15, 15)
		})
		if closeButton.Clicked() {
			a.win.Close()
		}
	})
}
