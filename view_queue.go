package main

import (
	"github.com/egoist/mygo/ui"
)

func (a *app) queueView(c *ui.Context, col colors) {
	total, _, failed, active := a.queueCounts()
	if total == 0 && len(a.pending) == 0 {
		emptyState(c, col, iconQueue, "队列为空", "在歌单中右键歌曲加入待下载队列，或直接下载歌曲。")
		return
	}

	activeItems := a.filteredQueue()
	completedItems := a.completedQueue()

	ui.Column(c).Gap(10).Children(func() {
		if len(a.pending) > 0 {
			a.pendingSection(c, col)
		}
		if len(a.queue) > 0 {
			sectionHeader(c, col, "下载队列", func() {
				ui.Textf(c, "进行中 %d · 失败 %d", active, failed).FontSize(12).
					TextColor(col.faint)
			})

			ui.Row(c).Height(32).FillWidth().AlignItems(ui.Center).Children(func() {
				a.queueFilterTabs(c, col)
				ui.Spacer(c)
				ui.Textf(c, "%d 个任务", len(activeItems)).FontSize(12).TextColor(col.faint)
			})

			if len(activeItems) == 0 {
				if active+failed > 0 {
					emptyState(c, col, iconQueue, "没有符合条件的任务", "切换筛选条件查看其他任务。")
				}
			} else {
				sheet(c, col, func() {
					for _, item := range activeItems {
						a.queueRow(c, col, item)
					}
				})
			}
		}
		if len(completedItems) > 0 {
			sectionHeader(c, col, "已下载", func() {
				ui.Textf(c, "%d 首", len(completedItems)).FontSize(12).TextColor(col.faint)
				ghostButton(c, col, nil, "清理", a.clearCompleted)
			})
			sheet(c, col, func() {
				for _, item := range completedItems {
					a.queueRow(c, col, item)
				}
			})
		}
	})

	if total > 0 || len(a.logs) > 0 {
		a.queueLogs(c, col)
	}
}

func (a *app) queueFilterTabs(c *ui.Context, col colors) ui.Element {
	labels := []string{"全部", "进行中", "失败"}
	track := ui.Row(c).Height(28).Padding(2).Radius(7).Gap(1).
		Background(col.chip).AlignItems(ui.Center).Shrink(0)
	track.Children(func() {
		for i, label := range labels {
			i, label := i, label
			seg := ui.ButtonBase(c).Height(24).Padding(0, 12).Radius(5).Center().
				Cursor(ui.CursorPointer).Label(label)
			if i == a.queueFilter {
				seg.Background(col.segThumb)
				if col.dark {
					seg.Shadow(0, 1, 1, 0, ui.RGBA(0, 0, 0, 0.25))
				} else {
					seg.Shadow(0, 0.5, 1.5, 0, ui.RGBA(0, 0, 0, 0.16)).
						Shadow(0, 0, 0, 0.5, ui.RGBA(0, 0, 0, 0.04))
				}
			} else if seg.Hovered() {
				seg.Background(col.cardHover)
			}
			if seg.Clicked() {
				a.queueFilter = i
				a.refresh()
			}
			seg.Children(func() {
				ui.Text(c, label).FontSize(13).SingleLine().TextColor(col.text)
			})
		}
	})
	return track
}

func (a *app) pendingSection(c *ui.Context, col colors) {
	sectionHeader(c, col, "待下载队列", func() {
		ui.Textf(c, "%d 首", len(a.pending)).FontSize(12).TextColor(col.faint)
		ghostButton(c, col, nil, "清空", a.clearPending)
		primaryButton(c, col, iconDownload, "全部下载", a.startPendingDownloads)
	})
	sheet(c, col, func() {
		for i, song := range a.pending {
			i, song := i, song
			row := ui.Row(c).Gap(10).Padding(6, 10).Height(50).Radius(8).AlignItems(ui.Center)
			if row.Hovered() {
				row.Background(col.rowSelected)
				row = chosen(row, col)
			}
			row.Children(func() {
				a.cover(c, col, song.AlbumMid, 30)
				ui.Column(c).Gap(2).Grow(1).MinWidth(0).Children(func() {
					ui.Text(c, song.Title).FontSize(13).TextColor(col.text).SingleLine()
					ui.Textf(c, "%s · %s", song.Artist, song.Album).FontSize(11).
						TextColor(col.muted).SingleLine()
				})
				ui.Text(c, song.Duration).Width(38).FontSize(11).TextColor(col.faint).SingleLine()
				iconButton(c, col, iconTrash, "从待下载队列移除 "+song.Title, func() {
					a.removePending(i)
				})
			})
		}
	})
}

