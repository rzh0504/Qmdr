package main

import (
	"bytes"
	_ "embed"

	"github.com/egoist/mygo/ui"
)

//go:embed assets/icons/QMDR.png
var appIconPNG []byte

//go:embed assets/icons/Qmdr_badge.svg
var qmdrBadgeSVG []byte

// qmdrBadge parses the wordmark with currentColor fills, so it follows the
// theme (dark ink on light, white on dark) via TextColor.
var qmdrBadge = ui.MustParseSVG(bytes.ReplaceAll(qmdrBadgeSVG,
	[]byte(`fill="white"`), []byte(`fill="currentColor"`)))
