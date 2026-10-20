package main

import "github.com/egoist/mygo/ui"

func card(c *ui.Context, col colors, fn func()) ui.Element {
	return ui.Column(c).Padding(16).Gap(12).Radius(10).
		Background(col.card).Border(1, col.border).
		Shadow(0, 1, 3, 0, col.shadow).Children(fn)
}

func softCard(c *ui.Context, col colors, fn func()) ui.Element {
	return ui.Column(c).Padding(14).Gap(10).Radius(10).
		Background(col.cardAlt).Border(1, col.border).Children(fn)
}

func tableCard(c *ui.Context, col colors, fn func()) ui.Element {
	return ui.Column(c).Padding(8).Gap(0).Radius(10).
		Background(col.card).Border(1, col.border).
		Shadow(0, 1, 3, 0, col.shadow).Children(fn)
}

func pageHeader(c *ui.Context, col colors, title, subtitle string, actions func()) {
	ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
		ui.Column(c).Gap(3).Grow(1).Children(func() {
			ui.Text(c, title).FontSize(24).Bold().TextColor(col.text)
			ui.Text(c, subtitle).FontSize(12).TextColor(col.muted)
		})
		if actions != nil {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(actions)
		}
	})
}

func sectionHeader(c *ui.Context, col colors, title, subtitle string, trailing func()) {
	ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
		ui.Column(c).Gap(2).Grow(1).Children(func() {
			ui.Text(c, title).FontSize(15).Bold().TextColor(col.text)
			if subtitle != "" {
				ui.Text(c, subtitle).FontSize(11).TextColor(col.muted)
			}
		})
		if trailing != nil {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(trailing)
		}
	})
}

func pill(c *ui.Context, text string, fg, bg ui.Color) ui.Element {
	return ui.Row(c).Gap(4).Padding(2, 6).Radius(4).Background(bg).
		AlignItems(ui.Center).Children(func() {
		ui.Text(c, text).FontSize(9).Bold().TextColor(fg).SingleLine()
	})
}

func searchBox(c *ui.Context, col colors, query *string, placeholder string, onSubmit, onClear func()) ui.Element {
	row := ui.Row(c).Gap(7).Padding(0, 10).Height(36).Radius(7).
		Background(col.cardAlt).Border(1, col.border).AlignItems(ui.Center)
	row.Children(func() {
		ui.Icon(c, iconSearch).FontSize(15).TextColor(col.muted)
		input := ui.TextInputBase(c, query).
			Placeholder(placeholder).
			Label(placeholder).
			FontSize(13).
			Grow(1).Height(32)
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
	return ui.Icon(c, icon).FontSize(14).TextColor(fg)
}

func ghostButton(c *ui.Context, col colors, icon *ui.SVG, label string, onClick func()) ui.Element {
	btn := ui.ButtonBase(c).Gap(5).Padding(4, 8).Radius(6).
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
			ui.Icon(c, icon).FontSize(14).TextColor(fg)
		}
		ui.Text(c, label).FontSize(12).TextColor(fg).SingleLine()
	})
	return btn
}

func ghostDangerButton(c *ui.Context, col colors, icon *ui.SVG, label string, onClick func()) ui.Element {
	btn := ui.ButtonBase(c).Gap(5).Padding(4, 8).Radius(6).
		AlignItems(ui.Center).Cursor(ui.CursorPointer).Label(label)
	if btn.Hovered() {
		btn = btn.Background(col.red.Alpha(0.07))
	}
	if btn.Clicked() && onClick != nil {
		onClick()
	}
	btn.Children(func() {
		if icon != nil {
			ui.Icon(c, icon).FontSize(14).TextColor(col.red)
		}
		ui.Text(c, label).FontSize(12).TextColor(col.red).SingleLine()
	})
	return btn
}

func iconButton(c *ui.Context, col colors, icon *ui.SVG, label string, onClick func()) ui.Element {
	btn := ui.ButtonBase(c).Size(28, 28).Radius(6).Justify(ui.Center).
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
		ui.Icon(c, icon).FontSize(15).TextColor(fg)
	})
	return btn
}

