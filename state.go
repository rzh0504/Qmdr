package main

import (
	"context"
	"sync"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/rzh0504/qmdr/internal/history"
)

// searchState is the search page: the query, whether a fetch is running,
// and its result or error.
type searchState struct {
	slot asyncSlot

	query    string
	loading  bool
	searched bool
	err      string
	results  []Song
}

// playlistState is the playlists page: the list on the left, the songs of
// the selected one on the right, and the song selection between them.
type playlistState struct {
	slot      asyncSlot
	songsSlot asyncSlot

	owner        string
	loading      bool
	refreshing   bool
	loadingSongs bool
	err          string
	notice       string

	items    []Playlist
	selected int
	songs    []Song

	selecting     bool
	selectedMids  map[string]bool
	pressKey      string
	pressSeq      int
	pressTimer    *time.Timer
	suppressClick bool

	musicIDOpen  bool
	musicIDInput string
	musicIDErr   string
}

// resetFetches invalidates a playlists fetch and a songs fetch in flight,
// as a login or owner switch does.
func (s *playlistState) resetFetches() {
	s.slot.reset()
	s.songsSlot.reset()
}

// queueState is the download queue: pending songs, the running items, and
// the identity of the run for cancelling it or one task of it.
type queueState struct {
	pending     []Song
	items       []QueueItem
	filter      int
	seq         int
	downloading bool
	runID       int
	cancel      context.CancelFunc

	taskMu      sync.Mutex
	taskCancels map[int]context.CancelFunc
}

// contextMenuState is the app's own context menu, drawn with the smooth
// corners the system menu cannot take.
type contextMenuState struct {
	open  bool
	x, y  float32
	items []menuAction
}

type menuAction struct {
	label     string
	disabled  bool
	separator bool
	run       func()
}

// historyState is the download history page and its store handle.
type historyState struct {
	store      *history.Store
	items      []history.Record
	backlog    []history.Record
	logBacklog []LogEntry
	count      int
	bytes      int64
	err        string
	loading    bool
}

// coverState caches fetched cover bitmaps and throttles their fetches.
type coverState struct {
	cache   map[string]coverEntry
	loading map[string]bool
	sem     chan struct{}
}

// loginState is the account chip and the QR login dialog.
type loginState struct {
	loggedIn  bool
	userID    string
	loginType string

	qrOpen   bool
	qrFailed bool
	qrSeq    int
	qrType   string
	qrStatus string
	qrBitmap *ui.Bitmap
	qrCancel context.CancelFunc
}