func (a *app) queueRow(c *ui.Context, col colors, item QueueItem) {
	row := ui.Row(c).Gap(9).Padding(6, 10).Height(54).Radius(8).AlignItems(ui.Center)
	if row.Hovered() {
		row.Background(col.rowSelected)
		row = chosen(row, col)
	}
	row.Children(func() {
		fg := col.muted
		switch item.Status {
		case StatusDownloading:
			fg = col.accent
		case StatusDone:
			fg = col.green
		case StatusFailed:
			fg = col.red
		case StatusCancelled:
			fg = col.gold
		}
		ui.Row(c).Gap(6).Width(66).AlignItems(ui.Center).Children(func() {
			statusIcon(c, col, item.Status)
			ui.Text(c, item.Status.Label()).FontSize(11).TextColor(fg).SingleLine()
		})

		ui.Row(c).Gap(9).Grow(1).MinWidth(0).AlignItems(ui.Center).Children(func() {
			a.cover(c, col, item.Song.AlbumMid, 30)
			ui.Column(c).Gap(2).Grow(1).MinWidth(0).Children(func() {
				ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
					ui.Text(c, item.Song.Title).FontSize(13).FontWeight(400).
						TextColor(col.text).SingleLine()
					if item.Song.VIP {
						ui.Text(c, "VIP").FontSize(9).FontWeight(600).TextColor(col.gold)
					}
				})
				if item.Error != "" {
					ui.Text(c, item.Error).FontSize(11).TextColor(col.red).SingleLine()
				} else {
					ui.Text(c, item.Song.Artist).FontSize(12).TextColor(col.muted).SingleLine()
				}
			})
		})

		ui.Text(c, item.Quality).Width(56).FontSize(11).TextColor(col.muted).SingleLine()
		ui.Text(c, formatFileSize(item.FileSize)).Width(70).FontSize(11).
			TextColor(col.muted).SingleLine()

		progressBox := ui.Box(c).Width(84)
		progressBox.Children(func() {
			if item.Status == StatusDownloading {
				ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
					track := ui.Box(c).Height(3).Grow(1).Radius(2).Background(col.chip)
					track.Children(func() {
						if item.Progress > 0 {
							ui.Box(c).Height(3).WidthPercent(float32(item.Progress * 100)).
								Radius(2).Background(col.accent)
						}
					})
					ui.Textf(c, "%.0f%%", item.Progress*100).Width(30).FontSize(11).
						TextColor(col.muted).TextAlign(ui.End)
				})
			}
		})

		switch item.Status {
		case StatusDownloading:
			iconButton(c, col, iconX, "取消当前任务", func() {
				a.cancelQueue()
			})
		case StatusFailed, StatusCancelled:
			iconButton(c, col, iconRefresh, "重试 "+item.Song.Title, func() {
				a.retryItem(item.ID)
			})
		case StatusDone, StatusSkipped:
			iconButton(c, col, iconFolder, "打开下载目录", func() {
				a.openDownloadDir()
			})
		case StatusWaiting:
			iconButton(c, col, iconTrash, "移除 "+item.Song.Title, func() {
				a.removeItem(item.ID)
			})
		}
	})
}

func (a *app) queueLogs(c *ui.Context, col colors) {
	toggle := ui.ButtonBase(c).FillWidth().Padding(2, 4).Gap(6).
		AlignItems(ui.Center).Cursor(ui.CursorPointer).Label("任务日志")
	if toggle.Clicked() {
		a.showLogs = !a.showLogs
		a.refresh()
	}
	toggle.Children(func() {
		ui.Text(c, "任务日志").FontSize(12).FontWeight(600).TextColor(col.muted)
		ui.Textf(c, "%d", len(a.logs)).FontSize(12).TextColor(col.faint)
		ui.Spacer(c)
		label := "展开"
		icon := iconChevronDown
		if a.showLogs {
			label = "收起"
			icon = iconChevronUp
		}
		ui.Text(c, label).FontSize(12).TextColor(col.faint)
		ui.Icon(c, icon).Size(13, 13).TextColor(col.faint)
	})

	if !a.showLogs {
		return
	}
	if len(a.logs) == 0 {
		ui.Text(c, "暂无日志").FontSize(12).TextColor(col.faint).Padding(4, 4)
		return
	}

	panel := ui.Column(c).FillWidth().Padding(10, 12).Radius(10).Background(col.chip)
	panel.Children(func() {
		ui.Scroll(c).MaxHeight(160).Gap(3).Children(func() {
			for _, entry := range a.logs {
				fg := col.muted
				switch entry.Level {
				case "success":
					fg = col.green
				case "error":
					fg = col.red
				case "warning":
					fg = col.gold
				}
				ui.Row(c).Gap(10).AlignItems(ui.Start).Children(func() {
					ui.Text(c, entry.Time).Width(58).Font("monospace").FontSize(11).
						TextColor(col.faint).SingleLine()
					ui.Text(c, entry.Message).Font("monospace").FontSize(11).
						TextColor(fg).Grow(1).MaxLines(2).LineHeight(1.45)
				})
			}
		})
	})
}
