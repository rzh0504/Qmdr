package main

import "github.com/egoist/mygo/ui"

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
	btn := ui.ButtonBase(c).Height(30).Padding(0, 14).Gap(6).Radius(8).
		AlignItems(ui.Center).Cursor(ui.CursorPointer).Label(label)
	fill := col.accent
	if btn.Pressed() {
		fill = col.accentPressed
	} else if btn.Hovered() {
		fill = col.accentHover
	}
	// Native fill: at this size its edge stays crisp at fractional scale,
	// where a path fill can leave a pixel of raggedness on the accent.
	btn.Background(fill)
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

func primaryButtonDisabled(c *ui.Context, col colors, icon *ui.SVG, label string) ui.Element {
	btn := ui.ButtonBase(c).Height(30).Padding(0, 14).Gap(6).Radius(8).
		AlignItems(ui.Center).Label(label).Disabled(true)
	btn.Background(col.chip)
	btn.Children(func() {
		if icon != nil {
			ui.Icon(c, icon).Size(14, 14).TextColor(col.faint)
		}
		ui.Text(c, label).FontSize(13).FontWeight(400).TextColor(col.faint).SingleLine()
	})
	return btn
}

func dangerButton(c *ui.Context, col colors, label string, onClick func()) ui.Element {
	btn := ui.ButtonBase(c).Height(30).Padding(0, 14).Gap(6).Radius(8).
		AlignItems(ui.Center).Cursor(ui.CursorPointer).Label(label)
	fill := col.red
	if btn.Pressed() {
		fill = col.red.Alpha(0.82)
	} else if btn.Hovered() {
		fill = col.red.Alpha(0.9)
	}
	btn.Background(fill)
	if btn.Clicked() && onClick != nil {
		onClick()
	}
	fg := ui.RGB(255, 255, 255)
	if col.dark {
		fg = ui.RGB(29, 29, 31)
	}
	btn.Children(func() {
		ui.Text(c, label).FontSize(13).FontWeight(400).TextColor(fg).SingleLine()
	})
	return btn
}

func secondaryButton(c *ui.Context, col colors, icon *ui.SVG, label string, onClick func()) ui.Element {
	btn := ui.ButtonBase(c).Height(30).Padding(0, 12).Gap(6).
		AlignItems(ui.Center).Cursor(ui.CursorPointer).Label(label)
	bg := col.control
	if btn.Pressed() {
		bg = col.cardAlt
	} else if btn.Hovered() {
		bg = col.controlHover
	}
	smoothFace(btn, col, 8, bg)
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
