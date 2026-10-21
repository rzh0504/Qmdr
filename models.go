package main

type Song struct {
	Mid      string
	Title    string
	Artist   string
	Album    string
	AlbumMid string
	Duration string
	Quality  string
	VIP      bool
}

type Playlist struct {
	DirID int64
	TID   int64
	Name  string
	Count int
}

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

type QueueItem struct {
	ID       int
	Song     Song
	Status   QueueStatus
	Progress float64
	Speed    float64
	Quality  string
	Error    string
	FilePath string
}

type LogEntry struct {
	Time    string
	Message string
	Level   string
}

var qualityOptions = []string{"臻品母带", "FLAC 无损", "MP3 320K", "MP3 128K"}
var coverSizeOptions = []string{"150 px", "300 px", "500 px", "800 px"}
