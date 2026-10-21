package main

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

type Playlist struct {
	DirID int64  `json:"dir_id"`
	TID   int64  `json:"tid"`
	Name  string `json:"name"`
	Count int    `json:"count"`
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
	FileSize int64
}

type LogEntry struct {
	Time    string
	Message string
	Level   string
}

var qualityOptions = []string{"臻品母带", "FLAC 无损", "MP3 320K", "MP3 128K"}
var coverSizeOptions = []string{"150 px", "300 px", "500 px", "800 px"}
var historyKeepOptions = []string{"30 天", "90 天", "180 天", "365 天", "永久"}
