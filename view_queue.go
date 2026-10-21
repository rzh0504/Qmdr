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

	pageHeader(c, col, "下载队列", func() {
		if a.downloading {
			ghostDangerButton(c, col, iconX, "取消下载", func() {
				a.cancelQueue()
			})
		}
		if done+skipped > 0 {
			ghostButton(c, col, iconTrash, "清理已完成", func() {
				a.clearCompleted()
			})
		}
		if !a.downloading && active > 0 {
			primaryButton(c, col, iconPlay, "开始队列", func() {
				a.startQueue()
			})
		}
	})

	if total == 0 {
		emptyState(c, col, iconQueue, "队列为空", "从搜索结果下载歌曲后，任务会显示在这里。")
		return
	}

	a.queueSummary(c, col, total, done, failed, skipped, active)

	ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
		filter := ui.Segmented(c, &a.queueFilter, "全部", "进行中", "已完成", "失败").Label("筛选")
		if filter.Changed() {
			a.refresh()
		}
		ui.Spacer(c)
		ui.Textf(c, "%d 个任务", len(a.filteredQueue())).FontSize(9).TextColor(col.faint)
	})

	items := a.filteredQueue()
	if len(items) == 0 {
		emptyState(c, col, iconQueue, "没有符合条件的任务", "切换筛选条件查看其他任务。")
	} else {
		tableCard(c, col, func() {
			ui.Row(c).FillWidth().Padding(3, 8).Children(func() {
				ui.Text(c, "状态").Width(68).FontSize(9).TextColor(col.muted)
				ui.Text(c, "歌曲").Grow(1).FontSize(9).TextColor(col.muted)
				ui.Text(c, "音质").Width(58).FontSize(9).TextColor(col.muted)
				ui.Text(c, "进度").Width(80).FontSize(9).TextColor(col.muted)
				ui.Box(c).Width(26)
			})
			divider(c, col)
			for i, item := range items {
				a.queueRow(c, col, item, i)
				if i < len(items)-1 {
					divider(c, col)
				}
			}
		})
	}

	a.queueLogs(c, col)
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
			ui.Row(c).Gap(8).AlignItems(ui.End).Children(func() {
				ui.Textf(c, "%.0f%%", percent).FontSize(24).Bold().TextColor(col.text)
				ui.Textf(c, "%d / %d", processed, total).FontSize(9).TextColor(col.faint)
			})
			ui.Textf(c, "完成 %d · 跳过 %d · 失败 %d · 进行中 %d", done, skipped, failed, active).
				FontSize(9).TextColor(col.muted)
		})
		if activeDownloads > 0 {
			ui.Textf(c, "%d 个下载中 · %.1f MB/s", activeDownloads, speed).
				FontSize(9).TextColor(col.muted)
		}
	})

	track := ui.Box(c).Height(4).FillWidth().Radius(2).Background(col.cardAlt)
	track.Children(func() {
		if percent > 0 {
			ui.Box(c).Height(4).WidthPercent(float32(percent)).Radius(2).
				Background(col.accent)
		}
	})
}

func (a *app) queueRow(c *ui.Context, col colors, item QueueItem, index int) {
	row := ui.Row(c).Gap(8).Padding(5, 8).Radius(3).AlignItems(ui.Center)
	if row.Hovered() {
		row = row.Background(col.cardHover)
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
		ui.Row(c).Gap(5).Width(68).AlignItems(ui.Center).Children(func() {
			statusIcon(c, col, item.Status)
			ui.Text(c, item.Status.Label()).FontSize(9).TextColor(fg).SingleLine()
		})

		ui.Row(c).Gap(8).Grow(1).AlignItems(ui.Center).Children(func() {
			cover(c, col, item.ID, 28, iconMusic)
			ui.Column(c).Gap(1).Grow(1).Children(func() {
				ui.Row(c).Gap(5).AlignItems(ui.Center).Children(func() {
					ui.Text(c, item.Song.Title).FontSize(11).Bold().
						TextColor(col.text).SingleLine()
					if item.Song.VIP {
						ui.Text(c, "VIP").FontSize(8).Bold().TextColor(col.gold)
					}
				})
				if item.Error != "" {
					ui.Text(c, item.Error).FontSize(9).TextColor(col.red).SingleLine()
				} else {
					ui.Text(c, item.Song.Artist).FontSize(9).TextColor(col.muted).SingleLine()
				}
			})
		})

		ui.Text(c, item.Quality).Width(58).FontSize(9).TextColor(col.muted).SingleLine()

		progressBox := ui.Box(c).Width(80)
		progressBox.Children(func() {
			if item.Status == StatusDownloading {
				ui.Row(c).Gap(5).AlignItems(ui.Center).Children(func() {
					track := ui.Box(c).Height(3).Grow(1).Radius(2).Background(col.cardAlt)
					track.Children(func() {
						if item.Progress > 0 {
							ui.Box(c).Height(3).WidthPercent(float32(item.Progress * 100)).
								Radius(2).Background(col.accent)
						}
					})
					ui.Textf(c, "%.0f%%", item.Progress*100).FontSize(8).TextColor(col.muted)
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
	divider(c, col)

	toggle := ui.ButtonBase(c).FillWidth().Padding(4, 0).
		AlignItems(ui.Center).Label("任务日志")
	if toggle.Clicked() {
		a.showLogs = !a.showLogs
		a.refresh()
	}
	toggle.Children(func() {
		ui.Text(c, "任务日志").FontSize(11).Bold().TextColor(col.muted)
		ui.Textf(c, "%d", len(a.logs)).FontSize(9).TextColor(col.faint)
		ui.Spacer(c)
		label := "展开"
		icon := iconChevronDown
		if a.showLogs {
			label = "收起"
			icon = iconChevronUp
		}
		ui.Text(c, label).FontSize(9).TextColor(col.faint)
		ui.Icon(c, icon).FontSize(11).TextColor(col.faint)
	})

	if !a.showLogs {
		return
	}
	if len(a.logs) == 0 {
		ui.Text(c, "暂无日志").FontSize(10).TextColor(col.faint).Padding(4, 0)
		return
	}

	ui.Scroll(c).Height(160).Gap(4).Children(func() {
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
			ui.Row(c).Gap(6).AlignItems(ui.Start).Children(func() {
				ui.Text(c, entry.Time).Width(40).FontSize(8).TextColor(col.faint)
				ui.Text(c, entry.Message).FontSize(9).TextColor(fg).Grow(1).MaxLines(2)
			})
		}
	})
}
