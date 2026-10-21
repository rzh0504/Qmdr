package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/egoist/mygo/ui"
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
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0o644)
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
	a.update(func() {
		delete(a.coverLoading, albumMid)
		if err != nil {
			a.coverFailed[albumMid] = true
			return
		}
		bitmap, decodeErr := ui.DecodeBitmap(data)
		if decodeErr != nil {
			_ = os.Remove(coverCachePath(albumMid))
			a.coverFailed[albumMid] = true
			return
		}
		a.covers[albumMid] = bitmap
	})
}
