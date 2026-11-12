package fsutil

import (
	"errors"
	"path/filepath"
	"strings"
)

// RenderNameTemplate expands {artist}, {album} and {title} into a relative
// path without extension. Slashes separate directories, and an empty field
// drops its segment instead of leaving stray separators behind.
func RenderNameTemplate(template, artist, album, title string) (string, error) {
	template = strings.ReplaceAll(template, "\\", "/")
	replacer := strings.NewReplacer(
		"{artist}", artist,
		"{album}", album,
		"{title}", title,
	)
	var parts []string
	for _, raw := range strings.Split(template, "/") {
		part := replacer.Replace(raw)
		part = strings.Join(strings.Fields(part), " ")
		part = strings.Trim(part, " -")
		if part == "" {
			continue
		}
		parts = append(parts, SanitizeFilename(part))
	}
	if len(parts) == 0 {
		return "", errors.New("命名模板没有有效内容")
	}
	return filepath.Join(parts...), nil
}
