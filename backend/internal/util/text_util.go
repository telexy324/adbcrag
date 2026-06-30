package util

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

var headingPattern = regexp.MustCompile(`(?m)^(#{1,3})\s+(.+)$`)

func NormalizeText(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	return strings.TrimSpace(text)
}

func EstimateTokenCount(text string) int {
	return utf8.RuneCountInString(text)
}

func MarkdownHeadings(text string) []string {
	matches := headingPattern.FindAllStringSubmatch(text, -1)
	sections := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) >= 3 {
			sections = append(sections, strings.TrimSpace(match[2]))
		}
	}
	return sections
}
