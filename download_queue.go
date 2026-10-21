package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/rzh0504/qmdr/internal/settings"
)

func (a *app) addToPending(songs ...Song) {
	if len(songs) == 0 {
		return
	}
	added, duplicates, invalid := 0, 0, 0
	for _, song := range songs {
		if strings.TrimSpace(song.Mid) == "" {
			invalid++
			continue
		}
		exists := false
		for _, pending := range a.queue.pending {
			if pending.Mid == song.Mid {
				exists = true
				break
			}
		}
		if exists {
			duplicates++
			continue
		}
		a.queue.pending = append(a.queue.pending, song)
		added++
	}
	switch {
	case added == 0 && invalid > 0:
		a.setToast("歌曲缺少 MID，无法加入待下载队列")
		return
	case added == 0 && duplicates > 0:
		a.setToast("歌曲已在待下载队列")
		return
	case added == 0:
		return
	}
	if added == 1 {
		a.setToast("已加入待下载队列")
	} else {
		a.setToast(fmt.Sprintf("已将 %d 首歌曲加入待下载队列", added))
	}
	a.savePending()
}

func (a *app) removePending(index int) {
	if index < 0 || index >= len(a.queue.pending) {
		return
	}
	a.queue.pending = append(a.queue.pending[:index], a.queue.pending[index+1:]...)
	a.savePending()
	a.refresh()
}

func (a *app) clearPending() {
	if len(a.queue.pending) == 0 {
		return
	}
	a.queue.pending = nil
	a.savePending()
	a.setToast("已清空待下载队列")
}

func (a *app) startPendingDownloads() {
	if len(a.queue.pending) == 0 {
		return
	}
	if a.queue.downloading {
		a.setToast("已有下载任务在运行，请先等待或取消")
		return
	}
	songs := append([]Song(nil), a.queue.pending...)
	a.queue.pending = nil
	if !a.enqueueSongs(songs) {
		a.queue.pending = songs
		a.refresh()
		return
	}
	a.savePending()
}

func (a *app) enqueueSongs(songs []Song) bool {
	if len(songs) == 0 {
		return false
	}
	if a.queue.downloading {
		a.setToast("已有下载任务在运行，请先等待或取消")
		a.view = viewQueue
		a.refresh()
		return false
	}
	for _, song := range songs {
		if strings.TrimSpace(song.Mid) == "" {
			a.setToast("歌曲缺少 MID，无法下载")
			return false
		}
	}

	quality := a.quality
	now := time.Now()
	first := len(a.queue.items)
	for _, song := range songs {
		a.queue.seq++
		a.queue.items = append(a.queue.items, QueueItem{
			ID:        a.queue.seq,
			Song:      song,
			Status:    StatusWaiting,
			Quality:   quality,
			CreatedAt: now,
		})
	}
	a.persistQueueItems(a.queue.items[first:])
	a.appendLog(fmt.Sprintf("已添加 %d 首歌曲到下载队列", len(songs)), "info")
	a.view = viewQueue
	a.refresh()
	a.startQueue()
	return true
}

func (a *app) startQueue() {
	if a.queue.downloading {
		return
	}
	var tasks []downloadTask
	for i := len(a.queue.items) - 1; i >= 0; i-- {
		item := a.queue.items[i]
		if item.Status != StatusWaiting {
			continue
		}
		if strings.TrimSpace(item.Song.Mid) == "" {
			continue
		}
		tasks = append(tasks, downloadTask{
			ID:       item.ID,
			Mid:      item.Song.Mid,
			Title:    item.Song.Title,
			Artist:   item.Song.Artist,
			Album:    item.Song.Album,
			AlbumMid: item.Song.AlbumMid,
			Quality:  item.Quality,
		})
	}
	if len(tasks) == 0 {
		a.setToast("没有待下载的任务")
		return
	}

	workers := int(a.concurrency)
	if workers < 1 {
		workers = 1
	}
	if workers > 8 {
		workers = 8
	}

	ctx, cancel := context.WithCancel(context.Background())
	a.queue.cancel = cancel
	a.queue.downloading = true
	a.queue.runID++
	runID := a.queue.runID
	dir := a.downloadDir
	overwrite := a.overwrite
	coverSize := settings.CoverSizeValue(a.coverSize)
	// A download that a previous run left unfinished leaves its .part file
	// behind; start each run with a clean directory.
	cleanPartFiles(dir)
	claims := newDownloadClaims()
	a.appendLog(fmt.Sprintf("开始处理下载队列（并发 %d）", workers), "info")
	a.refresh()

	go func() {
		a.runWorkers(ctx, runID, tasks, workers, dir, overwrite, coverSize, claims)
		a.update(func() {
			if runID != a.queue.runID {
				return
			}
			a.queue.downloading = false
			a.queue.cancel = nil
			if ctx.Err() != nil {
				a.appendLog("下载已取消", "warning")
				a.toast = "下载已取消"
			} else {
				a.appendLog("队列处理完成", "success")
				a.toast = "队列处理完成"
			}
		})
	}()
}

