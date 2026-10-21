package main

import (
	"context"
	"time"

	"github.com/egoist/mygo/ui"
)

const (
	coverThumbSize  = 150
	maxCachedCovers = 500
	coverFetchLimit = 4
)

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
	go func() {
		a.coverSem <- struct{}{}
		defer func() { <-a.coverSem }()

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		data, _, err := a.api.Cover(ctx, albumMid, coverThumbSize)
		cancel()
		if err != nil {
			a.update(func() {
				delete(a.coverLoading, albumMid)
				a.coverFailed[albumMid] = true
			})
			return
		}
		a.update(func() {
			delete(a.coverLoading, albumMid)
			bitmap, err := ui.DecodeBitmap(data)
			if err != nil {
				a.coverFailed[albumMid] = true
				return
			}
			a.covers[albumMid] = bitmap
		})
	}()
	return nil
}
