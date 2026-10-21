package queue

import (
	"testing"
	"time"

	"github.com/rzh0504/qmdr/internal/model"
)

func TestRecordRoundTrip(t *testing.T) {
	createdAt := time.Unix(1700000000, 0)
	original := model.QueueItem{
		ID: 7,
		Song: model.Song{
			Mid:      "mid",
			Title:    "Title",
			Artist:   "Artist",
			Album:    "Album",
			AlbumMid: "album-mid",
			Duration: "03:00",
			Quality:  "无损",
			VIP:      true,
		},
		Status:    model.StatusFailed,
		Progress:  0.4,
		Quality:   "FLAC",
		Error:     "network",
		FilePath:  `C:\Music\song.flac`,
		FileSize:  12345,
		CreatedAt: createdAt,
	}
	restored := ItemFromRecord(RecordFromItem(original))
	if restored != original {
		t.Fatalf("queue round trip = %+v, want %+v", restored, original)
	}
}
