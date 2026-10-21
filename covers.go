package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/rzh0504/qmdr/internal/fsutil"
)

const (
	coverThumbSize  = 150
	maxCachedCovers = 500
	coverFetchLimit = 4
)

var coverMidPattern = regexp.MustCompile(`^[A-Za-z0-9]+$`)

func coverCachePath(albumMid string) string {
	if !coverMidPattern.MatchString(albumMid) {
		return ""
	}
	dir, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "Qmdr", "covers", fmt.Sprintf("%s-%d.img", albumMid, coverThumbSize))
}

func readCoverCache(albumMid string) ([]byte, bool) {
	path := coverCachePath(albumMid)
	if path == "" {
		return nil, false
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) < 1024 {
		return nil, false
	}
	return data, true
}

func writeCoverCache(albumMid string, data []byte) {
	path := coverCachePath(albumMid)
	if path == "" || len(data) < 1024 {
		return
	}
	_ = fsutil.WriteFileAtomic(path, data, 0o644)
}

func (a *app) cover(c *ui.Context, col colors, albumMid string, size float32) ui.Element {
	bitmap := a.coverBitmap(albumMid)
	box := ui.Box(c).Size(size, size).Radius(4).
		Background(col.cardAlt).Border(1, col.border).Clip()
	box.Children(func() {
		if bitmap != nil {
			ui.Image(c, bitmap).Size(size, size).Fit(ui.Cover)
			return
		}
		ui.Row(c).Fill().Center().Children(func() {
			ui.Icon(c, iconMusic).FontSize(size * 0.38).TextColor(col.muted)
		})
	})
	return box
}

func (a *app) coverBitmap(albumMid string) *ui.Bitmap {
	if albumMid == "" {
		return nil
	}
	if bitmap, ok := a.covers[albumMid]; ok {
		return bitmap
	}
	if a.coverLoading[albumMid] || a.coverFailed[albumMid] || len(a.covers) >= maxCachedCovers {
		return nil
	}
	a.coverLoading[albumMid] = true
	go a.loadCover(albumMid)
	return nil
}

func (a *app) loadCover(albumMid string) {
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Printf("cover load panic: %v", recovered)
			a.finishCover(albumMid, nil, fmt.Errorf("封面加载异常"))
		}
	}()
	a.coverSem <- struct{}{}
	defer func() { <-a.coverSem }()

	if data, ok := readCoverCache(albumMid); ok {
		a.finishCover(albumMid, data, nil)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	data, _, err := a.api.Cover(ctx, albumMid, coverThumbSize)
	cancel()
	if err == nil {
		writeCoverCache(albumMid, data)
	}
	a.finishCover(albumMid, data, err)
}

func (a *app) finishCover(albumMid string, data []byte, err error) {
	var bitmap *ui.Bitmap
	if err == nil {
		bitmap, err = ui.DecodeBitmap(data)
		if err != nil {
			_ = os.Remove(coverCachePath(albumMid))
		}
	}
	a.update(func() {
		delete(a.coverLoading, albumMid)
		if err != nil {
			a.coverFailed[albumMid] = true
			return
		}
		a.covers[albumMid] = bitmap
	})
}
