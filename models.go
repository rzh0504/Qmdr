package main

type Song struct {
	Mid      string
	Title    string
	Artist   string
	Album    string
	Duration string
	Quality  string
	Size     string
	VIP      bool
}

type Playlist struct {
	ID    int
	Name  string
	Count int
	Songs []Song
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

func sampleSongs() []Song {
	return []Song{
		{Title: "晴天", Artist: "周杰伦", Album: "叶惠美", Duration: "04:29", Quality: "无损", Size: "32.4 MB"},
		{Title: "夜曲", Artist: "周杰伦", Album: "十一月的萧邦", Duration: "03:46", Quality: "无损", Size: "28.1 MB"},
		{Title: "稻香", Artist: "周杰伦", Album: "魔杰座", Duration: "03:43", Quality: "320K", Size: "10.2 MB"},
		{Title: "平凡之路", Artist: "朴树", Album: "猎户星座", Duration: "05:01", Quality: "无损", Size: "36.8 MB", VIP: true},
		{Title: "起风了", Artist: "买辣椒也用券", Album: "起风了", Duration: "05:25", Quality: "320K", Size: "12.3 MB"},
		{Title: "演员", Artist: "薛之谦", Album: "绅士", Duration: "04:21", Quality: "无损", Size: "31.7 MB", VIP: true},
		{Title: "句号", Artist: "G.E.M. 邓紫棋", Album: "启示录", Duration: "04:13", Quality: "无损", Size: "30.5 MB"},
		{Title: "如愿", Artist: "王菲", Album: "如愿", Duration: "04:25", Quality: "无损", Size: "33.9 MB", VIP: true},
		{Title: "孤勇者", Artist: "陈奕迅", Album: "孤勇者", Duration: "04:16", Quality: "320K", Size: "9.8 MB"},
		{Title: "光年之外", Artist: "G.E.M. 邓紫棋", Album: "光年之外", Duration: "03:55", Quality: "无损", Size: "29.4 MB"},
	}
}

func samplePlaylists() []Playlist {
	songs := sampleSongs()
	return []Playlist{
		{ID: 1, Name: "我喜欢的音乐", Count: 128, Songs: songs},
		{ID: 2, Name: "华语流行精选", Count: 64, Songs: songs[2:9]},
		{ID: 3, Name: "深夜电台", Count: 42, Songs: []Song{songs[1], songs[7], songs[5], songs[9]}},
		{ID: 4, Name: "跑步 BGM", Count: 88, Songs: []Song{songs[8], songs[2], songs[4], songs[6]}},
		{ID: 5, Name: "周杰伦精选", Count: 50, Songs: songs[0:3]},
	}
}
