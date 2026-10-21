package main

import (
	"time"

	"github.com/egoist/mygo/ui"
)

func b2f(v bool) float32 {
	if v {
		return 1
	}
	return 0
}

// lift raises a surface off the card with shadows alone.
func lift(e ui.Element, col colors) ui.Element {
	if col.dark {
		return e.Shadow(0, 1, 1, 0, ui.RGBA(0, 0, 0, 0.20)).
			Shadow(0, 2, 2, -1, ui.RGBA(0, 0, 0, 0.20)).
			Shadow(0, 4, 6, -3, ui.RGBA(0, 0, 0, 0.24))
	}
	return e.Shadow(0, 1, 2, 0, col.shadow).Shadow(0, 1, 6, 0, col.shadow2)
}

// raised lifts a surface off the card, with a lit edge in the dark.
func raised(e ui.Element, col colors) ui.Element {
	e = lift(e, col)
	if col.dark {
		e = e.Border(1, col.edge)
	}
	return e
}

// chosen lifts a chosen row a hair: a tight ring and a soft hint of a
// shadow, lighter than a card's. In the dark it rises as a pane does.
func chosen(e ui.Element, col colors) ui.Element {
	if col.dark {
		return raised(e, col)
	}
	return e.Shadow(0, 0, 0, 0.5, ui.RGBA(0, 0, 0, 0.06)).
		Shadow(0, 0.5, 2, 0, ui.RGBA(0, 0, 0, 0.05))
}

// face gives a control its face: a fill standing on a soft shadow, with no
// outline in the light and a lit edge in the dark.
func face(e ui.Element, col colors, bg ui.Color) ui.Element {
	if col.dark {
		return raised(e.Background(bg), col)
	}
	return e.Background(bg).
		Shadow(0, 0.5, 1.5, 0, ui.RGBA(0, 0, 0, 0.13)).
		Shadow(0, 0, 0, 0.5, ui.RGBA(0, 0, 0, 0.05))
}

// sheet is a list on the card's face: a soft group that rows sit on.
func sheet(c *ui.Context, col colors, fn func()) ui.Element {
	e := ui.Column(c).FillWidth().Padding(4).Gap(1)
	smoothFlat(e, col, 12, col.chip)
	return e.Children(fn)
}

func softCard(c *ui.Context, col colors, fn func()) ui.Element {
	e := ui.Column(c).FillWidth().Padding(12, 14).Gap(8)
	smoothFlat(e, col, 10, col.chip)
	return e.Children(fn)
}

func groupTitle(c *ui.Context, col colors, title string) ui.Element {
	return ui.Text(c, title).FontSize(12).FontWeight(600).TextColor(col.muted).Padding(0, 4)
}

func sectionHeader(c *ui.Context, col colors, title string, trailing func()) ui.Element {
	return ui.Row(c).FillWidth().AlignItems(ui.Center).Padding(0, 4).Children(func() {
		ui.Text(c, title).FontSize(12).FontWeight(600).TextColor(col.muted).Grow(1)
		if trailing != nil {
			ui.Row(c).Gap(4).AlignItems(ui.Center).Children(trailing)
		}
	})
}

func searchBox(c *ui.Context, col colors, query *string, placeholder string, onSubmit, onClear func()) ui.Element {
	row := ui.Row(c).Gap(7).Padding(0, 11).Height(34).Radius(8).
		Background(col.control).AlignItems(ui.Center)
	var input ui.Element
	row.Children(func() {
		ui.Icon(c, iconSearch).Size(15, 15).TextColor(col.faint)
		input = ui.TextInputBase(c, query).
			Placeholder(placeholder).
			Label(placeholder).
			FontSize(13).
			Grow(1).MinWidth(0).Height(26)
		if input.Submitted() && onSubmit != nil {
			onSubmit()
		}
		if *query != "" {
			iconButton(c, col, iconX, "清空搜索", onClear)
		}
	})
	glow := row.Animate("focus", b2f(input.Focused()), 120*time.Millisecond)
	row.InsetShadow(0, 0, 0, 1, col.fieldStroke)
	if glow > 0 {
		row.Shadow(0, 0, 0, 3.5*glow, col.accent.Alpha(0.32*glow)).
			InsetShadow(0, 0, 0, 1, col.accent.Alpha(0.75*glow))
	}
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
	return ui.Icon(c, icon).Size(14, 14).TextColor(fg)
}

