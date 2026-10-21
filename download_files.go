package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"

	"github.com/rzh0504/qmdr/internal/fsutil"
	"github.com/rzh0504/qmdr/internal/tag"
)

type downloadTask struct {
	ID       int
	Mid      string
	Title    string
	Artist   string
	Album    string
	AlbumMid string
	Quality  string
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
// never finished; they cannot be resumed.
func cleanPartFiles(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".part") {
			continue
		}
		if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil {
			log.Printf("remove stale part file: %v", err)
		}
	}
}

func (a *app) writeMetadataSafely(ctx context.Context, task downloadTask, dest string, coverSize int) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Printf("metadata panic: %v\n%s", recovered, debug.Stack())
			err = fmt.Errorf("元数据写入异常")
		}
	}()
	return a.writeMetadata(ctx, task, dest, coverSize)
}

func (a *app) writeMetadata(ctx context.Context, task downloadTask, dest string, coverSize int) error {
	lyrics, _ := a.api.Lyric(ctx, task.Mid)
	var cover []byte
	var coverMIME string
	if task.AlbumMid != "" {
		cover, coverMIME, _ = a.api.Cover(ctx, task.AlbumMid, coverSize)
	}
	return tag.Write(dest, tag.Info{
		Title:     task.Title,
		Artist:    task.Artist,
		Album:     task.Album,
		Lyrics:    lyrics,
		Cover:     cover,
		CoverMIME: coverMIME,
	})
}
