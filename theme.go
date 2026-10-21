package main

import "github.com/egoist/mygo/ui"

type colors struct {
	bg, sidebar, card, cardAlt, chip, cardHover, border ui.Color
	text, muted, faint                                  ui.Color
	accent, accentHover, accentPressed, accentText      ui.Color
	accentSoft, accentBorder                            ui.Color
	gold, red, green, blue                              ui.Color
	shadow, floatShadow                                 ui.Color
	control, controlHover                               ui.Color
	rowSelected, rowSelBorder, rowShadow                ui.Color
}

func themeColors(c *ui.Context) colors {
	col := colors{
		bg:            ui.Hex("#ebe0f3"),
		sidebar:       ui.Hex("#eceef4"),
		card:          ui.Hex("#ffffff"),
		cardAlt:       ui.Hex("#f4f3f6"),
		chip:          ui.RGBA(0, 0, 0, 0.045),
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
		floatShadow:   ui.RGBA(20, 20, 40, 0.14),
		control:       ui.Hex("#ffffff"),
		controlHover:  ui.Hex("#f7f7f9"),
		rowSelected:   ui.Hex("#ffffff"),
		rowSelBorder:  ui.RGBA(0, 0, 0, 0.05),
		rowShadow:     ui.RGBA(20, 20, 50, 0.08),
	}

	// Kero's borderless look: surfaces stand out by fill and shadow, not
	// by outlines around every control.
	t := *c.Theme()
	t.Background = col.bg
	t.Surface = col.control
	t.SurfaceHover = col.controlHover
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
	t.Scrollbar = ui.RGBA(0, 0, 0, 0.16)
	t.ScrollbarWidth = 7
	t.Radius = 8
	t.Spacing = 4
	c.SetTheme(&t)
	return col
}

// backdropGradient is kero's "rex" wash: pink lavender above, rose where
// the light comes in, butter cream below.
func backdropGradient() ui.LinearGradient {
	return ui.LinearGradient{
		Angle: 180,
		Oklab: true,
		Stops: []ui.GradientStop{
			{Color: ui.Hex("#ebe0f3"), At: 0},
			{Color: ui.Hex("#f3e7f1"), At: 0.34},
			{Color: ui.Hex("#f7eee4"), At: 0.68},
			{Color: ui.Hex("#f6efcf"), At: 1},
		},
	}
}

// backdropGlow is the rose light from the top right, fading by the middle
// of the window.
func backdropGlow() ui.LinearGradient {
	glow := ui.RGBA(247, 220, 241, 0.75)
	return ui.LinearGradient{
		Angle: 225,
		Oklab: true,
		Stops: []ui.GradientStop{
			{Color: glow, At: 0},
			{Color: glow.Alpha(0), At: 0.5},
		},
	}
}