func ghostButton(c *ui.Context, col colors, icon *ui.SVG, label string, onClick func()) ui.Element {
	btn := ui.ButtonBase(c).Gap(5).Padding(3, 8).Radius(6).
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
			ui.Icon(c, icon).Size(14, 14).TextColor(fg)
		}
		ui.Text(c, label).FontSize(12).TextColor(fg).SingleLine()
	})
	return btn
}

func ghostDangerButton(c *ui.Context, col colors, icon *ui.SVG, label string, onClick func()) ui.Element {
	btn := ui.ButtonBase(c).Gap(5).Padding(3, 8).Radius(6).
		AlignItems(ui.Center).Cursor(ui.CursorPointer).Label(label)
	if btn.Hovered() {
		btn = btn.Background(col.red.Alpha(0.07))
	}
	if btn.Clicked() && onClick != nil {
		onClick()
	}
	btn.Children(func() {
		if icon != nil {
			ui.Icon(c, icon).Size(14, 14).TextColor(col.red)
		}
		ui.Text(c, label).FontSize(12).TextColor(col.red).SingleLine()
	})
	return btn
}

func iconButton(c *ui.Context, col colors, icon *ui.SVG, label string, onClick func()) ui.Element {
	btn := ui.ButtonBase(c).Size(28, 26).Radius(6).Justify(ui.Center).
		AlignItems(ui.Center).Cursor(ui.CursorPointer).Label(label).Tooltip(label)
	hovered := btn.Hovered()
	if btn.Pressed() {
		btn = btn.Background(col.cardHover.Alpha(0.12))
	} else if hovered {
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
		ui.Icon(c, icon).Size(15, 15).TextColor(fg)
	})
	return btn
}

func primaryButton(c *ui.Context, col colors, icon *ui.SVG, label string, onClick func()) ui.Element {
	btn := ui.ButtonBase(c).Height(30).Padding(0, 14).Gap(6).
		AlignItems(ui.Center).Cursor(ui.CursorPointer).Label(label)
	fill := col.accent
	if btn.Pressed() {
		fill = col.accentPressed
	} else if btn.Hovered() {
		fill = col.accentHover
	}
	smoothFlat(btn, col, 8, fill)
	if btn.Clicked() && onClick != nil {
		onClick()
	}
	btn.Children(func() {
		if icon != nil {
			ui.Icon(c, icon).Size(14, 14).TextColor(col.accentText)
		}
		ui.Text(c, label).FontSize(13).FontWeight(400).TextColor(col.accentText).SingleLine()
	})
	return btn
}

func secondaryButton(c *ui.Context, col colors, icon *ui.SVG, label string, onClick func()) ui.Element {
	btn := ui.ButtonBase(c).Height(30).Padding(0, 12).Gap(6).Radius(8).
		AlignItems(ui.Center).Cursor(ui.CursorPointer).Label(label)
	bg := col.control
	if btn.Pressed() {
		bg = col.cardAlt
	} else if btn.Hovered() {
		bg = col.controlHover
	}
	face(btn, col, bg)
	if btn.Clicked() && onClick != nil {
		onClick()
	}
	btn.Children(func() {
		if icon != nil {
			ui.Icon(c, icon).Size(14, 14).TextColor(col.muted)
		}
		ui.Text(c, label).FontSize(13).FontWeight(400).TextColor(col.text).SingleLine()
	})
	return btn
}

func chipButton(c *ui.Context, col colors, icon *ui.SVG, label string, onClick func()) ui.Element {
	btn := ui.ButtonBase(c).Gap(4).Padding(2, 6).Radius(5).
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
			ui.Icon(c, icon).Size(12, 12).TextColor(fg)
		}
		ui.Text(c, label).FontSize(12).TextColor(fg).SingleLine()
	})
	return btn
}

func textLink(c *ui.Context, col colors, label string, onClick func()) ui.Element {
	return chipButton(c, col, nil, label, onClick)
}

