package main

import "github.com/egoist/mygo/ui"

// modal draws a dialog in the app's own look: a dimmed backdrop and a
// smooth panel with a title and close button, around body. close dismisses
// it, for the body's own buttons.
func (a *app) modal(c *ui.Context, col colors, open *bool, title string, width float32, body func(close func())) {
	ui.DialogBase(c, open, func(backdrop, panel ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, 0.4))
		panel.Width(width).Padding(18)
		smoothPanel(panel, col, 16, col.card, false)
		panel.Shadow(0, 16, 40, 0, col.floatShadow)
		close := func() {
			*open = false
			a.refresh()
		}
		panel.Children(func() {
			ui.Column(c).FillWidth().Gap(14).Children(func() {
				ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
					ui.Text(c, title).FontSize(15).FontWeight(600).TextColor(col.text).Grow(1)
					iconButton(c, col, iconX, "关闭", close)
				})
				body(close)
			})
		})
	})
}
