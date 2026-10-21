package main

import (
	"time"

	"github.com/egoist/mygo/ui"
)

func searchBox(c *ui.Context, col colors, query *string, placeholder string, onSubmit, onClear func()) ui.Element {
	row := ui.Row(c).Gap(7).Padding(0, 11).Height(34).AlignItems(ui.Center)
	var input ui.Element
	row.Children(func() {
		ui.Icon(c, iconSearch).Size(15, 15).TextColor(col.faint)
		input = ui.TextInputBase(c, query).
			Placeholder(placeholder).
			Label(placeholder).
			FontSize(13).
			Grow(1).MinWidth(0).Height(26).FocusRing(false)
		if input.Submitted() && onSubmit != nil {
			onSubmit()
		}
		if *query != "" {
			iconButton(c, col, iconX, "清空搜索", onClear)
		}
	})
	glow := row.Animate("focus", b2f(input.Focused()), 120*time.Millisecond)
	smoothField(row, col, 8, col.control, col.fieldStroke, col.accent, glow)
	return row
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
	box := ui.Row(c).Height(30).Padding(2, 3).AlignItems(ui.Center)
	smoothFace(box, col, 7, col.control)
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