func selectControl(c *ui.Context, col colors, value *string, options []string, width float32, bg ui.Color, onChange func()) ui.Element {
	sel := ui.SelectBase(c, value)
	trigger := sel.Trigger.Height(30).Padding(0, 10).Gap(8).
		AlignItems(ui.Center).Cursor(ui.CursorPointer).Label(*value)
	if width > 0 {
		trigger.Width(width)
	}
	fill := bg
	switch {
	case trigger.Pressed():
		fill = bg.Mix(col.text, 0.08)
	case trigger.Hovered():
		fill = bg.Mix(col.text, 0.04)
	}
	smoothFlat(trigger, col, 7, fill)
	if trigger.Changed() && onChange != nil {
		onChange()
	}
	trigger.Children(func() {
		ui.Text(c, *value).FontSize(13).TextColor(col.text).Grow(1).MinWidth(0).SingleLine()
		ui.Icon(c, iconChevronDown).Size(13, 13).TextColor(col.muted)
	})
	sel.Popup(func(panel ui.Element) {
		panel.Padding(4).Radius(10).Background(col.card).Border(1, col.border).
			Shadow(0, 1, 3, 0, col.shadow).
			Shadow(0, 8, 24, 0, col.floatShadow)
		for _, option := range options {
			item := sel.Item(option).Height(28).Padding(0, 10).Radius(6).AlignItems(ui.Center)
			fg := col.text
			if item.Highlighted() {
				item.Background(col.accent)
				fg = col.accentText
			} else if option == *value {
				item.Background(col.chip)
			}
			item.Children(func() {
				ui.Text(c, option).FontSize(13).TextColor(fg).SingleLine()
			})
		}
	})
	return trigger
}

func segmentedControl(c *ui.Context, col colors, selected *int, labels ...string) ui.Element {
	seg := ui.SegmentedBase(c, selected, len(labels))
	seg.Track.Height(28).Padding(2).Radius(7).
		Background(col.chip).AlignItems(ui.Center).Children(func() {
		for n, label := range labels {
			if n > 0 {
				sep := ui.Box(c).Width(1).Height(12).Shrink(0)
				if n != *selected && n-1 != *selected {
					sep.Background(col.text.Alpha(0.14))
				}
			}
			s := seg.Segment(n).FillHeight().Padding(0, 12).Radius(5).Center()
			if n == *selected {
				s.Background(col.segThumb)
				if col.dark {
					s.Shadow(0, 1, 1, 0, ui.RGBA(0, 0, 0, 0.25))
				} else {
					s.Shadow(0, 0.5, 1.5, 0, ui.RGBA(0, 0, 0, 0.16)).
						Shadow(0, 0, 0, 0.5, ui.RGBA(0, 0, 0, 0.04))
				}
			}
			s.Children(func() {
				ui.Text(c, label).FontSize(13).SingleLine().TextColor(col.text)
			})
		}
	})
	return seg.Track
}

func stringSegmented(c *ui.Context, col colors, value *string, values, labels []string, onChange func()) ui.Element {
	index := 0
	for i, v := range values {
		if v == *value {
			index = i
			break
		}
	}
	control := segmentedControl(c, col, &index, labels...)
	if control.Changed() && index >= 0 && index < len(values) {
		*value = values[index]
		if onChange != nil {
			onChange()
		}
	}
	return control
}

func stepperControl(c *ui.Context, col colors, value *float64, lo, hi, step float64, onChange func()) ui.Element {
	box := ui.Row(c).Height(30).Padding(2, 3).Radius(7).Background(col.control).
		AlignItems(ui.Center)
	face(box, col, col.control)
	change := func(next float64) {
		*value = min(hi, max(lo, next))
		if onChange != nil {
			onChange()
		}
	}
	box.Children(func() {
		minus := ui.ButtonBase(c).Size(24, 26).Radius(5).Justify(ui.Center).
			AlignItems(ui.Center).Label("减少").Tooltip("减少")
		if minus.Hovered() {
			minus.Background(col.cardHover)
		}
		if minus.Clicked() {
			change(*value - step)
		}
		minus.Children(func() {
			ui.Icon(c, iconMinus).Size(13, 13).TextColor(col.muted)
		})
		ui.Textf(c, "%.0f", *value).Width(28).FontSize(13).
			TextAlign(ui.Center).TextColor(col.text)
		plus := ui.ButtonBase(c).Size(24, 26).Radius(5).Justify(ui.Center).
			AlignItems(ui.Center).Label("增加").Tooltip("增加")
		if plus.Hovered() {
			plus.Background(col.cardHover)
		}
		if plus.Clicked() {
			change(*value + step)
		}
		plus.Children(func() {
			ui.Icon(c, iconPlus).Size(13, 13).TextColor(col.muted)
		})
	})
	return box
}

