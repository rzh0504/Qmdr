package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"

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
		dest, err := safeDownloadPath(dir, name)
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

func coverSizeValue(value string) int {
	value = strings.TrimSpace(strings.TrimSuffix(strings.ToLower(value), "px"))
	size, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 800
	}
	switch size {
	case 150, 300, 500, 800:
		return size
	default:
		return 800
	}
}

func fileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

func formatFileSize(size int64) string {
	switch {
	case size <= 0:
		return ""
	case size >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(size)/float64(1<<30))
	case size >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(size)/float64(1<<20))
	case size >= 1<<10:
		return fmt.Sprintf("%.0f KB", float64(size)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", size)
	}
}

func sanitizeFilename(name string) string {
	name = strings.Map(func(r rune) rune {
		switch r {
		case '\\', '/', ':', '*', '?', '"', '<', '>', '|':
			return '_'
		}
		if r < 32 {
			return -1
		}
		return r
	}, name)
	// Windows drops trailing dots and spaces when it saves the file, so
	// trim them here to keep the stored path equal to the real one;
	// leading whitespace goes too, as the reference backend trims it.
	name = strings.TrimRight(strings.TrimSpace(name), " .")
	if name == "" {
		name = "未知歌曲"
	}
	base := strings.ToUpper(strings.TrimSuffix(name, filepath.Ext(name)))
	switch base {
	case "CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
		name = "_" + name
	}
	runes := []rune(name)
	const maxRunes = 160
	if len(runes) > maxRunes {
		name = strings.TrimRight(strings.TrimSpace(string(runes[:maxRunes])), " .")
		if name == "" {
			name = "未知歌曲"
		}
	}
	return name
}

func safeDownloadPath(base, name string) (string, error) {
	if strings.TrimSpace(base) == "" {
		return "", errors.New("未设置下载目录")
	}
	absBase, err := filepath.Abs(base)
	if err != nil {
		return "", err
	}
	dest := filepath.Join(absBase, name)
	rel, err := filepath.Rel(absBase, dest)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", errors.New("无效的下载路径")
	}
	return dest, nil
}
