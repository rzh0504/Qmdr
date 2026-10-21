package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/rzh0504/qmdr/internal/format"
	"github.com/rzh0504/qmdr/internal/history"
	"github.com/rzh0504/qmdr/internal/settings"
)

const historyPageSize = 200

func (a *app) initHistory() {
	path := history.DefaultPath()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	store, err := history.Open(ctx, path)
	if err != nil {
		a.update(func() {
			a.history.loading = false
			a.history.err = friendlyError(err)
			a.history.backlog = nil
		})
		return
	}
	now := time.Now()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	if err := store.PruneQueue(ctx, dayStart); err != nil {
		log.Printf("prune download queue: %v", err)
	}
	if err := store.PruneLogs(ctx, now.AddDate(0, 0, -logRetentionDays)); err != nil {
		log.Printf("prune task logs: %v", err)
	}
	queueRecords, queueErr := store.LoadQueue(ctx, dayStart)
	records, recentErr := store.Recent(ctx, historyPageSize)
	stats, statsErr := store.Stats(ctx)
	if a.closed.Load() {
		_ = store.Close()
		return
	}
	a.update(func() {
		a.history.store = store
		a.flushLogBacklog()
		a.history.loading = false
		a.history.items = records
		a.history.count = stats.Count
		a.history.bytes = stats.Bytes
		a.history.err = ""
		if recentErr != nil {
			a.history.err = "读取记录失败：" + friendlyError(recentErr)
		} else if statsErr != nil {
			a.history.err = "读取统计失败：" + friendlyError(statsErr)
		}
		if queueErr != nil {
			log.Printf("load download queue: %v", queueErr)
		} else {
			a.restoreQueue(queueRecords)
		}
		a.flushHistoryBacklog()
		a.pruneHistoryNow()
	})
}

func (a *app) pruneHistoryNow() {
	store := a.history.store
	if store == nil {
		return
	}
	days := settings.HistoryKeepDays(a.historyKeep)
	if days <= 0 {
		return
	}
	cutoff := time.Now().AddDate(0, 0, -days)
	a.dispatchDisk(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		removed, err := store.Prune(ctx, cutoff)
		if err != nil {
			log.Printf("prune download history: %v", err)
			a.update(func() {
				if a.history.store == store {
					a.toast = "清理过期下载历史失败：" + friendlyError(err)
				}
			})
			return
		}
		if removed == 0 {
			return
		}
		stats, statsErr := store.Stats(ctx)
		a.update(func() {
			if a.history.store != store {
				return
			}
			items := a.history.items[:0]
			for _, record := range a.history.items {
				if !record.CompletedAt.Before(cutoff) {
					items = append(items, record)
				}
			}
			a.history.items = items
			if statsErr == nil {
				a.history.count = stats.Count
				a.history.bytes = stats.Bytes
			}
			a.toast = fmt.Sprintf("已清理 %d 条过期下载记录", removed)
		})
	})
}

func (a *app) retryHistory() {
	if a.history.loading {
		return
	}
	if a.history.store != nil {
		store := a.history.store
		a.history.store = nil
		// Close on the worker, after any job still using the store.
		a.dispatchDisk(func() { _ = store.Close() })
	}
	a.history.loading = true
	a.history.err = ""
	a.history.items = nil
	a.history.count = 0
	a.history.bytes = 0
	a.refresh()
	go a.initHistory()
}

func (a *app) recordHistory(record history.Record) {
	if record.CompletedAt.IsZero() {
		record.CompletedAt = time.Now()
	}
	store := a.history.store
	if store == nil {
		if a.history.loading && a.history.err == "" {
			a.history.backlog = append(a.history.backlog, record)
		}
		return
	}
	a.dispatchDisk(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		id, err := store.Add(ctx, record)
		if err != nil {
			log.Printf("record download history: %v", err)
			return
		}
		a.update(func() {
			if a.history.store != store {
				return
			}
			record.ID = id
			a.prependHistory(record)
		})
	})
}

func (a *app) flushHistoryBacklog() {
	if a.history.store == nil || len(a.history.backlog) == 0 {
		return
	}
	store := a.history.store
	records := a.history.backlog
	a.history.backlog = nil
	a.dispatchDisk(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		var added []history.Record
		for _, record := range records {
			id, err := store.Add(ctx, record)
			if err != nil {
				log.Printf("record download history: %v", err)
				continue
			}
			record.ID = id
			added = append(added, record)
		}
		if len(added) == 0 {
			return
		}
		a.update(func() {
			if a.history.store != store {
				return
			}
			for _, record := range added {
				a.prependHistory(record)
			}
		})
	})
}

func (a *app) prependHistory(record history.Record) {
	a.history.count++
	a.history.bytes += record.FileSize
	a.history.items = append(a.history.items, history.Record{})
	copy(a.history.items[1:], a.history.items[:len(a.history.items)-1])
	a.history.items[0] = record
	if len(a.history.items) > historyPageSize {
		a.history.items = a.history.items[:historyPageSize]
	}
}

func (a *app) confirmClearHistory() {
	if a.history.store == nil || a.history.count == 0 {
		return
	}
	a.history.confirmClear = true
	a.refresh()
}