func toggleSwitch(c *ui.Context, col colors, on *bool) ui.Element {
	sw := ui.SwitchBase(c, on).Size(38, 22).Radius(11).Padding(2).Shrink(0)
	track := col.text.Alpha(0.14)
	if *on {
		track = col.accent
	}
	sw.Background(sw.AnimateColor("track", track, ui.Animation{Duration: 150 * time.Millisecond}))
	x := sw.AnimateTo("knob", b2f(*on)*16, ui.Animation{
		Spring: ui.SpringOptions{Duration: 260 * time.Millisecond, Bounce: 0.15},
	})
	sw.Children(func() {
		ui.Box(c).Size(18, 18).Radius(9).Background(ui.RGB(255, 255, 255)).Translate(x, 0).
			Shadow(0, 1, 2, 0, ui.RGBA(0, 0, 0, 0.22)).
			Shadow(0, 0, 0, 0.5, ui.RGBA(0, 0, 0, 0.06))
	})
	return sw
}

func playlistCover(c *ui.Context, col colors, index int, size float32) ui.Element {
	box := ui.Box(c).Size(size, size).Radius(6).Background(col.chip)
	box.Children(func() {
		ui.Row(c).Fill().Center().Children(func() {
			ui.Icon(c, iconLibrary).Size(size*0.36, size*0.36).TextColor(col.faint)
		})
	})
	return box
}

func (a *app) songRow(c *ui.Context, col colors, song Song, index int, showIndex bool, onDownload func()) ui.Element {
	row := ui.Row(c).Gap(10).Padding(7, 10).Height(52).Radius(8).AlignItems(ui.Center)
	if row.Hovered() {
		row.Background(col.rowSelected)
		row = chosen(row, col)
	}
	row.Children(func() {
		if showIndex {
			ui.Textf(c, "%02d", index+1).Width(24).FontSize(11).
				TextColor(col.faint).SingleLine()
		}
		a.cover(c, col, song.AlbumMid, 34)
		ui.Column(c).Gap(2).Grow(1).MinWidth(0).Children(func() {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Text(c, song.Title).FontSize(13).FontWeight(400).TextColor(col.text).SingleLine()
				if song.VIP {
					ui.Text(c, "VIP").FontSize(9).FontWeight(600).TextColor(col.gold).SingleLine()
				}
				if song.Quality != "" {
					ui.Text(c, song.Quality).FontSize(10).TextColor(col.muted).SingleLine()
				}
			})
			ui.Textf(c, "%s · %s", song.Artist, song.Album).FontSize(12).
				TextColor(col.muted).SingleLine()
		})
		ui.Text(c, song.Duration).Width(38).FontSize(11).TextColor(col.faint).SingleLine()
		if onDownload != nil {
			iconButton(c, col, iconDownload, "下载 "+song.Title, onDownload)
		}
	})
	return row
}

func emptyState(c *ui.Context, col colors, icon *ui.SVG, title, subtitle string) ui.Element {
	return ui.Column(c).FillWidth().Padding(36).Center().Gap(0).Children(func() {
		ui.Box(c).Size(48, 48).Radius(13).Background(col.chip).Children(func() {
			ui.Row(c).Fill().Center().Children(func() {
				ui.Icon(c, icon).Size(22, 22).TextColor(col.faint)
			})
		})
		ui.Text(c, title).FontSize(15).FontWeight(600).TextColor(col.text).Margin(14, 0, 0, 0)
		ui.Text(c, subtitle).FontSize(12).TextColor(col.muted).MaxWidth(380).
			TextAlign(ui.Center).LineHeight(1.45).Margin(5, 0, 0, 0)
	})
}

func divider(c *ui.Context, col colors) ui.Element {
	return ui.Box(c).Height(1).FillWidth().Background(col.text.Alpha(0.07))
}
