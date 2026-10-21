package main

import (
	"context"
	"log"
	"time"

	"github.com/rzh0504/qmdr/internal/history"
	"github.com/rzh0504/qmdr/internal/queue"
)

func (a *app) persistQueueItems(items []QueueItem) {
	if len(items) == 0 {
		return
	}
	store := a.history.store
	if store == nil {
		return
	}
	records := make([]history.QueueRecord, 0, len(items))
	for _, item := range items {
		records = append(records, queue.RecordFromItem(item))
	}
	a.dispatchDisk(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := store.SaveQueueItems(ctx, records); err != nil {
			log.Printf("persist download queue: %v", err)
		}
	})
}

func (a *app) persistQueueItem(item QueueItem) {
	a.persistQueueItems([]QueueItem{item})
}

func (a *app) deleteStoredQueueItems(ids []int) {
	if len(ids) == 0 {
		return
	}
	store := a.history.store
	if store == nil {
		return
	}
	a.dispatchDisk(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := store.DeleteQueueItems(ctx, ids); err != nil {
			log.Printf("delete download queue items: %v", err)
		}
	})
}

func (a *app) replaceStoredQueue() {
	store := a.history.store
	if store == nil {
		return
	}
	records := make([]history.QueueRecord, 0, len(a.queue.items))
	for _, item := range a.queue.items {
		records = append(records, queue.RecordFromItem(item))
	}
	a.dispatchDisk(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := store.ReplaceQueue(ctx, records); err != nil {
			log.Printf("replace download queue: %v", err)
		}
	})
}

func (a *app) restoreQueue(records []history.QueueRecord) {
	if a.history.store == nil {
		return
	}
	if len(a.queue.items) > 0 {
		a.replaceStoredQueue()
		return
	}
	if len(records) == 0 {
		return
	}

	maxID := 0
	var changed []QueueItem
	for _, record := range records {
		item := queue.ItemFromRecord(record)
		if item.Status == StatusDownloading {
			item.Status = StatusWaiting
			item.Progress = 0
			item.Error = ""
			changed = append(changed, item)
		}
		if item.ID > maxID {
			maxID = item.ID
		}
		a.queue.items = append(a.queue.items, item)
	}
	a.queue.seq = maxID
	if len(changed) > 0 {
		a.persistQueueItems(changed)
	}
}
