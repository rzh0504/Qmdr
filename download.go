package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"github.com/rzh0504/qmdr/internal/history"
	"github.com/rzh0504/qmdr/internal/tag"
)

type downloadTask struct {
	ID       int
	Mid      string
	Title    string
	Artist   string
	Album    string
	AlbumMid string
	Quality  string
}

type claimResult int

const (
	claimOK claimResult = iota
	claimSkip
	claimExhausted
)

type downloadClaims struct {
	mu    sync.Mutex
	paths map[string]bool
}

func newDownloadClaims() *downloadClaims {
	return &downloadClaims{paths: make(map[string]bool)}
}

func (c *downloadClaims) reserve(dir, base, ext string, overwrite bool) (string, claimResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i := 1; i <= 999; i++ {
		name := base + ext
		if i > 1 {
			name = fmt.Sprintf("%s (%d)%s", base, i, ext)
		}
		dest, err := safeDownloadPath(dir, name)
		if err != nil {
			return "", claimExhausted
		}
		if _, err := os.Stat(dest); err == nil && !overwrite {
			return dest, claimSkip
		}
		key := strings.ToLower(dest)
		if c.paths[key] {
			continue
		}
		c.paths[key] = true
		return dest, claimOK
	}
	return "", claimExhausted
}

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
		for _, pending := range a.pending {
			if pending.Mid == song.Mid {
				exists = true
				break
			}
		}
		if exists {
			duplicates++
			continue
		}
		a.pending = append(a.pending, song)
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
	if err := a.savePending(); err != nil {
		a.setToast("待下载队列保存失败：" + friendlyError(err))
		return
	}
	if added == 1 {
		a.setToast("已加入待下载队列")
	} else {
		a.setToast(fmt.Sprintf("已将 %d 首歌曲加入待下载队列", added))
	}
}

func (a *app) removePending(index int) {
	if index < 0 || index >= len(a.pending) {
		return
	}
	a.pending = append(a.pending[:index], a.pending[index+1:]...)
	if err := a.savePending(); err != nil {
		a.setToast("待下载队列保存失败：" + friendlyError(err))
		return
	}
	a.refresh()
}

func (a *app) clearPending() {
	if len(a.pending) == 0 {
		return
	}
	a.pending = nil
	if err := a.savePending(); err != nil {
		a.setToast("待下载队列保存失败：" + friendlyError(err))
		return
	}
	a.setToast("已清空待下载队列")
}

func (a *app) startPendingDownloads() {
	if len(a.pending) == 0 {
		return
	}
	if a.downloading {
		a.setToast("已有下载任务在运行，请先等待或取消")
		return
	}
	songs := append([]Song(nil), a.pending...)
	a.pending = nil
	if !a.enqueueSongs(songs) {
		a.pending = songs
		a.refresh()
		return
	}
	if err := a.savePending(); err != nil {
		a.setToast("待下载队列保存失败：" + friendlyError(err))
	}
}

