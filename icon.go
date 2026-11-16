package main

import (
	_ "embed"

	"github.com/egoist/mygo/ui"
)

//go:embed assets/icons/QMDR.png
var appIconPNG []byte

//go:embed assets/icons/Qmdr_badge.svg
var qmdrBadgeSVG []byte

// qmdrBadge is the sidebar wordmark lockup (white artwork, drawn on an
// accent plate).
var qmdrBadge = ui.MustParseSVG(qmdrBadgeSVG)
