package main

import (
	"runtime"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

type colors struct {
	dark bool

	bg, sidebar, card, cardAlt, chip, cardHover, border ui.Color
	text, muted, faint                                  ui.Color
	accent, accentHover, accentPressed, accentText      ui.Color
	accentSoft, accentBorder                            ui.Color
	gold, red, green, blue                              ui.Color
	shadow, shadow2, floatShadow                        ui.Color
	control, controlHover                               ui.Color
	rowSelected, rowSelBorder, rowShadow                ui.Color
	fieldStroke, sheetHover, segThumb, edge             ui.Color
}

func (a *app) themeColors(c *ui.Context) colors {
	dark := c.Theme().Dark
	w := washByID(a.background)
	col := colors{dark: dark}
	if dark {
		col.bg = ui.Hex(w.dark[0])
		col.sidebar = ui.Hex("#141418")
		col.card = ui.Hex("#232328")
		col.cardAlt = ui.Hex("#2b2b31")
		col.chip = ui.RGBA(255, 255, 255, 0.07)
		col.cardHover = ui.RGBA(255, 255, 255, 0.05)
		col.border = ui.RGBA(255, 255, 255, 0.08)
		col.text = ui.Hex("#ececf0")
		col.muted = ui.Hex("#a9a9b4")
		col.faint = ui.Hex("#83838f")
		col.accent = ui.Hex("#6b93ff")
		col.accentHover = ui.Hex("#8aabff")
		col.accentPressed = ui.Hex("#5a7fe6")
		col.accentText = ui.Hex("#ffffff")
		col.accentSoft = ui.RGBA(107, 147, 255, 0.14)
		col.accentBorder = ui.RGBA(107, 147, 255, 0.35)
		col.gold = ui.Hex("#f0a33a")
		col.red = ui.Hex("#ff6961")
		col.green = ui.Hex("#4fd17a")
		col.blue = col.accent
		col.shadow = ui.RGBA(0, 0, 0, 0.35)
		col.shadow2 = ui.RGBA(0, 0, 0, 0.20)
		col.floatShadow = ui.RGBA(0, 0, 0, 0.45)
		col.control = ui.Hex("#404047")
		col.controlHover = ui.Hex("#4a4a52")
		col.rowSelected = ui.RGBA(255, 255, 255, 0.10)
		col.rowSelBorder = ui.RGBA(255, 255, 255, 0.06)
		col.rowShadow = ui.RGBA(0, 0, 0, 0.25)
		col.fieldStroke = ui.RGBA(255, 255, 255, 0.13)
		col.sheetHover = ui.RGBA(255, 255, 255, 0.05)
		col.segThumb = ui.RGBA(255, 255, 255, 0.20)
		col.edge = ui.RGBA(255, 255, 255, 0.09)
	} else {
		col.bg = ui.Hex(w.light[0])
		col.sidebar = ui.Hex("#eceef4")
		col.card = ui.Hex("#ffffff")
		col.cardAlt = ui.Hex("#f4f3f6")
		col.chip = ui.RGBA(0, 0, 0, 0.045)
		col.cardHover = ui.RGBA(0, 0, 0, 0.04)
		col.border = ui.RGBA(0, 0, 0, 0.07)
		col.text = ui.Hex("#1d1d1f")
		col.muted = ui.Hex("#6e6e78")
		col.faint = ui.Hex("#a3a3ad")
		col.accent = ui.Hex("#3b6cf6")
		col.accentHover = ui.Hex("#2f5bd8")
		col.accentPressed = ui.Hex("#2448b0")
		col.accentText = ui.Hex("#ffffff")
		col.accentSoft = ui.RGBA(59, 108, 246, 0.10)
		col.accentBorder = ui.RGBA(59, 108, 246, 0.28)
		col.gold = ui.Hex("#c27b0f")
		col.red = ui.Hex("#d93a35")
		col.green = ui.Hex("#22863a")
		col.blue = col.accent
		col.shadow = ui.RGBA(0, 0, 0, 0.07)
		col.shadow2 = ui.RGBA(0, 0, 0, 0.035)
		col.floatShadow = ui.RGBA(20, 20, 40, 0.14)
		col.control = ui.Hex("#ffffff")
		col.controlHover = ui.Hex("#f7f7f9")
		col.rowSelected = ui.Hex("#ffffff")
		col.rowSelBorder = ui.RGBA(0, 0, 0, 0.05)
		col.rowShadow = ui.RGBA(20, 20, 50, 0.08)
		col.fieldStroke = ui.RGBA(0, 0, 0, 0.13)
		col.sheetHover = ui.RGBA(255, 255, 255, 0.55)
		col.segThumb = ui.Hex("#ffffff")
		col.edge = ui.Transparent
	}

	// Surfaces stand out by fill and shadow, not by outlines around every
	// control.
	t := *c.Theme()
	t.Dark = dark
	t.Background = col.bg
	t.Surface = col.control
	t.SurfaceHover = col.controlHover
	t.SurfacePressed = ui.Hex("#ececf0")
	if dark {
		t.SurfacePressed = ui.Hex("#4f4f58")
	}
	t.Border = ui.Transparent
	t.Font = platformUIFont()
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
	if dark {
		t.Selection = col.accent.Alpha(0.35)
		t.Focus = col.accent.Alpha(0.65)
		t.Scrollbar = ui.RGBA(255, 255, 255, 0.22)
	}
	t.ScrollbarWidth = 7
	t.Radius = 8
	t.Spacing = 4
	c.SetTheme(&t)
	return col
}

// wash is a window's wash of color: a base gradient and the light that
// comes over it from the top right.
type wash struct {
	id, name            string
	light, dark         []string
	lightGlow, darkGlow ui.Color
}

var backgroundWashes = []wash{
	{
		id: "rex", name: "花瓣",
		light:     []string{"#ebe0f3", "#f3e7f1", "#f7eee4", "#f6efcf"},
		dark:      []string{"#17141f", "#191620", "#1b1916"},
		lightGlow: ui.RGBA(247, 220, 241, 0.75),
		darkGlow:  ui.RGBA(120, 70, 120, 0.22),
	},
	{
		id: "mist", name: "薄雾",
		light:     []string{"#e3e9f4", "#ebeff6", "#f1f2f3", "#f2f1ec"},
		dark:      []string{"#12151b", "#14171d", "#17181b"},
		lightGlow: ui.RGBA(210, 225, 248, 0.75),
		darkGlow:  ui.RGBA(70, 100, 150, 0.22),
	},
	{
		id: "meadow", name: "草甸",
		light:     []string{"#e2eee5", "#ebf2e7", "#f3f2e2", "#f5f0d6"},
		dark:      []string{"#111814", "#141a16", "#181a14"},
		lightGlow: ui.RGBA(214, 240, 222, 0.75),
		darkGlow:  ui.RGBA(70, 130, 95, 0.20),
	},
	{
		id: "dusk", name: "黄昏",
		light:     []string{"#e4dbf3", "#efdeee", "#f6e3dc", "#f7e6cf"},
		dark:      []string{"#191321", "#1e161e", "#221a17"},
		lightGlow: ui.RGBA(246, 206, 222, 0.80),
		darkGlow:  ui.RGBA(150, 80, 110, 0.25),
	},
	{
		id: "plain", name: "纯色",
		light:     []string{"#eceef4", "#eceef4", "#eceef4"},
		dark:      []string{"#141418", "#141418", "#141418"},
		lightGlow: ui.RGBA(0, 0, 0, 0),
		darkGlow:  ui.RGBA(0, 0, 0, 0),
	},
}

func washByID(id string) wash {
	for _, w := range backgroundWashes {
		if w.id == id {
			return w
		}
	}
	return backgroundWashes[0]
}

func validBackground(id string) bool {
	for _, w := range backgroundWashes {
		if w.id == id {
			return true
		}
	}
	return false
}

func validAppearance(value string) bool {
	return value == "light" || value == "dark" || value == "system"
}

// platformUIFont returns the families to name on Windows, whose system UI
// font leaves CJK text to a fallback; macOS and Linux draw with their own
// interface font, and their own fallback for other scripts, when it is "".
func platformUIFont() string {
	if runtime.GOOS == "windows" {
		return "Microsoft YaHei UI, Noto Sans SC"
	}
	return ""
}

func themeSource(value string) mygo.ThemeSource {
	switch value {
	case "dark":
		return mygo.ThemeDark
	case "system":
		return mygo.ThemeSystem
	default:
		return mygo.ThemeLight
	}
}

func (w wash) backgroundCSS() string {
	return "light-dark(" + w.light[0] + ", " + w.dark[0] + ")"
}

func (w wash) gradients(dark bool) (ui.LinearGradient, ui.LinearGradient) {
	hexes, glow := w.light, w.lightGlow
	if dark {
		hexes, glow = w.dark, w.darkGlow
	}
	stops := make([]ui.GradientStop, len(hexes))
	for i, hex := range hexes {
		at := float32(0)
		if len(hexes) > 1 {
			at = float32(i) / float32(len(hexes)-1)
		}
		stops[i] = ui.GradientStop{Color: ui.Hex(hex), At: at}
	}
	base := ui.LinearGradient{Angle: 180, Oklab: true, Stops: stops}
	light := ui.LinearGradient{
		Angle: 225,
		Oklab: true,
		Stops: []ui.GradientStop{
			{Color: glow, At: 0},
			{Color: glow.Alpha(0), At: 0.5},
		},
	}
	return base, light
}

func (a *app) backdrop(dark bool) (ui.LinearGradient, ui.LinearGradient) {
	return washByID(a.background).gradients(dark)
}

func (a *app) backgroundCSS() string {
	return washByID(a.background).backgroundCSS()
}