func (a *app) enqueueSongs(songs []Song) bool {
	if len(songs) == 0 {
		return false
	}
	if a.downloading {
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
	first := len(a.queue)
	for _, song := range songs {
		a.queueSeq++
		a.queue = append(a.queue, QueueItem{
			ID:        a.queueSeq,
			Song:      song,
			Status:    StatusWaiting,
			Quality:   quality,
			CreatedAt: now,
		})
	}
	a.persistQueueItems(a.queue[first:])
	a.appendLog(fmt.Sprintf("已添加 %d 首歌曲到下载队列", len(songs)), "info")
	a.view = viewQueue
	a.refresh()
	a.startQueue()
	return true
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
	a.cancel = cancel
	a.downloading = true
	a.runID++
	runID := a.runID
	dir := a.downloadDir
	overwrite := a.overwrite
	coverSize := coverSizeValue(a.coverSize)
	claims := newDownloadClaims()
	a.appendLog(fmt.Sprintf("开始处理下载队列（并发 %d）", workers), "info")
	a.refresh()

	go func() {
		a.runWorkers(ctx, runID, tasks, workers, dir, overwrite, coverSize, claims)
		a.update(func() {
			if runID != a.runID {
				return
			}
			a.downloading = false
			a.cancel = nil
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

func (a *app) runWorkers(ctx context.Context, runID int, tasks []downloadTask, workers int, dir string, overwrite bool, coverSize int, claims *downloadClaims) {
	jobs := make(chan downloadTask)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range jobs {
				if ctx.Err() != nil {
					return
				}
				a.processDownloadSafely(ctx, runID, task, dir, overwrite, coverSize, claims)
			}
		}()
	}

feed:
	for _, task := range tasks {
		select {
		case <-ctx.Done():
			break feed
		case jobs <- task:
		}
	}
	close(jobs)
	wg.Wait()
}

func (a *app) processDownloadSafely(ctx context.Context, runID int, task downloadTask, dir string, overwrite bool, coverSize int, claims *downloadClaims) {
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Printf("download task panic: %v\n%s", recovered, debug.Stack())
			a.markFailed(runID, task, fmt.Errorf("下载任务异常：%v", recovered))
		}
	}()
	a.processDownload(ctx, runID, task, dir, overwrite, coverSize, claims)
}

func (a *app) processDownload(ctx context.Context, runID int, task downloadTask, dir string, overwrite bool, coverSize int, claims *downloadClaims) {
	a.update(func() {
		if runID != a.runID || a.closed.Load() {
			return
		}
		item := a.queueItem(task.ID)
		if item == nil {
			return
		}
		item.Status = StatusDownloading
		item.Progress = 0
		item.Error = ""
		a.persistQueueItem(*item)
		a.appendLog("开始下载："+task.Title, "info")
	})

	resolveCtx, cancelResolve := context.WithTimeout(ctx, 30*time.Second)
	resolved, err := a.api.ResolveURL(resolveCtx, task.Mid, task.Quality)
	cancelResolve()
	if err != nil {
		if ctx.Err() != nil {
			a.markCancelled(runID, task)
			return
		}
		a.markFailed(runID, task, err)
		return
	}

	base := sanitizeFilename(task.Artist + " - " + task.Title)
	dest, claim := claims.reserve(dir, base, resolved.Ext, overwrite)
	switch claim {
	case claimSkip:
		a.markSkipped(runID, task, dest, resolved.Quality)
		return
	case claimExhausted:
		a.markFailed(runID, task, errors.New("找不到可用的目标文件名"))
		return
	}

	written, err := a.api.Download(ctx, resolved.URL, dest, overwrite, func(received, size int64) {
		progress := 0.0
		if size > 0 {
			progress = min(1, float64(received)/float64(size))
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
		})
	})
	if err != nil {
		if ctx.Err() != nil {
			a.markCancelled(runID, task)
			return
		}
		a.markFailed(runID, task, err)
		return
	}

	metaCtx, metaCancel := context.WithTimeout(context.Background(), 30*time.Second)
	metaErr := a.writeMetadataSafely(metaCtx, task, dest, coverSize)
	metaCancel()

	size := fileSize(dest)
	if size == 0 {
		size = written
	}
	record := history.Record{
		SongMID:     task.Mid,
		Title:       task.Title,
		Artist:      task.Artist,
		Album:       task.Album,
		AlbumMID:    task.AlbumMid,
		Quality:     resolved.Quality,
		FilePath:    dest,
		FileSize:    size,
		Status:      history.StatusDone,
		CompletedAt: time.Now(),
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
		item.Quality = resolved.Quality
		item.FilePath = dest
		item.FileSize = size
		a.persistQueueItem(*item)
		a.recordHistory(record)
		a.appendLog("下载成功："+task.Title+"（"+resolved.Quality+"）", "success")
		if metaErr != nil {
			a.appendLog("元数据写入失败："+task.Title+"（"+friendlyError(metaErr)+"）", "warning")
		}
	})
}

