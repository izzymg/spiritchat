package data

import (
	"html"
	"regexp"
	"strings"
)

// Replace 3 or more newlines, including possible spaces
var newlines = regexp.MustCompile("(\n\\s*){3,}")

// Replace all carriage returns with normal newlines
var carriageReturns = regexp.MustCompile("\r\n")

/*
CheckContent validates a post's contents, returning the content sanitized as
the first argument, or a human-readable error message as the second. */
func CheckContent(content string) (string, string) {
	content = strings.TrimSpace(
		html.EscapeString(
			strings.ToValidUTF8(
				content,
				"",
			),
		),
	)
	content = carriageReturns.ReplaceAllString(content, "\n")
	content = newlines.ReplaceAllString(content, "\n")
	if len(content) < minContentLen || len(content) > maxContentLen {
		return "", InvalidContentLen
	}
	return content, ""
}
