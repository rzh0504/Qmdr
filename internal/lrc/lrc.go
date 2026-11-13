package lrc

import (
	"regexp"
	"strconv"
	"strings"
)

var timestampPattern = regexp.MustCompile(`^\[(\d{1,3}):(\d{2})(?:[.:](\d{1,3}))?\]`)

// translatedLine is one line of the translation file.
type translatedLine struct {
	ms   int
	text string
}

func parseTimestamp(raw string) (int, bool) {
	match := timestampPattern.FindStringSubmatch(raw)
	if match == nil {
		return 0, false
	}
	minutes, _ := strconv.Atoi(match[1])
	seconds, _ := strconv.Atoi(match[2])
	ms := (minutes*60 + seconds) * 1000
	switch len(match[3]) {
	case 1:
		value, _ := strconv.Atoi(match[3])
		ms += value * 100
	case 2:
		value, _ := strconv.Atoi(match[3])
		ms += value * 10
	case 3:
		value, _ := strconv.Atoi(match[3])
		ms += value
	}
	return ms, true
}

// parseTranslation keeps only translated lines that carry actual text; the
// service marks untranslated lines with "//".
func parseTranslation(translation string) []translatedLine {
	var lines []translatedLine
	for _, raw := range strings.Split(translation, "\n") {
		raw = strings.TrimRight(raw, "\r")
		ms, ok := parseTimestamp(raw)
		if !ok {
			continue
		}
		text := strings.TrimSpace(raw[len(timestampPattern.FindString(raw)):])
		if text == "" || text == "//" {
			continue
		}
		lines = append(lines, translatedLine{ms: ms, text: text})
	}
	return lines
}

// translationFor finds the translated line at ms, tolerating the small
// offset the service sometimes emits between lyric and translation times.
func translationFor(lines []translatedLine, ms int) (string, bool) {
	const tolerance = 100
	bestDiff := tolerance + 1
	best := ""
	for _, line := range lines {
		diff := line.ms - ms
		if diff < 0 {
			diff = -diff
		}
		if diff < bestDiff {
			bestDiff = diff
			best = line.text
		}
	}
	return best, best != ""
}

// Merge appends each translated line under its original line, reusing the
// original timestamp so players show both lines together. A lyric without
// usable translations is returned unchanged.
func Merge(original, translation string) string {
	lines := parseTranslation(translation)
	if len(lines) == 0 {
		return original
	}
	var out strings.Builder
	for _, raw := range strings.Split(original, "\n") {
		line := strings.TrimRight(raw, "\r")
		out.WriteString(line)
		out.WriteString("\n")
		ms, ok := parseTimestamp(line)
		if !ok {
			continue
		}
		text, ok := translationFor(lines, ms)
		if !ok {
			continue
		}
		out.WriteString(timestampPattern.FindString(line))
		out.WriteString(text)
		out.WriteString("\n")
	}
	return strings.TrimSuffix(out.String(), "\n")
}
