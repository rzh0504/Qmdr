package main

import "github.com/egoist/mygo/ui"

func card(c *ui.Context, col colors, fn func()) ui.Element {
	return ui.Column(c).Padding(14).Gap(10).Radius(8).
		Background(col.card).Border(1, col.border).Children(fn)
}

func softCard(c *ui.Context, col colors, fn func()) ui.Element {
	return ui.Column(c).Padding(12).Gap(8).Radius(6).
		Background(col.cardAlt).Border(1, col.border).Children(fn)
}

func tableCard(c *ui.Context, col colors, fn func()) ui.Element {
	return ui.Column(c).Gap(0).
		BorderWidth(1, 0, 1, 0).BorderColor(col.border).Children(fn)
}

func pageHeader(c *ui.Context, col colors, title string, actions func()) {
	ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
		ui.Text(c, title).FontSize(20).Bold().TextColor(col.text).Grow(1)
		if actions != nil {
			ui.Row(c).Gap(4).AlignItems(ui.Center).Children(actions)
		}
	})
	divider(c, col)
}

func sectionHeader(c *ui.Context, col colors, title string, trailing func()) {
	ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
		ui.Text(c, title).FontSize(13).Bold().TextColor(col.text).Grow(1)
		if trailing != nil {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(trailing)
		}
	})
}

func pill(c *ui.Context, text string, fg, bg ui.Color) ui.Element {
	return ui.Row(c).Gap(3).Padding(1, 5).Radius(3).Background(bg).
		AlignItems(ui.Center).Children(func() {
		ui.Text(c, text).FontSize(9).Bold().TextColor(fg).SingleLine()
	})
}

func searchBox(c *ui.Context, col colors, query *string, placeholder string, onSubmit, onClear func()) ui.Element {
	row := ui.Row(c).Gap(6).Padding(0, 9).Height(34).Radius(6).
		Background(col.cardAlt).Border(1, col.border).AlignItems(ui.Center)
	row.Children(func() {
		ui.Icon(c, iconSearch).FontSize(14).TextColor(col.muted)
		input := ui.TextInputBase(c, query).
			Placeholder(placeholder).
			Label(placeholder).
			FontSize(12).
			Grow(1).Height(28)
		if input.Submitted() && onSubmit != nil {
			onSubmit()
		}
		if *query != "" {
			iconButton(c, col, iconX, "清空搜索", onClear)
		}
	})
	return row
}

func statusIcon(c *ui.Context, col colors, status QueueStatus) ui.Element {
	icon := iconClock
	fg := col.muted
	switch status {
	case StatusDownloading:
		icon, fg = iconDownload, col.accent
	case StatusDone:
		icon, fg = iconCheck, col.green
	case StatusFailed:
		icon, fg = iconAlert, col.red
	case StatusCancelled:
		icon, fg = iconX, col.gold
	case StatusWaiting:
		icon, fg = iconClock, col.muted
	}
	return ui.Icon(c, icon).FontSize(13).TextColor(fg)
}

func ghostButton(c *ui.Context, col colors, icon *ui.SVG, label string, onClick func()) ui.Element {
	btn := ui.ButtonBase(c).Gap(4).Padding(3, 6).Radius(5).
		AlignItems(ui.Center).Cursor(ui.CursorPointer).Label(label)
	hovered := btn.Hovered()
	if hovered {
		btn = btn.Background(col.cardHover)
	}
	if btn.Clicked() && onClick != nil {
		onClick()
	}
	fg := col.muted
	if hovered {
		fg = col.text
	}
	btn.Children(func() {
		if icon != nil {
			ui.Icon(c, icon).FontSize(13).TextColor(fg)
		}
		ui.Text(c, label).FontSize(11).TextColor(fg).SingleLine()
	})
	return btn
}

func ghostDangerButton(c *ui.Context, col colors, icon *ui.SVG, label string, onClick func()) ui.Element {
	btn := ui.ButtonBase(c).Gap(4).Padding(3, 6).Radius(5).
		AlignItems(ui.Center).Cursor(ui.CursorPointer).Label(label)
	if btn.Hovered() {
		btn = btn.Background(col.red.Alpha(0.07))
	}
	if btn.Clicked() && onClick != nil {
		onClick()
	}
	btn.Children(func() {
		if icon != nil {
			ui.Icon(c, icon).FontSize(13).TextColor(col.red)
		}
		ui.Text(c, label).FontSize(11).TextColor(col.red).SingleLine()
	})
	return btn
}

func iconButton(c *ui.Context, col colors, icon *ui.SVG, label string, onClick func()) ui.Element {
	btn := ui.ButtonBase(c).Size(26, 26).Radius(5).Justify(ui.Center).
		AlignItems(ui.Center).Cursor(ui.CursorPointer).Label(label).Tooltip(label)
	hovered := btn.Hovered()
	if hovered {
		btn = btn.Background(col.cardHover)
	}
	if btn.Clicked() && onClick != nil {
		onClick()
	}
	fg := col.muted
	if hovered {
		fg = col.text
	}
	btn.Children(func() {
		ui.Icon(c, icon).FontSize(14).TextColor(fg)
	})
	return btn
}

