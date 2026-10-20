package main

import (
	"fmt"

	"github.com/egoist/mygo/ui"
)

func (a *app) queueView(c *ui.Context, col colors) {
	_, _, _, active := a.queueCounts()

	pageHeader(c, col, "下载队列", "查看整体进度、当前任务与下载日志", func() {
		ghostDangerButton(c, col, iconX, "取消全部", func() {
			a.cancelQueue()
		})
		ghostButton(c, col, iconTrash, "清理已完成", func() {
			a.clearCompleted()
		})
		if !a.downloading && active > 0 {
			primaryButton(c, col, iconPlay, "开始队列", func() {
				a.startQueue()
			})
		}
	})

	a.queueSummary(c, col)

	ui.Row(c).Gap(16).AlignItems(ui.Start).Children(func() {
		ui.Column(c).Grow(1).Gap(16).Children(func() {
			a.currentTaskCard(c, col)
			a.queueListCard(c, col)
		})
		ui.Column(c).Width(300).Gap(16).Children(func() {
			a.queueStatsCard(c, col)
			a.queueLogCard(c, col)
		})
	})
}

func (a *app) queueSummary(c *ui.Context, col colors) {
	total, done, failed, active := a.queueCounts()
	skipped := 0
	for _, item := range a.queue {
		if item.Status == StatusSkipped {
			skipped++
		}
	}
	percent := a.overallProgress * 100
	speed := "--"
	if current := a.currentQueueItem(); current != nil && current.Status == StatusDownloading {
		speed = fmt.Sprintf("%.1f MB/s", current.Speed)
	}

	card(c, col, func() {
		ui.Row(c).FillWidth().AlignItems(ui.End).Children(func() {
			ui.Column(c).Gap(2).Grow(1).Children(func() {
				ui.Text(c, "总体进度").FontSize(11).TextColor(col.muted)
				ui.Textf(c, "%.0f%%", percent).FontSize(30).Bold().TextColor(col.text)
				ui.Textf(c, "%d 个任务 · %d 已完成 · %d 失败 · %d 进行中",
					total, done, failed, active).FontSize(11).TextColor(col.muted)
			})
			ui.Column(c).Gap(2).AlignItems(ui.End).Children(func() {
				ui.Text(c, "当前速度").FontSize(10).TextColor(col.muted)
				ui.Text(c, speed).FontSize(12).Bold().TextColor(col.text)
			})
		})

		track := ui.Box(c).Height(6).FillWidth().Radius(3).Background(col.cardAlt)
		track.Children(func() {
			if percent > 0 {
				ui.Box(c).Height(6).WidthPercent(float32(percent)).Radius(3).
					Background(col.accent)
			}
		})

		ui.Textf(c, "已完成 %d · 已跳过 %d · 失败 %d", done, skipped, failed).
			FontSize(10).TextColor(col.muted)
	})
}

func (a *app) currentTaskCard(c *ui.Context, col colors) {
	current := a.currentQueueItem()
	if current == nil || current.Status != StatusDownloading {
		softCard(c, col, func() {
			ui.Row(c).Gap(9).AlignItems(ui.Center).Children(func() {
				ui.Box(c).Size(30, 30).Radius(6).Background(col.card).
					Border(1, col.border).Center().Children(func() {
					ui.Icon(c, iconPlay).FontSize(14).TextColor(col.muted)
				})
				ui.Column(c).Gap(1).Grow(1).Children(func() {
					ui.Text(c, "队列空闲").FontSize(13).Bold().TextColor(col.text)
					ui.Text(c, "从搜索结果或歌单中添加歌曲后，任务会显示在这里。").
						FontSize(11).TextColor(col.muted)
				})
				_, _, _, active := a.queueCounts()
				if active > 0 {
					primaryButton(c, col, iconPlay, "开始下载", func() {
						a.startQueue()
					})
				}
			})
		})
		return
	}

	box := ui.Column(c).Padding(14).Gap(8).Radius(10).
		Background(col.card).Border(1, col.border)
	box.Children(func() {
		ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
			cover(c, col, current.ID, 36, iconMusic)
			ui.Column(c).Gap(1).Grow(1).Children(func() {
				ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
					ui.Text(c, current.Song.Title).FontSize(13).Bold().TextColor(col.text).SingleLine()
					if current.Song.VIP {
						ui.Text(c, "VIP").FontSize(9).Bold().TextColor(col.gold)
					}
					ui.Text(c, current.Quality).FontSize(9).TextColor(col.muted)
				})
				ui.Textf(c, "%s · %.1f MB/s", current.Song.Artist, current.Speed).
					FontSize(10).TextColor(col.muted)
			})
			ui.Textf(c, "%.0f%%", current.Progress*100).FontSize(16).Bold().TextColor(col.text)
			iconButton(c, col, iconX, "取消当前任务", func() {
				a.cancelQueue()
			})
		})

		track := ui.Box(c).Height(5).FillWidth().Radius(2).Background(col.cardAlt)
		track.Children(func() {
			if current.Progress > 0 {
				ui.Box(c).Height(5).WidthPercent(float32(current.Progress * 100)).Radius(2).
					Background(col.accent)
			}
		})
	})
}