func (a *app) markCancelled(runID int, task downloadTask) {
	a.update(func() {
		if runID != a.runID {
			return
		}
		if item := a.queueItem(task.ID); item != nil {
			item.Status = StatusCancelled
			a.persistQueueItem(*item)
		}
		a.appendLog("已取消："+task.Title, "warning")
	})
}

func (a *app) markFailed(runID int, task downloadTask, err error) {
	message := friendlyError(err)
	a.update(func() {
		if runID != a.runID {
			return
		}
		if item := a.queueItem(task.ID); item != nil {
			item.Status = StatusFailed
			item.Error = message
			a.persistQueueItem(*item)
		}
		a.appendLog("下载失败："+task.Title+"（"+message+"）", "error")
	})
}

func (a *app) markSkipped(runID int, task downloadTask, dest, quality string) {
	size := fileSize(dest)
	record := history.Record{
		SongMID:     task.Mid,
		Title:       task.Title,
		Artist:      task.Artist,
		Album:       task.Album,
		AlbumMID:    task.AlbumMid,
		Quality:     quality,
		FilePath:    dest,
		FileSize:    size,
		Status:      history.StatusSkipped,
		CompletedAt: time.Now(),
	}
	a.update(func() {
		if runID != a.runID {
			return
		}
		if item := a.queueItem(task.ID); item != nil {
			item.Status = StatusSkipped
			item.Progress = 1
			item.Quality = quality
			item.FilePath = dest
			item.FileSize = size
			a.persistQueueItem(*item)
		}
		a.recordHistory(record)
		a.appendLog("文件已存在，跳过："+task.Title, "muted")
	})
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
	var removedIDs []int
	for _, item := range a.queue {
		if item.Status == StatusDone || item.Status == StatusSkipped {
			removed++
			removedIDs = append(removedIDs, item.ID)
			continue
		}
		out = append(out, item)
	}
	a.queue = out
	if removed == 0 {
		a.setToast("没有已完成的任务")
		return
	}
	a.deleteStoredQueueItems(removedIDs)
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
	item.Error = ""
	item.FileSize = 0
	a.persistQueueItem(*item)
	a.appendLog("重新加入队列："+item.Song.Title, "info")
	a.startQueue()
}

func (a *app) removeItem(id int) {
	if a.downloading {
		a.setToast("请先取消当前下载")
		return
	}
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
	a.queue = append(a.queue[:index], a.queue[index+1:]...)
	a.deleteStoredQueueItems([]int{id})
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

func (a *app) writeMetadataSafely(ctx context.Context, task downloadTask, dest string, coverSize int) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Printf("metadata panic: %v\n%s", recovered, debug.Stack())
			err = fmt.Errorf("元数据写入异常")
		}
	}()
	return a.writeMetadata(ctx, task, dest, coverSize)
}

func (a *app) writeMetadata(ctx context.Context, task downloadTask, dest string, coverSize int) error {
	lyrics, _ := a.api.Lyric(ctx, task.Mid)
	var cover []byte
	var coverMIME string
	if task.AlbumMid != "" {
		cover, coverMIME, _ = a.api.Cover(ctx, task.AlbumMid, coverSize)
	}
	return tag.Write(dest, tag.Info{
		Title:     task.Title,
		Artist:    task.Artist,
		Album:     task.Album,
		Lyrics:    lyrics,
		Cover:     cover,
		CoverMIME: coverMIME,
	})
}

func coverSizeValue(value string) int {
	value = strings.TrimSpace(strings.TrimSuffix(strings.ToLower(value), "px"))
	size, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 800
	}
	switch size {
	case 150, 300, 500, 800:
		return size
	default:
		return 800
	}
}

func fileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

func formatFileSize(size int64) string {
	switch {
	case size <= 0:
		return ""
	case size >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(size)/float64(1<<30))
	case size >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(size)/float64(1<<20))
	case size >= 1<<10:
		return fmt.Sprintf("%.0f KB", float64(size)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", size)
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
