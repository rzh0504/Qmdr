package main

import (
	"time"

	"github.com/egoist/mygo/ui"
)

// downloadsView is the merged download page: a tab bar that stays in place,
// with each tab keeping its own scroll position.
func (a *app) downloadsView(c *ui.Context, col colors) {
	ui.Column(c).Grow(1).MinHeight(0).Gap(12).Children(func() {
		a.downloadsTabs(c, col)
		if a.view == viewHistory && a.history.loading && a.history.store == nil {
			loadingState(c, col, "正在加载下载历史…")
			return
		}
		scroll := ui.Scroll(c).Grow(1).MinHeight(0).Gap(18)
		if a.view == viewQueue {
			scroll = scroll.TrackScroll(&a.queue.scroll)
		} else {
			scroll = scroll.TrackScroll(&a.history.scroll)
		}
		scroll.Children(func() {
			if a.view == viewQueue {
				a.queueView(c, col)
			} else {
				a.historyView(c, col)
			}
		})
	})
}

func (a *app) downloadsTabs(c *ui.Context, col colors) {
	ui.Row(c).FillWidth().Height(30).AlignItems(ui.Center).Children(func() {
		segW := float32(84)
		track := ui.Row(c).Height(30).Padding(2).Radius(8).AlignItems(ui.Center).Shrink(0)
		pos := track.Animate("downloads-tab", b2f(a.view == viewHistory), 160*time.Millisecond)
		track.Draw(func(p *ui.Painter, r ui.Rect) {
			chip := smoothRectPath(snapRect(p, r), 8)
			p.FillPath(&chip, col.chip)
			thumb := smoothRectPath(snapRect(p, ui.Rect{
				X: r.X + 2 + pos*segW, Y: r.Y + 2, W: segW, H: 26,
			}), 6)
			p.FillPath(&thumb, col.segThumb)
		})
		track.Children(func() {
			for _, option := range []struct {
				view  viewID
				label string
			}{
				{viewQueue, "下载队列"},
				{viewHistory, "下载历史"},
			} {
				option := option
				active := a.view == option.view
				seg := ui.ButtonBase(c).Width(segW).Height(26).Radius(6).Center().
					Cursor(ui.CursorPointer).Label(option.label)
				if seg.Clicked() {
					a.showDownloadsTab(option.view)
				}
				fg := col.muted
				if active {
					fg = col.text
				}
				seg.Children(func() {
					ui.Text(c, option.label).FontSize(13).TextColor(fg).SingleLine()
				})
			}
		})
		ui.Spacer(c)
	})
}

func (a *app) showDownloadsTab(tab viewID) {
	if tab != viewQueue && tab != viewHistory {
		return
	}
	if a.view == tab {
		return
	}
	if a.view == viewHistory {
		a.resetHistorySelection()
	}
	a.view = tab
	a.downloadsTab = tab
	if tab == viewHistory {
		a.pruneHistoryNow()
	}
	a.refresh()
}

// openQueueTab is used when a download starts, so the merged page opens on
// the queue side and any history selection is put away.
func (a *app) openQueueTab() {
	if a.view == viewHistory {
		a.resetHistorySelection()
	}
	a.view = viewQueue
	a.downloadsTab = viewQueue
}