func (a *app) clearHistoryModal(c *ui.Context, col colors) {
	a.modal(c, col, &a.history.confirmClear, "清空下载历史", 380, func(close func()) {
		ui.Row(c).Gap(10).AlignItems(ui.Start).Children(func() {
			ui.Icon(c, iconAlert).Size(20, 20).TextColor(col.gold)
			ui.Column(c).Gap(4).Grow(1).MinWidth(0).Children(func() {
				ui.Text(c, "确定清空所有下载历史吗？").FontSize(14).TextColor(col.text).LineHeight(1.35)
				ui.Text(c, "只删除记录，不会删除已下载的文件。").FontSize(12).TextColor(col.muted).LineHeight(1.35)
			})
		})
		ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
			secondaryButton(c, col, nil, "取消", close)
			dangerButton(c, col, "清空", func() {
				close()
				a.clearHistory()
			})
		})
	})
}

func (a *app) clearHistory() {
	store := a.history.store
	if store == nil {
		return
	}
	a.dispatchDisk(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := store.Clear(ctx)
		a.update(func() {
			if a.history.store != store {
				return
			}
			if err != nil {
				a.toast = "清空下载历史失败：" + friendlyError(err)
				return
			}
			a.history.items = nil
			a.history.backlog = nil
			a.history.count = 0
			a.history.bytes = 0
			a.toast = "下载历史已清空"
		})
	})
}

func (a *app) historyView(c *ui.Context, col colors) {
	if a.history.loading && a.history.store == nil {
		softCard(c, col, func() {
			ui.Row(c).Gap(9).AlignItems(ui.Center).Children(func() {
				ui.Spinner(c).Size(15, 15)
				ui.Text(c, "正在加载下载历史…").FontSize(12).TextColor(col.muted)
			})
		})
		return
	}
	if a.history.err != "" && len(a.history.items) == 0 {
		softCard(c, col, func() {
			ui.Row(c).Gap(9).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, iconAlert).Size(15, 15).TextColor(col.red)
				ui.Text(c, "下载历史不可用："+a.history.err).FontSize(12).TextColor(col.red).
					Grow(1).MaxLines(3)
				secondaryButton(c, col, iconRefresh, "重试", a.retryHistory)
			})
		})
		return
	}
	if len(a.history.items) == 0 {
		emptyState(c, col, iconClock, "还没有下载记录", "完成下载后，歌曲和文件信息会记录在这里。")
		return
	}

	ui.Column(c).Gap(10).Children(func() {
		if a.history.err != "" {
			ui.Row(c).Gap(9).AlignItems(ui.Center).Padding(0, 4).Children(func() {
				ui.Icon(c, iconAlert).Size(14, 14).TextColor(col.red)
				ui.Text(c, a.history.err).FontSize(11).TextColor(col.red).Grow(1).MaxLines(2)
				secondaryButton(c, col, iconRefresh, "重试", a.retryHistory)
			})
		}

		sizeText := format.ByteSize(a.history.bytes)
		if sizeText == "" {
			sizeText = "0 B"
		}
		ui.Row(c).FillWidth().AlignItems(ui.Center).Padding(0, 4).Children(func() {
			ui.Textf(c, "共 %d 条记录 · 合计 %s", a.history.count, sizeText).FontSize(12).
				TextColor(col.muted).Grow(1)
			ghostDangerButton(c, col, iconTrash, "清空历史", a.confirmClearHistory)
		})

		sheet(c, col, func() {
			for _, record := range a.history.items {
				a.historyRow(c, col, record)
			}
		})

		if a.history.count > len(a.history.items) {
			ui.Textf(c, "仅显示最近 %d 条记录", historyPageSize).FontSize(11).
				TextColor(col.faint).Padding(0, 4)
		}
	})
}

func (a *app) historyRow(c *ui.Context, col colors, record history.Record) {
	row := ui.Row(c).Gap(10).Padding(7, 10).Height(52).Radius(8).AlignItems(ui.Center)
	if row.Hovered() {
		row.Background(col.rowSelected)
		row = chosen(row, col)
	}
	row.Children(func() {
		a.cover(c, col, record.AlbumMID, 34)
		ui.Column(c).Gap(2).Grow(1).MinWidth(0).Children(func() {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Text(c, record.Title).FontSize(13).TextColor(col.text).SingleLine()
				if record.Status == history.StatusSkipped {
					ui.Text(c, "已跳过").FontSize(9).FontWeight(600).
						TextColor(col.gold).SingleLine()
				}
			})
			ui.Textf(c, "%s · %s", record.Artist, record.Album).FontSize(12).
				TextColor(col.muted).SingleLine()
		})
		ui.Text(c, record.Quality).Width(56).FontSize(11).TextColor(col.muted).SingleLine()
		ui.Text(c, format.ByteSize(record.FileSize)).Width(70).FontSize(11).
			TextColor(col.muted).SingleLine()
		ui.Text(c, record.CompletedAt.Local().Format("2006-01-02 15:04")).Width(116).
			FontSize(11).TextColor(col.faint).SingleLine()
		iconButton(c, col, iconFolder, "在文件夹中显示 "+record.Title, func() {
			a.showHistoryFile(record.FilePath)
		})
	})
}

func (a *app) showHistoryFile(path string) {
	if strings.TrimSpace(path) == "" {
		a.setToast("没有可打开的文件路径")
		return
	}
	if _, err := os.Stat(path); err != nil {
		a.setToast("文件不存在或已被移动")
		return
	}
	mygo.Shell.ShowItemInFolder(path)
}
