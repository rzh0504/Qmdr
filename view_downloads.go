package main

import "github.com/egoist/mygo/ui"

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
		track := ui.Row(c).Height(30).Padding(2).Radius(8).AlignItems(ui.Center).Shrink(0)
		smoothFlat(track, col, 8, col.chip)
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
				seg := ui.ButtonBase(c).Height(26).Padding(0, 14).Radius(6).Center().
					Cursor(ui.CursorPointer).Label(option.label)
				if active {
					seg.Background(col.segThumb)
					if col.dark {
						seg.Shadow(0, 1, 1, 0, ui.RGBA(0, 0, 0, 0.25))
					} else {
						seg.Shadow(0, 0.5, 1.5, 0, ui.RGBA(0, 0, 0, 0.16)).
							Shadow(0, 0, 0, 0.5, ui.RGBA(0, 0, 0, 0.04))
					}
				} else if seg.Hovered() {
					seg.Background(col.cardHover)
				}
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
