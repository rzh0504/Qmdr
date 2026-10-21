package queue

import (
	"time"

	"github.com/rzh0504/qmdr/internal/history"
	"github.com/rzh0504/qmdr/internal/model"
)

// RecordFromItem converts a queue row to its stored history record.
func RecordFromItem(item model.QueueItem) history.QueueRecord {
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

// ItemFromRecord converts a stored record back to a queue row, putting an
// unknown status back to waiting.
func ItemFromRecord(record history.QueueRecord) model.QueueItem {
	status := model.QueueStatus(record.Status)
	if status < model.StatusWaiting || status > model.StatusCancelled {
		status = model.StatusWaiting
	}
	createdAt := record.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	return model.QueueItem{
		ID: record.ID,
		Song: model.Song{
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
