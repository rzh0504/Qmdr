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
	"github.com/rzh0504/qmdr/internal/history"
)

const historyPageSize = 200

func (a *app) initHistory() {
	path := history.DefaultPath()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	store, err := history.Open(ctx, path)
	if err != nil {
		a.update(func() {
			a.historyLoading = false
			a.historyErr = friendlyError(err)
			a.historyBacklog = nil
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
		a.historyStore = store
		a.flushLogBacklog()
		a.historyLoading = false
		a.historyItems = records
		a.historyCount = stats.Count
		a.historyBytes = stats.Bytes
		a.historyErr = ""
		if recentErr != nil {
			a.historyErr = "读取记录失败：" + friendlyError(recentErr)
		} else if statsErr != nil {
			a.historyErr = "读取统计失败：" + friendlyError(statsErr)
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
	store := a.historyStore
	if store == nil {
		return
	}
	days := historyKeepDays(a.historyKeep)
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
				if a.historyStore == store {
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
			if a.historyStore != store {
				return
			}
			items := a.historyItems[:0]
			for _, record := range a.historyItems {
				if !record.CompletedAt.Before(cutoff) {
					items = append(items, record)
				}
			}
			a.historyItems = items
			if statsErr == nil {
				a.historyCount = stats.Count
				a.historyBytes = stats.Bytes
			}
			a.toast = fmt.Sprintf("已清理 %d 条过期下载记录", removed)
		})
	})
}

func (a *app) retryHistory() {
	if a.historyLoading {
		return
	}
	if a.historyStore != nil {
		store := a.historyStore
		a.historyStore = nil
		// Close on the worker, after any job still using the store.
		a.dispatchDisk(func() { _ = store.Close() })
	}
	a.historyLoading = true
	a.historyErr = ""
	a.historyItems = nil
	a.historyCount = 0
	a.historyBytes = 0
	a.refresh()
	go a.initHistory()
}

func (a *app) recordHistory(record history.Record) {
	if record.CompletedAt.IsZero() {
		record.CompletedAt = time.Now()
	}
	store := a.historyStore
	if store == nil {
		if a.historyLoading && a.historyErr == "" {
			a.historyBacklog = append(a.historyBacklog, record)
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
			if a.historyStore != store {
				return
			}
			record.ID = id
			a.prependHistory(record)
		})
	})
}

func (a *app) flushHistoryBacklog() {
	if a.historyStore == nil || len(a.historyBacklog) == 0 {
		return
	}
	store := a.historyStore
	records := a.historyBacklog
	a.historyBacklog = nil
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
			if a.historyStore != store {
				return
			}
			for _, record := range added {
				a.prependHistory(record)
			}
		})
	})
}

func (a *app) prependHistory(record history.Record) {
	a.historyCount++
	a.historyBytes += record.FileSize
	a.historyItems = append(a.historyItems, history.Record{})
	copy(a.historyItems[1:], a.historyItems[:len(a.historyItems)-1])
	a.historyItems[0] = record
	if len(a.historyItems) > historyPageSize {
		a.historyItems = a.historyItems[:historyPageSize]
	}
}

func (a *app) confirmClearHistory() {
	if a.historyStore == nil || a.historyCount == 0 {
		return
	}
	win := a.win
	go func() {
		result, err := mygo.Dialog.Message(mygo.MessageOptions{
			Parent:        win,
			Type:          mygo.MessageWarning,
			Title:         "清空下载历史",
			Message:       "确定清空所有下载历史吗？",
			Detail:        "只删除记录，不会删除已下载的文件。",
			Buttons:       []string{"清空", "取消"},
			DefaultButton: 1,
			CancelButton:  1,
		})
		if err != nil || result.Button != 0 {
			return
		}
		a.clearHistory()
	}()
}

func (a *app) clearHistory() {
	store := a.historyStore
	if store == nil {
		return
	}
	a.dispatchDisk(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := store.Clear(ctx)
		a.update(func() {
			if a.historyStore != store {
				return
			}
			if err != nil {
				a.toast = "清空下载历史失败：" + friendlyError(err)
				return
			}
			a.historyItems = nil
			a.historyBacklog = nil
			a.historyCount = 0
			a.historyBytes = 0
			a.toast = "下载历史已清空"
		})
	})
}

func (a *app) historyView(c *ui.Context, col colors) {
	if a.historyLoading && a.historyStore == nil {
		softCard(c, col, func() {
			ui.Row(c).Gap(9).AlignItems(ui.Center).Children(func() {
				ui.Spinner(c).Size(15, 15)
				ui.Text(c, "正在加载下载历史…").FontSize(12).TextColor(col.muted)
			})
		})
		return
	}
	if a.historyErr != "" && len(a.historyItems) == 0 {
		softCard(c, col, func() {
			ui.Row(c).Gap(9).AlignItems(ui.Center).Children(func() {
				ui.Icon(c, iconAlert).Size(15, 15).TextColor(col.red)
				ui.Text(c, "下载历史不可用："+a.historyErr).FontSize(12).TextColor(col.red).
					Grow(1).MaxLines(3)
				secondaryButton(c, col, iconRefresh, "重试", a.retryHistory)
			})
		})
		return
	}
	if len(a.historyItems) == 0 {
		emptyState(c, col, iconClock, "还没有下载记录", "完成下载后，歌曲和文件信息会记录在这里。")
		return
	}

	ui.Column(c).Gap(10).Children(func() {
		if a.historyErr != "" {
			ui.Row(c).Gap(9).AlignItems(ui.Center).Padding(0, 4).Children(func() {
				ui.Icon(c, iconAlert).Size(14, 14).TextColor(col.red)
				ui.Text(c, a.historyErr).FontSize(11).TextColor(col.red).Grow(1).MaxLines(2)
				secondaryButton(c, col, iconRefresh, "重试", a.retryHistory)
			})
		}

		sizeText := formatFileSize(a.historyBytes)
		if sizeText == "" {
			sizeText = "0 B"
		}
		ui.Row(c).FillWidth().AlignItems(ui.Center).Padding(0, 4).Children(func() {
			ui.Textf(c, "共 %d 条记录 · 合计 %s", a.historyCount, sizeText).FontSize(12).
				TextColor(col.muted).Grow(1)
			ghostDangerButton(c, col, iconTrash, "清空历史", a.confirmClearHistory)
		})

		sheet(c, col, func() {
			for _, record := range a.historyItems {
				a.historyRow(c, col, record)
			}
		})

		if a.historyCount > len(a.historyItems) {
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
		ui.Text(c, formatFileSize(record.FileSize)).Width(70).FontSize(11).
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
