package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo"
)

type downloadTask struct {
	ID      int
	Mid     string
	Title   string
	Artist  string
	Quality string
}

func (a *app) enqueueSongs(songs []Song) {
	if len(songs) == 0 {
		return
	}
	if a.downloading {
		a.setToast("已有下载任务在运行，请先等待或取消")
		a.view = viewQueue
		a.refresh()
		return
	}
	for _, song := range songs {
		if strings.TrimSpace(song.Mid) == "" {
			a.setToast("歌单下载接口待接入，当前请使用搜索下载")
			return
		}
	}

	quality := a.quality
	for _, song := range songs {
		a.queueSeq++
		a.queue = append(a.queue, QueueItem{
			ID:      a.queueSeq,
			Song:    song,
			Status:  StatusWaiting,
			Quality: quality,
		})
	}
	a.appendLog(fmt.Sprintf("已添加 %d 首歌曲到下载队列", len(songs)), "info")
	a.view = viewQueue
	a.recalcOverallProgress()
	a.refresh()
	a.startQueue()
}

func (a *app) startQueue() {
	if a.downloading {
		return
	}
	var tasks []downloadTask
	for _, item := range a.queue {
		if item.Status != StatusWaiting {
			continue
		}
		if strings.TrimSpace(item.Song.Mid) == "" {
			continue
		}
		tasks = append(tasks, downloadTask{
			ID:      item.ID,
			Mid:     item.Song.Mid,
			Title:   item.Song.Title,
			Artist:  item.Song.Artist,
			Quality: item.Quality,
		})
	}
	if len(tasks) == 0 {
		a.setToast("没有待下载的任务")
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel
	a.downloading = true
	a.runID++
	runID := a.runID
	dir := a.downloadDir
	overwrite := a.overwrite
	a.appendLog("开始处理下载队列", "info")
	a.refresh()
	go a.downloadLoop(ctx, runID, tasks, dir, overwrite)
}

func (a *app) downloadLoop(ctx context.Context, runID int, tasks []downloadTask, dir string, overwrite bool) {
	total := len(tasks)
	for index, task := range tasks {
		if ctx.Err() != nil {
			a.update(func() {
				if runID != a.runID {
					return
				}
				a.downloading = false
				a.cancel = nil
				a.currentTaskID = 0
				a.appendLog("下载已取消", "warning")
				a.recalcOverallProgress()
				a.toast = "下载已取消"
			})
			return
		}

		a.update(func() {
			if runID != a.runID || a.closed {
				return
			}
			item := a.queueItem(task.ID)
			if item == nil {
				return
			}
			item.Status = StatusDownloading
			item.Progress = 0
			item.Speed = 0
			item.Error = ""
			a.currentTaskID = task.ID
			a.appendLog("开始下载："+task.Title, "info")
			a.recalcOverallProgress()
		})

		resolveCtx, cancelResolve := context.WithTimeout(ctx, 30*time.Second)
		resolved, err := a.api.ResolveURL(resolveCtx, task.Mid, task.Quality)
		cancelResolve()
		if err != nil {
			if ctx.Err() != nil {
				a.finishCancelled(runID, task.ID)
				return
			}
			a.finishFailed(runID, task.ID, task.Title, err, index == total-1)
			continue
		}

		name := sanitizeFilename(task.Artist + " - " + task.Title)
		dest, err := safeDownloadPath(dir, name+resolved.Ext)
		if err != nil {
			a.finishFailed(runID, task.ID, task.Title, err, index == total-1)
			continue
		}

		if _, statErr := os.Stat(dest); statErr == nil && !overwrite {
			a.update(func() {
				if runID != a.runID {
					return
				}
				item := a.queueItem(task.ID)
				if item == nil {
					return
				}
				item.Status = StatusSkipped
				item.Progress = 1
				item.Speed = 0
				item.Quality = resolved.Quality
				item.FilePath = dest
				a.appendLog("文件已存在，跳过："+task.Title, "muted")
				a.currentTaskID = 0
				a.recalcOverallProgress()
				if index == total-1 {
					a.finishQueueRun()
				}
			})
			continue
		}

		started := time.Now()
		_, err = a.api.Download(ctx, resolved.URL, dest, overwrite, func(received, size int64) {
			progress := 0.0
			if size > 0 {
				progress = min(1, float64(received)/float64(size))
			}
			elapsed := time.Since(started).Seconds()
			speed := 0.0
			if elapsed > 0 {
				speed = float64(received) / (1 << 20) / elapsed
			}
			a.update(func() {
				if runID != a.runID {
					return
				}
				item := a.queueItem(task.ID)
				if item == nil {
					return
				}
				item.Progress = progress
				item.Speed = speed
				a.recalcOverallProgress()
			})
		})
		if err != nil {
			if ctx.Err() != nil {
				a.finishCancelled(runID, task.ID)
				return
			}
			a.finishFailed(runID, task.ID, task.Title, err, index == total-1)
			continue
		}

		a.update(func() {
			if runID != a.runID {
				return
			}
			item := a.queueItem(task.ID)
			if item == nil {
				return
			}
			item.Status = StatusDone
			item.Progress = 1
			item.Speed = 0
			item.Quality = resolved.Quality
			item.FilePath = dest
			a.appendLog("下载成功："+task.Title+"（"+resolved.Quality+"）", "success")
			a.currentTaskID = 0
			a.recalcOverallProgress()
			if index == total-1 {
				a.finishQueueRun()
			}
		})
	}
}

func (a *app) finishCancelled(runID, taskID int) {
	a.update(func() {
		if runID != a.runID {
			return
		}
		if item := a.queueItem(taskID); item != nil {
			item.Status = StatusCancelled
			item.Speed = 0
		}
		a.currentTaskID = 0
		a.downloading = false
		a.cancel = nil
		a.appendLog("下载已取消", "warning")
		a.recalcOverallProgress()
		a.toast = "下载已取消"
	})
}

func (a *app) finishFailed(runID, taskID int, title string, err error, last bool) {
	message := friendlyError(err)
	a.update(func() {
		if runID != a.runID {
			return
		}
		if item := a.queueItem(taskID); item != nil {
			item.Status = StatusFailed
			item.Speed = 0
			item.Error = message
		}
		a.appendLog("下载失败："+title+"（"+message+"）", "error")
		a.currentTaskID = 0
		a.recalcOverallProgress()
		if last {
			a.finishQueueRun()
		}
	})
}

func (a *app) finishQueueRun() {
	a.downloading = false
	a.cancel = nil
	a.currentTaskID = 0
	a.appendLog("队列处理完成", "success")
	a.toast = "队列处理完成"
}

func (a *app) cancelQueue() {
	if !a.downloading {
		a.setToast("当前没有正在下载的任务")
		return
	}
	if a.cancel != nil {
		a.cancel()
		a.setToast("正在取消下载…")
	}
}

func (a *app) clearCompleted() {
	out := a.queue[:0]
	removed := 0
	for _, item := range a.queue {
		if item.Status == StatusDone || item.Status == StatusSkipped {
			removed++
			continue
		}
		out = append(out, item)
	}
	a.queue = out
	if removed == 0 {
		a.setToast("没有已完成的任务")
		return
	}
	a.recalcOverallProgress()
	a.appendLog(fmt.Sprintf("已清理 %d 个已完成任务", removed), "info")
	a.setToast(fmt.Sprintf("已清理 %d 个已完成任务", removed))
}

func (a *app) retryItem(id int) {
	if a.downloading {
		a.setToast("请先等待当前任务结束")
		return
	}
	item := a.queueItem(id)
	if item == nil {
		return
	}
	item.Status = StatusWaiting
	item.Progress = 0
	item.Speed = 0
	item.Error = ""
	a.recalcOverallProgress()
	a.appendLog("重新加入队列："+item.Song.Title, "info")
	a.startQueue()
}

func (a *app) removeItem(id int) {
	index := -1
	for i := range a.queue {
		if a.queue[i].ID == id {
			index = i
			break
		}
	}
	if index < 0 {
		return
	}
	if a.queue[index].Status == StatusDownloading {
		a.setToast("请先取消当前任务")
		return
	}
	a.queue = append(a.queue[:index], a.queue[index+1:]...)
	a.recalcOverallProgress()
	a.refresh()
}

func (a *app) queueItem(id int) *QueueItem {
	for i := range a.queue {
		if a.queue[i].ID == id {
			return &a.queue[i]
		}
	}
	return nil
}

func (a *app) currentQueueItem() *QueueItem {
	if item := a.queueItem(a.currentTaskID); item != nil {
		return item
	}
	for i := range a.queue {
		if a.queue[i].Status == StatusDownloading {
			return &a.queue[i]
		}
	}
	return nil
}

func (a *app) filteredQueue() []QueueItem {
	switch a.queueFilter {
	case 1:
		var out []QueueItem
		for _, item := range a.queue {
			if item.Status == StatusWaiting || item.Status == StatusDownloading {
				out = append(out, item)
			}
		}
		return out
	case 2:
		var out []QueueItem
		for _, item := range a.queue {
			if item.Status == StatusDone || item.Status == StatusSkipped {
				out = append(out, item)
			}
		}
		return out
	case 3:
		var out []QueueItem
		for _, item := range a.queue {
			if item.Status == StatusFailed || item.Status == StatusCancelled {
				out = append(out, item)
			}
		}
		return out
	default:
		return a.queue
	}
}

func (a *app) openDownloadDir() {
	dir := a.downloadDir
	if dir == "" {
		dir = defaultDownloadDir()
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		a.setToast("无法创建下载目录：" + err.Error())
		return
	}
	if err := mygo.Shell.OpenPath(dir); err != nil {
		a.setToast("打开下载目录失败：" + err.Error())
	}
}

func sanitizeFilename(name string) string {
	name = strings.Map(func(r rune) rune {
		switch r {
		case '\\', '/', ':', '*', '?', '"', '<', '>', '|':
			return '_'
		}
		if r < 32 {
			return -1
		}
		return r
	}, name)
	name = strings.TrimSpace(strings.Trim(name, "."))
	if name == "" {
		name = "未知歌曲"
	}
	base := strings.ToUpper(strings.TrimSuffix(name, filepath.Ext(name)))
	switch base {
	case "CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
		name = "_" + name
	}
	runes := []rune(name)
	const maxRunes = 160
	if len(runes) > maxRunes {
		name = string(runes[:maxRunes])
	}
	return name
}

func safeDownloadPath(base, name string) (string, error) {
	if strings.TrimSpace(base) == "" {
		return "", errors.New("未设置下载目录")
	}
	absBase, err := filepath.Abs(base)
	if err != nil {
		return "", err
	}
	dest := filepath.Join(absBase, name)
	rel, err := filepath.Rel(absBase, dest)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", errors.New("无效的下载路径")
	}
	return dest, nil
}
