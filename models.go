package main

import "github.com/rzh0504/qmdr/internal/model"

type Song = model.Song
type Playlist = model.Playlist
type QueueStatus = model.QueueStatus
type QueueItem = model.QueueItem
type LogEntry = model.LogEntry

const (
	StatusWaiting     = model.StatusWaiting
	StatusDownloading = model.StatusDownloading
	StatusDone        = model.StatusDone
	StatusSkipped     = model.StatusSkipped
	StatusFailed      = model.StatusFailed
	StatusCancelled   = model.StatusCancelled
)
