package data

import (
	"fmt"
	"html"
	"regexp"
	"strings"
)

const maxContentLen = 300

const minContentLen = 2

const minSubjectLen = 5
const maxSubjectLen = 80

// InvalidContentLen is a message describing an invalid post content length.
var InvalidContentLen = fmt.Sprintf(
	"Content must be between %d and %d characters",
	minContentLen,
	maxContentLen,
)

var InvalidSubjectLen = fmt.Sprintf(
	"Subject must be between %d and %d characters",
	minSubjectLen,
	maxSubjectLen,
)

// Replace 3 or more manyNewlines, including possible spaces
var manyNewlines = regexp.MustCompile("(\n\\s*){3,}")

// Replace one newline
var newline = regexp.MustCompile(`\n`)

// Replace all carriage returns with normal newlines
var carriageReturns = regexp.MustCompile("\r\n")

func sanitize(data string) string {
	return strings.TrimSpace(
		html.EscapeString(
			strings.ToValidUTF8(
				data,
				"",
			),
		),
	)
}

/*
*
CheckSubject sanitizes a subject and returns the content or a human-readable error message
*/
func CheckSubject(subject string) (string, string) {
	subject = newline.ReplaceAllString(carriageReturns.ReplaceAllString(sanitize(subject), ""), "")
	runeLength := len([]rune(subject))
	if runeLength < minSubjectLen || runeLength > maxSubjectLen {
		return "", InvalidSubjectLen
	}
	return subject, ""
}

/*
CheckContent validates a post's contents, returning the content sanitized as
the first argument, or a human-readable error message as the second.
*/
func CheckContent(content string) (string, string) {
	content = sanitize(content)
	content = carriageReturns.ReplaceAllString(content, "\n")
	content = manyNewlines.ReplaceAllString(content, "\n")
	if len([]rune(content)) < minContentLen || len([]rune(content)) > maxContentLen {
		return "", InvalidContentLen
	}
	return content, ""
}
