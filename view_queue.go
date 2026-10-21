package main

import (
	"github.com/egoist/mygo/ui"
)

func (a *app) queueView(c *ui.Context, col colors) {
	total, done, failed, active := a.queueCounts()
	skipped := 0
	for _, item := range a.queue {
		if item.Status == StatusSkipped {
			skipped++
		}
	}

	if total == 0 && len(a.pending) == 0 {
		emptyState(c, col, iconQueue, "队列为空", "在歌单中右键歌曲加入待下载队列，或直接下载歌曲。")
		return
	}

	items := a.filteredQueue()

	ui.Column(c).Gap(10).Children(func() {
		if len(a.pending) > 0 {
			a.pendingSection(c, col)
		}
		if total == 0 {
			return
		}
		a.queueSummary(c, col, total, done, failed, skipped, active)

		ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
			filter := segmentedControl(c, col, &a.queueFilter, "全部", "进行中", "已完成", "失败")
			if filter.Changed() {
				a.refresh()
			}
			ui.Spacer(c)
			ui.Textf(c, "%d 个任务", len(items)).FontSize(12).TextColor(col.faint)
		})

		if len(items) == 0 {
			emptyState(c, col, iconQueue, "没有符合条件的任务", "切换筛选条件查看其他任务。")
			return
		}
		sheet(c, col, func() {
			for _, item := range items {
				a.queueRow(c, col, item)
			}
		})
	})

	if total > 0 || len(a.logs) > 0 {
		a.queueLogs(c, col)
	}
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

func (a *app) queueSummary(c *ui.Context, col colors, total, done, failed, skipped, active int) {
	processed := done + failed + skipped
	percent := a.overallProgress * 100
	activeDownloads := 0
	speed := 0.0
	for _, item := range a.queue {
		if item.Status == StatusDownloading {
			activeDownloads++
			speed += item.Speed
		}
	}

	ui.Row(c).FillWidth().AlignItems(ui.End).Children(func() {
		ui.Column(c).Gap(2).Grow(1).Children(func() {
			ui.Row(c).Gap(9).AlignItems(ui.End).Children(func() {
				ui.Textf(c, "%.0f%%", percent).FontSize(24).FontWeight(600).TextColor(col.text)
				ui.Textf(c, "%d / %d", processed, total).FontSize(11).TextColor(col.faint)
			})
			ui.Textf(c, "完成 %d · 跳过 %d · 失败 %d · 进行中 %d", done, skipped, failed, active).
				FontSize(12).TextColor(col.muted)
		})
		if activeDownloads > 0 {
			ui.Textf(c, "%d 个下载中 · %.1f MB/s", activeDownloads, speed).
				FontSize(12).TextColor(col.muted)
		}
	})

	track := ui.Box(c).Height(4).FillWidth().Radius(2).Background(col.chip)
	track.Children(func() {
		if percent > 0 {
			ui.Box(c).Height(4).WidthPercent(float32(percent)).Radius(2).
				Background(col.accent)
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
					ui.Textf(c, "%.0f%%", item.Progress*100).FontSize(11).TextColor(col.muted)
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
