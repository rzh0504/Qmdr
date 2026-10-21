package main

import "github.com/egoist/mygo/ui"

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
