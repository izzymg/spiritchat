package serve

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

// ErrInvalidContentLen is a message describing an invalid post content length.
var ErrInvalidContentLen = fmt.Errorf(
	"content must be between %d and %d characters",
	minContentLen,
	maxContentLen,
)

var ErrInvalidSubjectLen = fmt.Errorf(
	"subject must be between %d and %d characters",
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
func checkSubject(subject string, isThread bool) (string, error) {
	// Replies should never have subjects
	if !isThread {
		return "", nil
	}

	subject = newline.ReplaceAllString(carriageReturns.ReplaceAllString(sanitize(subject), ""), "")
	runeLength := len([]rune(subject))
	if runeLength < minSubjectLen || runeLength > maxSubjectLen {
		return "", ErrInvalidSubjectLen
	}
	return subject, nil
}

/*
CheckContent validates a post's contents, returning the content sanitized as
the first argument, or a human-readable error message as the second.
*/
func checkContent(content string) (string, error) {
	content = sanitize(content)
	content = carriageReturns.ReplaceAllString(content, "\n")
	content = manyNewlines.ReplaceAllString(content, "\n")
	if len([]rune(content)) < minContentLen || len([]rune(content)) > maxContentLen {
		return "", ErrInvalidContentLen
	}
	return content, nil
}
