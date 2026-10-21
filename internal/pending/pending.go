package pending

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/rzh0504/qmdr/internal/fsutil"
	"github.com/rzh0504/qmdr/internal/model"
)

const stateVersion = 1

type State struct {
	Version int          `json:"version"`
	Songs   []model.Song `json:"songs"`
}

func Path() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "Qmdr", "pending.json")
}

func Load() []model.Song { return LoadFile(Path()) }

func LoadFile(path string) []model.Song {
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil || state.Version != stateVersion {
		return nil
	}
	return Normalize(state.Songs)
}

func SaveFile(path string, songs []model.Song) error {
	state := State{Version: stateVersion, Songs: Normalize(songs)}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(path, data, 0o644)
}

// Normalize drops songs without a MID and keeps the first of duplicate
// MIDs, preserving order.
func Normalize(songs []model.Song) []model.Song {
	out := make([]model.Song, 0, len(songs))
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
