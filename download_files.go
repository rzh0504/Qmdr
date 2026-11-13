package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"

	"github.com/rzh0504/qmdr/internal/fsutil"
	"github.com/rzh0504/qmdr/internal/lrc"
	"github.com/rzh0504/qmdr/internal/qqmusic"
	"github.com/rzh0504/qmdr/internal/tag"
)

type downloadTask struct {
	ID          int
	Mid         string
	Title       string
	Artist      string
	Album       string
	AlbumMid    string
	Quality     string
	Track       int
	Disc        int
	ReleaseDate string
	File        qqmusic.SongFile
}

func songFileOf(song Song) qqmusic.SongFile {
	return qqmusic.SongFile{
		MediaMid:    song.MediaMid,
		SizeMaster:  song.SizeMaster,
		SizeAtmos2:  song.SizeAtmos2,
		SizeAtmos51: song.SizeAtmos51,
		SizeFLAC:    song.SizeFLAC,
		SizeMP3320:  song.SizeMP3320,
		SizeMP3128:  song.SizeMP3128,
	}
}

type claimResult int

const (
	claimOK claimResult = iota
	claimSkip
	claimExhausted
)

type downloadClaims struct {
	mu    sync.Mutex
	paths map[string]bool
}

func newDownloadClaims() *downloadClaims {
	return &downloadClaims{paths: make(map[string]bool)}
}

func (c *downloadClaims) reserve(dir, base, ext string, overwrite bool) (string, claimResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i := 1; i <= 999; i++ {
		name := base + ext
		if i > 1 {
			name = fmt.Sprintf("%s (%d)%s", base, i, ext)
		}
		dest, err := fsutil.SafePath(dir, name)
		if err != nil {
			return "", claimExhausted
		}
		if _, err := os.Stat(dest); err == nil && !overwrite {
			return dest, claimSkip
		}
		key := strings.ToLower(dest)
		if c.paths[key] {
			continue
		}
		c.paths[key] = true
		return dest, claimOK
	}
	return "", claimExhausted
}

// cleanPartFiles removes the .part files of downloads that a previous run
// never finished; they cannot be resumed. Naming templates can put files in
// album subdirectories, so the whole tree is checked.
func cleanPartFiles(dir string) {
	_ = filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(entry.Name(), ".part") {
			return nil
		}
		if err := os.Remove(path); err != nil {
			log.Printf("remove stale part file: %v", err)
		}
		return nil
	})
}

func (a *app) writeMetadataSafely(ctx context.Context, task downloadTask, dest string, coverSize int, lyricsTrans bool) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Printf("metadata panic: %v\n%s", recovered, debug.Stack())
			err = fmt.Errorf("元数据写入异常")
		}
	}()
	return a.writeMetadata(ctx, task, dest, coverSize, lyricsTrans)
}

func (a *app) writeMetadata(ctx context.Context, task downloadTask, dest string, coverSize int, lyricsTrans bool) error {
	lyrics, _ := a.api.Lyric(ctx, task.Mid)
	lyricText := lyrics.LRC
	if lyricsTrans {
		lyricText = lrc.Merge(lyricText, lyrics.Translation)
	}
	var cover []byte
	var coverMIME string
	if task.AlbumMid != "" {
		cover, coverMIME = a.metadataCover(ctx, task.AlbumMid, coverSize)
	}
	meta, _ := a.api.SongMeta(ctx, task.Mid)
	credits, _ := a.api.SongCredits(ctx, task.Mid)
	date := task.ReleaseDate
	if date == "" {
		date = meta.ReleaseDate
	}
	return tag.Write(dest, tag.Info{
		Title:       task.Title,
		Artist:      task.Artist,
		Album:       task.Album,
		Lyrics:      lyricText,
		Cover:       cover,
		CoverMIME:   coverMIME,
		TrackNumber: task.Track,
		DiscNumber:  task.Disc,
		Date:        date,
		Genre:       meta.Genre,
		Language:    meta.Language,
		Composer:    credits.Composer,
		Lyricist:    credits.Lyricist,
	})
}

const maxCachedMetadataCovers = 64

// metadataCover fetches an album cover once per size: later songs of the
// same album reuse it, and concurrent downloads wait for the fetch already
// in flight instead of requesting the same image again.
func (a *app) metadataCover(ctx context.Context, albumMid string, size int) ([]byte, string) {
	key := albumMid + "-" + strconv.Itoa(size)
	state := &a.metaCovers
	state.mu.Lock()
	if entry, ok := state.cache[key]; ok {
		state.mu.Unlock()
		return entry.data, entry.mime
	}
	if call, ok := state.pending[key]; ok {
		state.mu.Unlock()
		select {
		case <-call.done:
			return call.data, call.mime
		case <-ctx.Done():
			return nil, ""
		}
	}
	call := &metadataCoverCall{done: make(chan struct{})}
	state.pending[key] = call
	state.mu.Unlock()

	data, mime, err := a.api.Cover(ctx, albumMid, size)
	if err == nil && len(data) > 0 {
		call.data, call.mime = data, mime
		state.mu.Lock()
		state.cache[key] = metadataCover{data: data, mime: mime}
		state.order = append(state.order, key)
		for len(state.order) > maxCachedMetadataCovers {
			oldest := state.order[0]
			state.order = state.order[1:]
			delete(state.cache, oldest)
		}
		state.mu.Unlock()
	}
	close(call.done)
	state.mu.Lock()
	delete(state.pending, key)
	state.mu.Unlock()
	return call.data, call.mime
}
