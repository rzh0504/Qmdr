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
		id, outcome, removed, err := store.SaveDownload(ctx, record)
		if err != nil {
			log.Printf("record download history: %v", err)
			return
		}
		stats, statsErr := store.Stats(ctx)
		a.update(func() {
			if a.history.store != store {
				return
			}
			record.ID = id
			for _, removedID := range removed {
				a.removeHistoryRecord(removedID)
			}
			switch outcome {
			case history.DownloadUpdated, history.DownloadInserted:
				a.prependHistoryItem(record)
			}
			if statsErr == nil {
				a.history.count = stats.Count
				a.history.bytes = stats.Bytes
			}
			a.syncTagStudioHistory(record, outcome, removed)
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
		for _, record := range records {
			if _, _, _, err := store.SaveDownload(ctx, record); err != nil {
				log.Printf("record download history: %v", err)
			}
		}
		recent, recentErr := store.Recent(ctx, historyPageSize)
		stats, statsErr := store.Stats(ctx)
		a.update(func() {
			if a.history.store != store {
				return
			}
			if recentErr == nil {
				a.history.items = recent
			}
			if statsErr == nil {
				a.history.count = stats.Count
				a.history.bytes = stats.Bytes
			}
		})
	})
}

// removeHistoryRecord drops one row from the in-memory list; the totals
// come from the store's stats after the write.
func (a *app) removeHistoryRecord(id int64) {
	for i := range a.history.items {
		if a.history.items[i].ID != id {
			continue
		}
		a.history.items = append(a.history.items[:i], a.history.items[i+1:]...)
		delete(a.history.selectedIDs, id)
		if a.history.selecting && a.historySelectedCount() == 0 {
			a.resetHistorySelection()
		}
		return
	}
}

func (a *app) prependHistoryItem(record history.Record) {
	a.removeHistoryRecord(record.ID)
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
				if a.tagStudio.dirty && a.tagStudio.source == "history" {
					ui.Text(c, "Tag Studio 中未保存的标签修改也会被丢弃。").FontSize(12).
						TextColor(col.gold).LineHeight(1.35)
				}
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
			a.history.deleteIDs = nil
			a.resetHistorySelection()
			a.clearTagHistoryEntries()
			a.toast = "下载历史已清空"
		})
	})
}

func (a *app) historyView(c *ui.Context, col colors) {
	if a.history.loading && a.history.store == nil {
		loadingState(c, col, "正在加载下载历史…")
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
		ui.Row(c).FillWidth().Height(30).AlignItems(ui.Center).Padding(0, 4).Children(func() {
			if a.history.selecting {
				ui.Textf(c, "已选 %d / %d 条", a.historySelectedCount(), len(a.history.items)).
					FontSize(12).TextColor(col.accent).Grow(1)
				return
			}
			ui.Textf(c, "共 %d 条记录 · 合计 %s", a.history.count, sizeText).FontSize(12).
				TextColor(col.muted).Grow(1)
			ghostDangerButton(c, col, iconTrash, "清空历史", a.confirmClearHistory)
		})

		sheet(c, col, func() {
			for i, record := range a.history.items {
				a.historyRow(c, col, i, record)
			}
		})

		if a.history.count > len(a.history.items) {
			ui.Textf(c, "仅显示最近 %d 条记录", historyPageSize).FontSize(11).
				TextColor(col.faint).Padding(0, 4)
		}
	})
}

func (a *app) historyRow(c *ui.Context, col colors, index int, record history.Record) {
	s := &a.history
	selected := s.selecting && s.selectedIDs[record.ID]
	row := ui.Row(c).Gap(10).Padding(7, 10).Height(52).Radius(8).AlignItems(ui.Center)
	switch {
	case s.selecting && selected:
		fill := col.accent.Alpha(0.06)
		if col.dark {
			fill = col.accent.Alpha(0.12)
		}
		row.Background(fill).Cursor(ui.CursorPointer)
	case s.selecting:
		row.Cursor(ui.CursorPointer)
		if row.Hovered() {
			row.Background(col.rowSelected)
			row = chosen(row, col)
		}
	case row.Hovered():
		row.Background(col.rowSelected)
		row = chosen(row, col)
	}
	if !s.selecting {
		a.trackHistoryPress(row, record, func() { a.beginHistorySelectionWith(record) })
	}
	if s.selecting && row.Clicked() {
		if s.suppressClick && s.pressID == record.ID {
			s.suppressClick = false
			s.pressID = 0
		} else {
			s.suppressClick = false
			s.pressID = 0
			a.toggleHistorySelection(record)
		}
	}
	row.Children(func() {
		ui.Row(c).Width(24).Center().Children(func() {
			if s.selecting {
				songSelectionMark(c, col, selected)
				return
			}
			ui.Textf(c, "%02d", index+1).FontSize(11).TextColor(col.faint).SingleLine()
		})
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
		if !s.selecting {
			iconButton(c, col, iconFolder, "在文件夹中显示 "+record.Title, func() {
				a.showHistoryFile(record.FilePath)
			})
		}
	})
	a.openHistoryMenu(row, record)
}

func (a *app) trackHistoryPress(row ui.Element, record history.Record, onLongPress func()) {
	if record.ID == 0 {
		return
	}
	if row.Pressed() {
		if a.history.pressID == record.ID {
			return
		}
		a.stopHistoryPress()
		a.history.pressID = record.ID
		a.history.pressSeq++
		seq := a.history.pressSeq
		a.history.suppressClick = false
		a.history.pressTimer = time.AfterFunc(songLongPressDelay, func() {
			a.update(func() {
				if a.history.pressID != record.ID || a.history.pressSeq != seq {
					return
				}
				a.history.pressTimer = nil
				a.history.suppressClick = true
				onLongPress()
			})
		})
		return
	}
	if a.history.pressID == record.ID {
		a.stopHistoryPress()
	}
}

