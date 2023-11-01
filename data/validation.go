package data

import (
	"fmt"
	"html"
	"regexp"
	"strings"
)

const maxContentLen = 300

const minContentLen = 2

// InvalidContentLen is a message describing an invalid post content length.
var InvalidContentLen = fmt.Sprintf(
	"Content must be between %d and %d characters",
	minContentLen,
	maxContentLen,
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
	if len([]rune(content)) < minContentLen || len([]rune(content)) > maxContentLen {
		return "", InvalidContentLen
	}
	return content, ""
}
