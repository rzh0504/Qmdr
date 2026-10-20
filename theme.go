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
	t := c.Theme()
	return colors{
		bg:            t.Background,
		sidebar:       t.Surface,
		card:          t.Background,
		cardAlt:       t.Surface,
		cardHover:     t.SurfaceHover,
		border:        t.Border,
		text:          t.Text,
		muted:         t.TextMuted,
		faint:         t.TextMuted.Alpha(0.65),
		accent:        t.Accent,
		accentHover:   t.AccentHover,
		accentPressed: t.AccentPressed,
		accentText:    t.AccentText,
		accentSoft:    t.Accent.Alpha(0.08),
		accentBorder:  t.Accent.Alpha(0.30),
		gold:          t.Warning,
		red:           t.Danger,
		green:         t.Success,
		blue:          t.Accent,
		shadow:        ui.RGBA(0, 0, 0, 0.06),
	}
}