func primaryButton(c *ui.Context, col colors, icon *ui.SVG, label string, onClick func()) ui.Element {
	btn := ui.PrimaryButton(c, "").Label(label).Padding(0, 12).Height(32).Radius(7)
	if btn.Clicked() && onClick != nil {
		onClick()
	}
	btn.Children(func() {
		if icon != nil {
			ui.Icon(c, icon).FontSize(14)
		}
		ui.Text(c, label).FontSize(12).Bold().SingleLine()
	})
	return btn
}

func secondaryButton(c *ui.Context, col colors, icon *ui.SVG, label string, onClick func()) ui.Element {
	btn := ui.Button(c, "").Label(label).Padding(0, 12).Height(32).Radius(7).
		Background(col.card).Border(1, col.border)
	if btn.Hovered() {
		btn = btn.Background(col.cardHover)
	}
	if btn.Clicked() && onClick != nil {
		onClick()
	}
	btn.Children(func() {
		if icon != nil {
			ui.Icon(c, icon).FontSize(14).TextColor(col.muted)
		}
		ui.Text(c, label).FontSize(12).TextColor(col.text).SingleLine()
	})
	return btn
}

func chipButton(c *ui.Context, col colors, icon *ui.SVG, label string, onClick func()) ui.Element {
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
			ui.Icon(c, icon).FontSize(12).TextColor(fg)
		}
		ui.Text(c, label).FontSize(11).TextColor(fg).SingleLine()
	})
	return btn
}

func textLink(c *ui.Context, col colors, label string, onClick func()) ui.Element {
	return chipButton(c, col, nil, label, onClick)
}

func cover(c *ui.Context, col colors, index int, size float32, icon *ui.SVG) ui.Element {
	box := ui.Box(c).Size(size, size).Radius(size*0.16).
		Background(col.cardAlt).Border(1, col.border)
	box.Children(func() {
		ui.Row(c).Fill().Center().Children(func() {
			ui.Icon(c, icon).FontSize(size * 0.40).TextColor(col.muted)
		})
	})
	return box
}

func songRow(c *ui.Context, col colors, song Song, index int, showIndex bool, onDownload func()) ui.Element {
	row := ui.Row(c).Gap(10).Padding(7, 10).Radius(6).AlignItems(ui.Center)
	if row.Hovered() {
		row = row.Background(col.cardHover)
	}
	row.Children(func() {
		if showIndex {
			ui.Textf(c, "%02d", index+1).Width(24).FontSize(10).
				TextColor(col.faint).SingleLine()
		}
		cover(c, col, index, 38, iconMusic)
		ui.Column(c).Gap(2).Grow(1).Children(func() {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Text(c, song.Title).FontSize(13).Bold().TextColor(col.text).SingleLine()
				if song.VIP {
					ui.Text(c, "VIP").FontSize(9).Bold().TextColor(col.gold).SingleLine()
				}
				if song.Quality != "" {
					ui.Text(c, song.Quality).FontSize(9).TextColor(col.muted).SingleLine()
				}
			})
			ui.Textf(c, "%s · %s", song.Artist, song.Album).FontSize(11).
				TextColor(col.muted).SingleLine()
		})
		ui.Text(c, song.Duration).Width(42).FontSize(11).TextColor(col.muted).SingleLine()
		if onDownload != nil {
			iconButton(c, col, iconDownload, "下载 "+song.Title, onDownload)
		}
	})
	return row
}

func emptyState(c *ui.Context, col colors, icon *ui.SVG, title, subtitle string) ui.Element {
	return ui.Column(c).FillWidth().Padding(30).Center().Gap(8).Children(func() {
		ui.Box(c).Size(44, 44).Radius(10).Background(col.cardAlt).
			Border(1, col.border).Children(func() {
			ui.Row(c).Fill().Center().Children(func() {
				ui.Icon(c, icon).FontSize(20).TextColor(col.muted)
			})
		})
		ui.Text(c, title).FontSize(14).Bold().TextColor(col.text)
		ui.Text(c, subtitle).FontSize(11).TextColor(col.muted)
	})
}

func divider(c *ui.Context, col colors) ui.Element {
	return ui.Box(c).Height(1).FillWidth().Background(col.border)
}
