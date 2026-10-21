package fsutil

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// SanitizeFilename makes name safe for a file on Windows: it replaces
// reserved characters, trims trailing dots and spaces the system drops,
// escapes device names, and caps the length.
func SanitizeFilename(name string) string {
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

// SafePath joins name to base and rejects a result that escapes base.
func SafePath(base, name string) (string, error) {
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