func (a *app) queueListCard(c *ui.Context, col colors) {
	box := ui.Column(c).Padding(14).Gap(0).Radius(10).
		Background(col.card).Border(1, col.border).
		Shadow(0, 1, 3, 0, col.shadow)
	box.Children(func() {
		ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
			ui.Text(c, "任务列表").FontSize(14).Bold().TextColor(col.text)
			total, _, _, _ := a.queueCounts()
			pill(c, fmt.Sprint(total), col.muted, col.cardAlt)
			ui.Spacer(c)
			a.queueFilterChip(c, col, 0, "全部")
			a.queueFilterChip(c, col, 1, "进行中")
			a.queueFilterChip(c, col, 2, "已完成")
			a.queueFilterChip(c, col, 3, "失败")
		})

		ui.Box(c).Height(8)

		items := a.filteredQueue()
		if len(items) == 0 {
			emptyState(c, col, iconQueue, "这里还没有任务", "切换筛选条件，或从搜索结果中添加歌曲。")
			return
		}

		ui.Row(c).FillWidth().Padding(4, 8).Children(func() {
			ui.Text(c, "状态").Width(74).FontSize(10).TextColor(col.muted)
			ui.Text(c, "歌曲").Grow(1).FontSize(10).TextColor(col.muted)
			ui.Text(c, "音质").Width(68).FontSize(10).TextColor(col.muted)
			ui.Box(c).Width(92)
			ui.Box(c).Width(28)
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

func (a *app) queueFilterChip(c *ui.Context, col colors, index int, label string) {
	selected := a.queueFilter == index
	btn := ui.ButtonBase(c).Padding(3, 7).Radius(5).
		AlignItems(ui.Center).Label(label).Cursor(ui.CursorPointer)
	if selected {
		btn = btn.Background(col.cardAlt)
	} else if btn.Hovered() {
		btn = btn.Background(col.cardHover)
	}
	if btn.Clicked() {
		a.queueFilter = index
		a.refresh()
	}
	fg := col.muted
	if selected {
		fg = col.text
	}
	btn.Children(func() {
		ui.Text(c, label).FontSize(10).TextColor(fg).SingleLine()
	})
}

func (a *app) queueRow(c *ui.Context, col colors, item QueueItem, index int) {
	row := ui.Row(c).Gap(8).Padding(5, 8).Radius(4).AlignItems(ui.Center)
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
		ui.Row(c).Gap(5).Width(74).AlignItems(ui.Center).Children(func() {
			statusIcon(c, col, item.Status)
			ui.Text(c, item.Status.Label()).FontSize(10).TextColor(fg).SingleLine()
		})

		ui.Row(c).Gap(8).Grow(1).AlignItems(ui.Center).Children(func() {
			cover(c, col, item.ID, 30, iconMusic)
			ui.Column(c).Gap(1).Grow(1).Children(func() {
				ui.Text(c, item.Song.Title).FontSize(12).Bold().
					TextColor(col.text).SingleLine()
				if item.Error != "" {
					ui.Text(c, item.Error).FontSize(9).TextColor(col.red).SingleLine()
				} else {
					ui.Text(c, item.Song.Artist).FontSize(9).TextColor(col.muted).SingleLine()
				}
			})
		})

		ui.Text(c, item.Quality).Width(68).FontSize(9).TextColor(col.muted).SingleLine()

		progressBox := ui.Box(c).Width(92)
		progressBox.Children(func() {
			if item.Status == StatusDownloading {
				ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
					track := ui.Box(c).Height(3).Grow(1).Radius(2).Background(col.cardAlt)
					track.Children(func() {
						if item.Progress > 0 {
							ui.Box(c).Height(3).WidthPercent(float32(item.Progress * 100)).
								Radius(2).Background(col.accent)
						}
					})
					ui.Textf(c, "%.0f%%", item.Progress*100).FontSize(9).TextColor(col.muted)
				})
			}
		})

		switch item.Status {
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
		default:
			ui.Box(c).Size(28, 28)
		}
	})
}

func (a *app) queueStatsCard(c *ui.Context, col colors) {
	total, done, failed, active := a.queueCounts()
	skipped := 0
	for _, item := range a.queue {
		if item.Status == StatusSkipped {
			skipped++
		}
	}
	card(c, col, func() {
		sectionHeader(c, col, "下载统计", "", nil)
		a.statLine(c, col, "全部任务", fmt.Sprint(total))
		a.statLine(c, col, "已完成", fmt.Sprint(done))
		a.statLine(c, col, "已跳过", fmt.Sprint(skipped))
		a.statLine(c, col, "失败", fmt.Sprint(failed))
		a.statLine(c, col, "等待 / 下载中", fmt.Sprint(active))
	})
}

func (a *app) statLine(c *ui.Context, col colors, label, value string) {
	ui.Row(c).FillWidth().AlignItems(ui.Center).Padding(3, 0).Children(func() {
		ui.Text(c, label).FontSize(11).TextColor(col.muted).Grow(1)
		ui.Text(c, value).FontSize(12).Bold().TextColor(col.text)
	})
}

func (a *app) queueLogCard(c *ui.Context, col colors) {
	box := ui.Column(c).Padding(14).Gap(8).Radius(10).
		Background(col.card).Border(1, col.border).
		Shadow(0, 1, 3, 0, col.shadow)
	box.Children(func() {
		ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
			ui.Text(c, "任务日志").FontSize(14).Bold().TextColor(col.text)
			ui.Spacer(c)
			ui.Text(c, "实时").FontSize(9).TextColor(col.muted)
		})

		logs := ui.Scroll(c).Height(260).Gap(6)
		logs.Children(func() {
			if len(a.logs) == 0 {
				emptyState(c, col, iconClock, "暂无日志", "任务状态会显示在这里。")
				return
			}
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
					ui.Text(c, entry.Time).Width(44).FontSize(9).TextColor(col.faint)
					ui.Text(c, entry.Message).FontSize(10).TextColor(fg).Grow(1).MaxLines(2)
				})
			}
		})

		secondaryButton(c, col, iconFolder, "打开下载目录", func() {
			a.openDownloadDir()
		})
	})
}
