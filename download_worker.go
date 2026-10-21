package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"runtime/debug"
	"sync"
	"time"

	"github.com/rzh0504/qmdr/internal/history"
)

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
	// Each task carries its own cancel so one row can be cancelled without
	// touching the rest of the queue; the run-level ctx still cancels all.
	taskCtx, taskCancel := context.WithCancel(ctx)
	a.queue.taskMu.Lock()
	if a.queue.taskCancels == nil {
		a.queue.taskCancels = make(map[int]context.CancelFunc)
	}
	a.queue.taskCancels[task.ID] = taskCancel
	a.queue.taskMu.Unlock()
	defer func() {
		a.queue.taskMu.Lock()
		delete(a.queue.taskCancels, task.ID)
		a.queue.taskMu.Unlock()
		taskCancel()
	}()

	a.update(func() {
		if runID != a.queue.runID || a.closed.Load() {
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

	resolveCtx, cancelResolve := context.WithTimeout(taskCtx, 30*time.Second)
	resolved, err := a.api.ResolveURL(resolveCtx, task.Mid, task.Quality)
	cancelResolve()
	if err != nil {
		if ctx.Err() != nil || taskCtx.Err() != nil {
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

	written, err := a.api.Download(taskCtx, resolved.URL, dest, overwrite, func(received, size int64) {
		progress := 0.0
		if size > 0 {
			progress = min(1, float64(received)/float64(size))
		}
		a.update(func() {
			if runID != a.queue.runID {
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
		if ctx.Err() != nil || taskCtx.Err() != nil {
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
		if runID != a.queue.runID {
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
		if runID != a.queue.runID {
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
		if runID != a.queue.runID {
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
		if runID != a.queue.runID {
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
