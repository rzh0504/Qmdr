package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/rzh0504/qmdr/internal/fsutil"
)

const pendingStateVersion = 1

type pendingState struct {
	Version int    `json:"version"`
	Songs   []Song `json:"songs"`
}

func pendingStatePath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "Qmdr", "pending.json")
}

func loadPending() []Song {
	return loadPendingFile(pendingStatePath())
}

func loadPendingFile(path string) []Song {
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var state pendingState
	if err := json.Unmarshal(data, &state); err != nil || state.Version != pendingStateVersion {
		return nil
	}
	return normalizePendingSongs(state.Songs)
}

func (a *app) savePending() {
	path := pendingStatePath()
	if path == "" {
		a.setToast("找不到配置目录")
		return
	}
	state := pendingState{
		Version: pendingStateVersion,
		Songs:   normalizePendingSongs(a.queue.pending),
	}
	a.dispatchDisk(func() {
		if err := savePendingFile(path, state.Songs); err != nil {
			a.update(func() {
				a.toast = "待下载队列保存失败：" + friendlyError(err)
			})
		}
	})
}

func savePendingFile(path string, songs []Song) error {
	state := pendingState{
		Version: pendingStateVersion,
		Songs:   normalizePendingSongs(songs),
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(path, data, 0o644)
}

func normalizePendingSongs(songs []Song) []Song {
	out := make([]Song, 0, len(songs))
	seen := make(map[string]bool, len(songs))
	for _, song := range songs {
		mid := strings.TrimSpace(song.Mid)
		if mid == "" || seen[mid] {
			continue
		}
		seen[mid] = true
		out = append(out, song)
	}
	return out
}