func (a *app) stopHistoryPress() {
	if a.history.pressTimer != nil {
		a.history.pressTimer.Stop()
		a.history.pressTimer = nil
	}
	a.history.pressID = 0
}

func (a *app) beginHistorySelectionWith(record history.Record) {
	if record.ID == 0 {
		return
	}
	if !a.history.selecting {
		a.history.selecting = true
		a.history.selectedIDs = make(map[int64]bool)
	}
	a.history.selectedIDs[record.ID] = true
	a.refresh()
}

func (a *app) endHistorySelection() {
	a.resetHistorySelection()
	a.refresh()
}

func (a *app) resetHistorySelection() {
	a.stopHistoryPress()
	a.history.selecting = false
	a.history.selectedIDs = nil
	a.history.suppressClick = false
}

func (a *app) toggleHistorySelection(record history.Record) {
	if !a.history.selecting || record.ID == 0 {
		return
	}
	if a.history.selectedIDs[record.ID] {
		delete(a.history.selectedIDs, record.ID)
		if a.historySelectedCount() == 0 {
			a.resetHistorySelection()
		}
	} else {
		a.history.selectedIDs[record.ID] = true
	}
	a.refresh()
}

func (a *app) toggleAllHistory() {
	if a.allHistorySelected() {
		a.resetHistorySelection()
		a.refresh()
		return
	}
	next := make(map[int64]bool, len(a.history.items))
	for _, record := range a.history.items {
		if record.ID != 0 {
			next[record.ID] = true
		}
	}
	a.history.selecting = true
	a.history.selectedIDs = next
	if len(next) == 0 {
		a.resetHistorySelection()
	}
	a.refresh()
}

func (a *app) allHistorySelected() bool {
	if len(a.history.items) == 0 {
		return false
	}
	for _, record := range a.history.items {
		if !a.history.selectedIDs[record.ID] {
			return false
		}
	}
	return true
}

func (a *app) historySelectedCount() int {
	count := 0
	for _, record := range a.history.items {
		if a.history.selectedIDs[record.ID] {
			count++
		}
	}
	return count
}

func (a *app) historySelectedIDs() []int64 {
	ids := make([]int64, 0, a.historySelectedCount())
	for _, record := range a.history.items {
		if a.history.selectedIDs[record.ID] {
			ids = append(ids, record.ID)
		}
	}
	return ids
}

func (a *app) confirmDeleteHistory(ids []int64) {
	if len(ids) == 0 || a.history.store == nil {
		return
	}
	a.history.deleteIDs = append([]int64(nil), ids...)
	a.history.confirmDelete = true
	a.refresh()
}

func (a *app) deleteHistoryModal(c *ui.Context, col colors) {
	a.modal(c, col, &a.history.confirmDelete, "删除下载记录", 380, func(close func()) {
		ids := a.history.deleteIDs
		title := "确定删除这条下载记录吗？"
		if len(ids) == 1 {
			for _, record := range a.history.items {
				if record.ID == ids[0] {
					title = fmt.Sprintf("确定删除「%s」的下载记录吗？", record.Title)
					break
				}
			}
		} else if len(ids) > 1 {
			title = fmt.Sprintf("确定删除选中的 %d 条下载记录吗？", len(ids))
		}
		ui.Row(c).Gap(10).AlignItems(ui.Start).Children(func() {
			ui.Icon(c, iconAlert).Size(20, 20).TextColor(col.gold)
			ui.Column(c).Gap(4).Grow(1).MinWidth(0).Children(func() {
				ui.Text(c, title).FontSize(14).TextColor(col.text).LineHeight(1.35)
				ui.Text(c, "只删除记录，不会删除已下载的文件。").FontSize(12).
					TextColor(col.muted).LineHeight(1.35)
				if a.tagStudioDraftLostByHistoryDelete(ids) {
					ui.Text(c, "Tag Studio 中这首歌未保存的修改也会被丢弃。").FontSize(12).
						TextColor(col.gold).LineHeight(1.35)
				}
			})
		})
		ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
			secondaryButton(c, col, nil, "取消", close)
			dangerButton(c, col, "删除", func() {
				ids := a.history.deleteIDs
				close()
				a.deleteHistoryRecords(ids)
			})
		})
	})
}

func (a *app) deleteHistoryRecords(ids []int64) {
	store := a.history.store
	if store == nil || len(ids) == 0 {
		return
	}
	a.dispatchDisk(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		changes, err := store.Delete(ctx, ids)
		if err != nil {
			a.update(func() { a.toast = "删除下载记录失败：" + friendlyError(err) })
			return
		}
		recent, recentErr := store.Recent(ctx, historyPageSize)
		stats, statsErr := store.Stats(ctx)
		a.update(func() {
			if a.history.store != store {
				return
			}
			for _, id := range ids {
				a.removeTagHistoryEntryByID(id)
			}
			if recentErr == nil {
				a.history.items = recent
			}
			if statsErr == nil {
				a.history.count = stats.Count
				a.history.bytes = stats.Bytes
			}
			a.resetHistorySelection()
			a.history.deleteIDs = nil
			if changes > 0 {
				a.toast = fmt.Sprintf("已删除 %d 条下载记录", changes)
			}
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
