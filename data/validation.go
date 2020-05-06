package data

import (
	"html"
	"strings"
)

/*
CheckContent validates a post's contents, returning the content sanitized as
the first argument, or a human-readable error message as the second. */
func CheckContent(content string) (string, string) {
	content = strings.TrimSpace(html.EscapeString(content))
	if len(content) < minContentLen || len(content) > maxContentLen {
		return "", InvalidContentLen
	}
	return content, ""
}
