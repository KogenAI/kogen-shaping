package shape

import (
	"regexp"
	"strings"
)

var controlsFailureLocation = regexp.MustCompile(`(?:^|\s)[^\s]+:[0-9]+(?::[0-9]+)?(?:[^0-9]|$)`)

func controlsFailureMessage(message string) string {
	var lines []string
	for _, line := range strings.Split(message, "\n") {
		if line = normalizeWhitespace(line); line != "" {
			lines = append(lines, line)
		}
	}
	if len(lines) == 0 {
		return ""
	}
	first := []rune(lines[0])
	if len(first) >= 400 {
		return string(first[:400])
	}
	result := string(first)
	count := len(first)
	extra := 0
	for _, location := range []bool{true, false} {
		for _, line := range lines[1:] {
			if controlsFailureLocation.MatchString(line) != location {
				continue
			}
			if extra == 3 || count+3 >= 400 {
				return result
			}
			runes := []rune(line)
			remaining := 400 - count - 3
			if len(runes) > remaining {
				runes = runes[:remaining]
			}
			result += " | " + string(runes)
			count += 3 + len(runes)
			extra++
		}
	}
	return result
}
