package main

import (
	"context"
	"log"
	"time"

	"github.com/rzh0504/qmdr/internal/history"
)

func queueRecordFromItem(item QueueItem) history.QueueRecord {
	createdAt := item.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	return history.QueueRecord{
		ID:          item.ID,
		SongMID:     item.Song.Mid,
		Title:       item.Song.Title,
		Artist:      item.Song.Artist,
		Album:       item.Song.Album,
		AlbumMID:    item.Song.AlbumMid,
		Duration:    item.Song.Duration,
		SongQuality: item.Song.Quality,
		VIP:         item.Song.VIP,
		Status:      int(item.Status),
		Progress:    item.Progress,
		Quality:     item.Quality,
		Error:       item.Error,
		FilePath:    item.FilePath,
		FileSize:    item.FileSize,
		CreatedAt:   createdAt,
	}
}

func queueItemFromRecord(record history.QueueRecord) QueueItem {
	status := QueueStatus(record.Status)
	if status < StatusWaiting || status > StatusCancelled {
		status = StatusWaiting
	}
	createdAt := record.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	return QueueItem{
		ID: record.ID,
		Song: Song{
			Mid:      record.SongMID,
			Title:    record.Title,
			Artist:   record.Artist,
			Album:    record.Album,
			AlbumMid: record.AlbumMID,
			Duration: record.Duration,
			Quality:  record.SongQuality,
			VIP:      record.VIP,
		},
		Status:    status,
		Progress:  record.Progress,
		Quality:   record.Quality,
		Error:     record.Error,
		FilePath:  record.FilePath,
		FileSize:  record.FileSize,
		CreatedAt: createdAt,
	}
}

func (a *app) persistQueueItems(items []QueueItem) {
	if a.historyStore == nil || len(items) == 0 {
		return
	}
	records := make([]history.QueueRecord, 0, len(items))
	for _, item := range items {
		records = append(records, queueRecordFromItem(item))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := a.historyStore.SaveQueueItems(ctx, records); err != nil {
		log.Printf("persist download queue: %v", err)
	}
}

func (a *app) persistQueueItem(item QueueItem) {
	a.persistQueueItems([]QueueItem{item})
}

func (a *app) deleteStoredQueueItems(ids []int) {
	if a.historyStore == nil || len(ids) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := a.historyStore.DeleteQueueItems(ctx, ids); err != nil {
		log.Printf("delete download queue items: %v", err)
	}
}

func (a *app) replaceStoredQueue() {
	if a.historyStore == nil {
		return
	}
	records := make([]history.QueueRecord, 0, len(a.queue))
	for _, item := range a.queue {
		records = append(records, queueRecordFromItem(item))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := a.historyStore.ReplaceQueue(ctx, records); err != nil {
		log.Printf("replace download queue: %v", err)
	}
}

func (a *app) restoreQueue(records []history.QueueRecord) {
	if a.historyStore == nil {
		return
	}
	if len(a.queue) > 0 {
		a.replaceStoredQueue()
		return
	}
	if len(records) == 0 {
		return
	}

	maxID := 0
	var changed []QueueItem
	for _, record := range records {
		item := queueItemFromRecord(record)
		if item.Status == StatusDownloading {
			item.Status = StatusWaiting
			item.Progress = 0
			item.Error = ""
			changed = append(changed, item)
		}
		if item.ID > maxID {
			maxID = item.ID
		}
		a.queue = append(a.queue, item)
	}
	a.queueSeq = maxID
	if len(changed) > 0 {
		a.persistQueueItems(changed)
	}
}