func primaryButton(c *ui.Context, col colors, icon *ui.SVG, label string, onClick func()) ui.Element {
	btn := ui.PrimaryButton(c, "").Label(label).Padding(0, 10).Height(30).Radius(6)
	if btn.Clicked() && onClick != nil {
		onClick()
	}
	btn.Children(func() {
		if icon != nil {
			ui.Icon(c, icon).FontSize(13)
		}
		ui.Text(c, label).FontSize(11).Bold().SingleLine()
	})
	return btn
}

func secondaryButton(c *ui.Context, col colors, icon *ui.SVG, label string, onClick func()) ui.Element {
	btn := ui.Button(c, "").Label(label).Padding(0, 10).Height(30).Radius(6).
		Background(col.card).Border(1, col.border)
	if btn.Hovered() {
		btn = btn.Background(col.cardHover)
	}
	if btn.Clicked() && onClick != nil {
		onClick()
	}
	btn.Children(func() {
		if icon != nil {
			ui.Icon(c, icon).FontSize(13).TextColor(col.muted)
		}
		ui.Text(c, label).FontSize(11).TextColor(col.text).SingleLine()
	})
	return btn
}

func chipButton(c *ui.Context, col colors, icon *ui.SVG, label string, onClick func()) ui.Element {
	btn := ui.ButtonBase(c).Gap(4).Padding(2, 5).Radius(4).
		AlignItems(ui.Center).Cursor(ui.CursorPointer).Label(label)
	hovered := btn.Hovered()
	if hovered {
		btn = btn.Background(col.cardHover)
	}
	if btn.Clicked() && onClick != nil {
		onClick()
	}
	fg := col.muted
	if hovered {
		fg = col.text
	}
	btn.Children(func() {
		if icon != nil {
			ui.Icon(c, icon).FontSize(11).TextColor(fg)
		}
		ui.Text(c, label).FontSize(10).TextColor(fg).SingleLine()
	})
	return btn
}

func textLink(c *ui.Context, col colors, label string, onClick func()) ui.Element {
	return chipButton(c, col, nil, label, onClick)
}

func playlistCover(c *ui.Context, col colors, index int, size float32) ui.Element {
	box := ui.Box(c).Size(size, size).Radius(4).
		Background(col.cardAlt).Border(1, col.border)
	box.Children(func() {
		ui.Row(c).Fill().Center().Children(func() {
			ui.Icon(c, iconLibrary).FontSize(size * 0.38).TextColor(col.muted)
		})
	})
	return box
}

func (a *app) songRow(c *ui.Context, col colors, song Song, index int, showIndex bool, onDownload func()) ui.Element {
	row := ui.Row(c).Gap(8).Padding(6, 8).Radius(4).AlignItems(ui.Center)
	if row.Hovered() {
		row = row.Background(col.cardHover)
	}
	row.Children(func() {
		if showIndex {
			ui.Textf(c, "%02d", index+1).Width(22).FontSize(9).
				TextColor(col.faint).SingleLine()
		}
		a.cover(c, col, song.AlbumMid, 32)
		ui.Column(c).Gap(1).Grow(1).Children(func() {
			ui.Row(c).Gap(5).AlignItems(ui.Center).Children(func() {
				ui.Text(c, song.Title).FontSize(12).Bold().TextColor(col.text).SingleLine()
				if song.VIP {
					ui.Text(c, "VIP").FontSize(8).Bold().TextColor(col.gold).SingleLine()
				}
				if song.Quality != "" {
					ui.Text(c, song.Quality).FontSize(8).TextColor(col.muted).SingleLine()
				}
			})
			ui.Textf(c, "%s · %s", song.Artist, song.Album).FontSize(10).
				TextColor(col.muted).SingleLine()
		})
		ui.Text(c, song.Duration).Width(38).FontSize(10).TextColor(col.muted).SingleLine()
		if onDownload != nil {
			iconButton(c, col, iconDownload, "下载 "+song.Title, onDownload)
		}
	})
	return row
}

func emptyState(c *ui.Context, col colors, icon *ui.SVG, title, subtitle string) ui.Element {
	return ui.Column(c).FillWidth().Padding(24).Center().Gap(6).Children(func() {
		ui.Box(c).Size(36, 36).Radius(8).Background(col.cardAlt).
			Border(1, col.border).Children(func() {
			ui.Row(c).Fill().Center().Children(func() {
				ui.Icon(c, icon).FontSize(17).TextColor(col.muted)
			})
		})
		ui.Text(c, title).FontSize(13).Bold().TextColor(col.text)
		ui.Text(c, subtitle).FontSize(10).TextColor(col.muted)
	})
}

func divider(c *ui.Context, col colors) ui.Element {
	return ui.Box(c).Height(1).FillWidth().Background(col.border)
}
