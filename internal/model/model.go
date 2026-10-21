package model

import "time"

// Song is a song the UI and the download queue share.
type Song struct {
	Mid      string `json:"mid"`
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	Album    string `json:"album"`
	AlbumMid string `json:"album_mid"`
	Duration string `json:"duration"`
	Quality  string `json:"quality"`
	VIP      bool   `json:"vip"`
}

// Playlist is a user's QQ Music playlist.
type Playlist struct {
	DirID int64  `json:"dir_id"`
	TID   int64  `json:"tid"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// QueueStatus is a download queue item's state.
type QueueStatus int

const (
	StatusWaiting QueueStatus = iota
	StatusDownloading
	StatusDone
	StatusSkipped
	StatusFailed
	StatusCancelled
)

func (s QueueStatus) Label() string {
	switch s {
	case StatusWaiting:
		return "等待中"
	case StatusDownloading:
		return "下载中"
	case StatusDone:
		return "已完成"
	case StatusSkipped:
		return "已跳过"
	case StatusFailed:
		return "失败"
	case StatusCancelled:
		return "已取消"
	default:
		return "未知"
	}
}

// QueueItem is one row of the download queue.
type QueueItem struct {
	ID        int
	Song      Song
	Status    QueueStatus
	Progress  float64
	Quality   string
	Error     string
	FilePath  string
	FileSize  int64
	CreatedAt time.Time
}

// LogEntry is one task log line.
type LogEntry struct {
	At      time.Time
	Level   string
	Message string
}
