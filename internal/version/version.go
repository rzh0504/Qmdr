package version

import (
	"strconv"
	"strings"
)

// Compare compares two dotted version strings, ignoring a leading v and
// any pre-release suffix: it returns -1, 0 or 1 as a is older, equal or
// newer than b.
func Compare(a, b string) int {
	av, bv := parts(a), parts(b)
	n := max(len(av), len(bv))
	for i := 0; i < n; i++ {
		var x, y int
		if i < len(av) {
			x = av[i]
		}
		if i < len(bv) {
			y = bv[i]
		}
		if x != y {
			if x > y {
				return 1
			}
			return -1
		}
	}
	return 0
}

func parts(v string) []int {
	v = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(v, "v"), "V"))
	if i := strings.IndexAny(v, "-+ "); i >= 0 {
		v = v[:i]
	}
	if v == "" {
		return nil
	}
	fields := strings.Split(v, ".")
	parts := make([]int, len(fields))
	for i, field := range fields {
		n, err := strconv.Atoi(field)
		if err != nil {
			n = 0
		}
		parts[i] = n
	}
	return parts
}
