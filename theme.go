package main

import "github.com/egoist/mygo/ui"

type colors struct {
	bg, sidebar, card, cardAlt, cardHover, border  ui.Color
	text, muted, faint                             ui.Color
	accent, accentHover, accentPressed, accentText ui.Color
	accentSoft, accentBorder                       ui.Color
	gold, red, green, blue                         ui.Color
	shadow                                         ui.Color
}

func themeColors(c *ui.Context) colors {
	col := colors{
		bg:            ui.Hex("#f1f2f3"),
		sidebar:       ui.Hex("#eceef4"),
		card:          ui.Hex("#ffffff"),
		cardAlt:       ui.Hex("#f4f4f6"),
		cardHover:     ui.RGBA(0, 0, 0, 0.04),
		border:        ui.RGBA(0, 0, 0, 0.07),
		text:          ui.Hex("#1d1d1f"),
		muted:         ui.Hex("#6e6e78"),
		faint:         ui.Hex("#a3a3ad"),
		accent:        ui.Hex("#3b6cf6"),
		accentHover:   ui.Hex("#2f5bd8"),
		accentPressed: ui.Hex("#2448b0"),
		accentText:    ui.Hex("#ffffff"),
		accentSoft:    ui.RGBA(59, 108, 246, 0.10),
		accentBorder:  ui.RGBA(59, 108, 246, 0.28),
		gold:          ui.Hex("#c27b0f"),
		red:           ui.Hex("#d93a35"),
		green:         ui.Hex("#22863a"),
		blue:          ui.Hex("#3b6cf6"),
		shadow:        ui.RGBA(0, 0, 0, 0.07),
	}

	t := *c.Theme()
	t.Background = col.bg
	t.Surface = col.card
	t.SurfaceHover = ui.Hex("#f7f7f9")
	t.SurfacePressed = ui.Hex("#ececf0")
	t.Border = ui.Transparent
	t.Text = col.text
	t.TextMuted = col.muted
	t.Accent = col.accent
	t.AccentHover = col.accentHover
	t.AccentPressed = col.accentPressed
	t.AccentText = col.accentText
	t.Danger = col.red
	t.Warning = col.gold
	t.Success = col.green
	t.Selection = col.accent.Alpha(0.25)
	t.Focus = col.accent.Alpha(0.55)
	t.Scrollbar = ui.RGBA(0, 0, 0, 0.20)
	t.ScrollbarWidth = 7
	t.Radius = 8
	c.SetTheme(&t)
	return col
}

func backdropGradient() ui.LinearGradient {
	return ui.LinearGradient{
		Angle: 180,
		Oklab: true,
		Stops: []ui.GradientStop{
			{Color: ui.Hex("#e3e9f4"), At: 0},
			{Color: ui.Hex("#ebeff6"), At: 0.34},
			{Color: ui.Hex("#f1f2f3"), At: 0.68},
			{Color: ui.Hex("#f2f1ec"), At: 1},
		},
	}
}
