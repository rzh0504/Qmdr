package main

import "github.com/egoist/mygo/ui"

func mustIcon(body string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">` + body + `</svg>`))
}

var (
	iconSearch       = mustIcon(`<circle cx="11" cy="11" r="7"/><path d="m20 20-3.8-3.8"/>`)
	iconLibrary      = mustIcon(`<path d="M9 18V5l12-2v13"/><circle cx="6" cy="18" r="3"/><circle cx="18" cy="16" r="3"/>`)
	iconQueue        = mustIcon(`<path d="M8 6h13M8 12h13M8 18h13M3 6h.01M3 12h.01M3 18h.01"/>`)
	iconSettings     = mustIcon(`<path d="M4 21v-7M4 10V3M12 21v-9M12 8V3M20 21v-5M20 12V3M1 14h6M9 8h6M17 16h6"/>`)
	iconChevronLeft  = mustIcon(`<path d="m15 18-6-6 6-6"/>`)
	iconChevronRight = mustIcon(`<path d="m9 18 6-6-6-6"/>`)
	iconChevronDown  = mustIcon(`<path d="m6 9 6 6 6-6"/>`)
	iconChevronUp    = mustIcon(`<path d="m18 15-6-6-6 6"/>`)
	iconWinMinimize  = mustIcon(`<path d="M5 12h14"/>`)
	iconWinMaximize  = mustIcon(`<rect x="5" y="5" width="14" height="14" rx="2"/>`)
	iconWinRestore   = mustIcon(`<rect x="7" y="7" width="10" height="10" rx="1.5"/><path d="M9 5h8a2 2 0 0 1 2 2v8"/>`)
	iconDownload     = mustIcon(`<path d="M12 3v12M7 10l5 5 5-5"/><path d="M5 21h14"/>`)
	iconFolder       = mustIcon(`<path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2Z"/>`)
	iconRefresh      = mustIcon(`<path d="M20 7v5h-5"/><path d="M4 17v-5h5"/><path d="M6.1 9A7 7 0 0 1 18.7 7M17.9 15A7 7 0 0 1 5.3 17"/>`)
	iconLogin        = mustIcon(`<path d="M14 3h5a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-5"/><path d="m10 17 5-5-5-5M15 12H3"/>`)
	iconExternal     = mustIcon(`<path d="M14 3h7v7M10 14 21 3"/><path d="M21 14v5a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5"/>`)
	iconCheck        = mustIcon(`<path d="m20 6-11 11-5-5"/>`)
	iconX            = mustIcon(`<path d="M18 6 6 18M6 6l12 12"/>`)
	iconClock        = mustIcon(`<circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/>`)
	iconAlert        = mustIcon(`<circle cx="12" cy="12" r="9"/><path d="M12 8v5M12 16h.01"/>`)
	iconTrash        = mustIcon(`<path d="M3 6h18M8 6V4h8v2M19 6l-1 15H6L5 6"/><path d="M10 11v6M14 11v6"/>`)
	iconMusic        = mustIcon(`<path d="M9 18V5l12-2v13"/><circle cx="6" cy="18" r="3"/><circle cx="18" cy="16" r="3"/>`)
	iconUser         = mustIcon(`<path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/>`)
	iconKey          = mustIcon(`<circle cx="8" cy="15" r="4"/><path d="m21 2-9.5 9.5M16 7l3 3 3-3-3-3"/>`)
	iconPlay         = mustIcon(`<path d="M8 5v14l11-7Z"/>`)
	iconSparkles     = mustIcon(`<path d="m12 3 1.8 4.7L18.5 9.5l-4.7 1.8L12 16l-1.8-4.7L5.5 9.5l4.7-1.8Z"/><path d="m19 15 .8 2.2L22 18l-2.2.8L19 21l-.8-2.2L16 18l2.2-.8Z"/>`)
	iconMore         = mustIcon(`<circle cx="5" cy="12" r="1"/><circle cx="12" cy="12" r="1"/><circle cx="19" cy="12" r="1"/>`)
	iconDisc         = mustIcon(`<circle cx="12" cy="12" r="9"/><circle cx="12" cy="12" r="2.5"/><path d="M12 3v3M12 18v3M3 12h3M18 12h3"/>`)
	iconGauge        = mustIcon(`<path d="M4.5 19a9 9 0 1 1 15 0"/><path d="m12 13 4-4"/><circle cx="12" cy="13" r="1"/>`)
	iconHeart        = mustIcon(`<path d="M20.8 4.6a5.5 5.5 0 0 0-7.8 0L12 5.7l-1.1-1.1a5.5 5.5 0 0 0-7.8 7.8L12 21l8.8-8.6a5.5 5.5 0 0 0 0-7.8Z"/>`)
	iconPlus         = mustIcon(`<path d="M12 5v14M5 12h14"/>`)
	iconWifi         = mustIcon(`<path d="M5 12.5a10 10 0 0 1 14 0M8.5 16a5.5 5.5 0 0 1 7 0M12 20h.01"/>`)
)
