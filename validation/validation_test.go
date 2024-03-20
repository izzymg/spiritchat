package validation

import (
	"strings"
	"testing"
)

// Generates string of "a" n times
func genStr(n int, s string) string {
	if n == 1 {
		return s
	}
	return genStr(n-1, s+"a")
}

func TestCheckSubject(t *testing.T) {
	onMin := genStr(minSubjectLen, "a")
	belowMin := genStr(minSubjectLen-1, "a")
	onMax := genStr(maxSubjectLen, "a")
	aboveMax := genStr(maxSubjectLen+1, "a")

	ret, nil := ValidateReplySubject("bunch of stuff", false)
	if len(ret) != 0 {
		t.Errorf("expected empty subject, got %s", ret)
	}

	_, err := ValidateReplySubject(onMin, true)
	if err != nil {
		t.Error("expected no err string")
	}

	_, err = ValidateReplySubject(belowMin, true)
	if err == nil {
		t.Error("expected an err string")
	}

	_, err = ValidateReplySubject(onMax, true)
	if err != nil {
		t.Error("expected no err string")
	}

	_, err = ValidateReplySubject(aboveMax, true)
	if err == nil {
		t.Error("expected an err string")
	}

	_, err = ValidateReplySubject("   a   ", true)
	if err == nil {
		t.Error("expected an err string")
	}

	ret, err = ValidateReplySubject("\rxxerwz\r \r\n  \r", true)
	if err != nil {
		t.Error("expected no err string")
	}
	if strings.ContainsAny(ret, "\r\n") {
		t.Error("expected no return chars")
	}
	if strings.ContainsAny(ret, "\r") {
		t.Error("expected no return chars")
	}
	if strings.ContainsAny(ret, "\n") {
		t.Error("expected no newlines")
	}

	ret, err = ValidateReplySubject("dog\n cat \n\n tiger \n\n\n\n\n bat", true)
	if err != nil {
		t.Error("Expected no err string")
	}
	if c := strings.Count(ret, "\n"); c != 0 {
		t.Errorf("Expected 0 newlines, got %d", c)
	}
}

// Test sanitizing a post's content.
func TestCheckContent(t *testing.T) {
	onMin := genStr(minContentLen, "a")
	belowMin := genStr(minContentLen-1, "a")
	onMax := genStr(maxContentLen, "a")
	aboveMax := genStr(maxContentLen+1, "a")

	_, err := ValidateReplyContent(onMin)
	if err != nil {
		t.Error("Expected no err string")
	}

	_, err = ValidateReplyContent(belowMin)
	if err == nil {
		t.Error("Expected an err string")
	}

	_, err = ValidateReplyContent(onMax)
	if err != nil {
		t.Error("Expected no err string")
	}

	_, err = ValidateReplyContent(aboveMax)
	if err == nil {
		t.Error("Expected an err string")
	}

	_, err = ValidateReplyContent("   a   ")
	if err == nil {
		t.Error("Expected an err string")
	}

	ret, err := ValidateReplyContent("\rxxz\r \r\n  \r")
	if err != nil {
		t.Error("Expected no err string")
	}
	if strings.ContainsAny(ret, "\r\n") {
		t.Error("Expected no return chars")
	}
	if strings.ContainsAny(ret, "\r") {
		t.Error("Expected no return chars")
	}

	ret, err = ValidateReplyContent("dog\n cat \n\n tiger \n\n\n\n\n bat")
	if err != nil {
		t.Error("Expected no err string")
	}
	if c := strings.Count(ret, "\n"); c != 4 {
		t.Errorf("Expected 3 newlines, got %d", c)
	}
}

func TestValidateEmail(t *testing.T) {
	tests := map[string]error{
		"":             ErrInvalidEmail,
		"no at symbol": ErrInvalidEmail,
		"cat@dog.com":  nil,
	}

	for input, expectErr := range tests {
		t.Run(input, func(t *testing.T) {
			_, err := ValidateEmail(input)
			if err != expectErr {
				t.Errorf("expected %v, got %v", expectErr, err)
			}
		})
	}
}