func (a *app) cancelQueue() {
	if !a.queue.downloading {
		a.setToast("当前没有正在下载的任务")
		return
	}
	if a.queue.cancel != nil {
		a.queue.cancel()
		a.setToast("正在取消下载…")
	}
}

// cancelTask stops one downloading item and leaves the queue running.
func (a *app) cancelTask(id int) {
	a.queue.taskMu.Lock()
	cancel := a.queue.taskCancels[id]
	a.queue.taskMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (a *app) clearCompleted() {
	out := a.queue.items[:0]
	removed := 0
	var removedIDs []int
	for _, item := range a.queue.items {
		if item.Status == StatusDone || item.Status == StatusSkipped {
			removed++
			removedIDs = append(removedIDs, item.ID)
			continue
		}
		out = append(out, item)
	}
	a.queue.items = out
	if removed == 0 {
		a.setToast("没有已完成的任务")
		return
	}
	a.deleteStoredQueueItems(removedIDs)
	a.appendLog(fmt.Sprintf("已清理 %d 个已完成任务", removed), "info")
	a.setToast(fmt.Sprintf("已清理 %d 个已完成任务", removed))
}

func (a *app) retryItem(id int) {
	if a.queue.downloading {
		a.setToast("请先等待当前任务结束")
		return
	}
	item := a.queueItem(id)
	if item == nil {
		return
	}
	item.Status = StatusWaiting
	item.Progress = 0
	item.Error = ""
	item.FileSize = 0
	a.persistQueueItem(*item)
	a.appendLog("重新加入队列："+item.Song.Title, "info")
	a.startQueue()
}

func (a *app) removeItem(id int) {
	if a.queue.downloading {
		a.setToast("请先取消当前下载")
		return
	}
	index := -1
	for i := range a.queue.items {
		if a.queue.items[i].ID == id {
			index = i
			break
		}
	}
	if index < 0 {
		return
	}
	a.queue.items = append(a.queue.items[:index], a.queue.items[index+1:]...)
	a.deleteStoredQueueItems([]int{id})
	a.refresh()
}

func (a *app) queueItem(id int) *QueueItem {
	for i := range a.queue.items {
		if a.queue.items[i].ID == id {
			return &a.queue.items[i]
		}
	}
	return nil
}

// filteredQueue returns the items of the selected filter newest first, so
// songs added later come first.
func (a *app) filteredQueue() []QueueItem {
	match := func(item QueueItem) bool {
		switch a.queue.filter {
		case 1:
			return item.Status == StatusWaiting || item.Status == StatusDownloading
		case 2:
			return item.Status == StatusDone || item.Status == StatusSkipped
		case 3:
			return item.Status == StatusFailed || item.Status == StatusCancelled
		default:
			return true
		}
	}
	out := make([]QueueItem, 0, len(a.queue.items))
	for i := len(a.queue.items) - 1; i >= 0; i-- {
		if match(a.queue.items[i]) {
			out = append(out, a.queue.items[i])
		}
	}
	return out
}

func (a *app) openDownloadDir() {
	dir := a.downloadDir
	if dir == "" {
		dir = settings.DefaultDownloadDir()
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		a.setToast("无法创建下载目录：" + err.Error())
		return
	}
	if err := mygo.Shell.OpenPath(dir); err != nil {
		a.setToast("打开下载目录失败：" + err.Error())
	}
}
