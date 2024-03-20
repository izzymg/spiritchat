package validation

import (
	"errors"
	"fmt"
	"html"
	"regexp"
	"strings"
)

const maxContentLen = 300

const minContentLen = 2

const minSubjectLen = 5
const maxSubjectLen = 80

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

var ErrInvalidEmail = errors.New("that doesn't look like an email")
var ErrInvalidUsername = errors.New("username required, > 3 characters")
var ErrInvalidPassword = errors.New("password required")

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
ValidateReplySubject sanitizes a subject and returns the content or a human-readable error message
*/
func ValidateReplySubject(subject string, isThread bool) (string, error) {
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
ValidateReplyContent validates a post's contents, returning the content sanitized as
the first argument, or a human-readable error message as the second.
*/
func ValidateReplyContent(content string) (string, error) {
	content = sanitize(content)
	content = carriageReturns.ReplaceAllString(content, "\n")
	content = manyNewlines.ReplaceAllString(content, "\n")
	if len([]rune(content)) < minContentLen || len([]rune(content)) > maxContentLen {
		return "", ErrInvalidContentLen
	}
	return content, nil
}

/*
ValidateEmail is a very basic email check. Returns human readable error if issues found.
*/
func ValidateEmail(email string) (string, error) {
	if len(email) < 1 {
		return "", ErrInvalidEmail
	}
	if !strings.ContainsRune(email, '@') {
		return "", ErrInvalidEmail
	}
	return email, nil
}

// ValidateUsername does a length check. Returns human readable errors if issues found.
func ValidateUsername(username string) (string, error) {
	if len(username) < 3 {
		return "", ErrInvalidUsername
	}
	return username, nil
}

// ValidatePassword does a length check. Returns human readable errors if issues found.
func ValidatePassword(password string) (string, error) {
	if len(password) < 1 {
		return "", ErrInvalidPassword
	}
	return password, nil
}
