package widgets

import (
	"regexp"
)

func bound(i int, t []string) int {
	switch {
	case len(t) == 0:
		return -1
	case i < 0:
		return 0
	case i >= len(t):
		return len(t) - 1
	default:
		return i
	}
}

var reStyle = regexp.MustCompile(`\[([^\]]+)\]\([^)]*\)`)

func clearStyles(s string) string {
	return reStyle.ReplaceAllString(s, "$1")
}
