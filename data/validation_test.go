package data

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

	_, errStr := CheckSubject(onMin)
	if len(errStr) > 0 {
		t.Fatal("expected no err string")
	}

	_, errStr = CheckSubject(belowMin)
	if len(errStr) == 0 {
		t.Fatal("expected an err string")
	}

	_, errStr = CheckSubject(onMax)
	if len(errStr) > 0 {
		t.Fatal("expected no err string")
	}

	_, errStr = CheckSubject(aboveMax)
	if len(errStr) == 0 {
		t.Fatal("expected an err string")
	}

	_, errStr = CheckSubject("   a   ")
	if len(errStr) == 0 {
		t.Fatal("expected an err string")
	}

	ret, errStr := CheckSubject("\rxxerwz\r \r\n  \r")
	if len(errStr) > 0 {
		t.Fatal("expected no err string")
	}
	if strings.ContainsAny(ret, "\r\n") {
		t.Fatal("expected no return chars")
	}
	if strings.ContainsAny(ret, "\r") {
		t.Fatal("expected no return chars")
	}
	if strings.ContainsAny(ret, "\n") {
		t.Fatal("expected no newlines")
	}

	ret, errStr = CheckSubject("dog\n cat \n\n tiger \n\n\n\n\n bat")
	if len(errStr) > 0 {
		t.Fatal("Expected no err string")
	}
	if c := strings.Count(ret, "\n"); c != 0 {
		t.Fatalf("Expected 0 newlines, got %d", c)
	}
}

// Test sanitizing a post's content.
func TestCheckContent(t *testing.T) {
	onMin := genStr(minContentLen, "a")
	belowMin := genStr(minContentLen-1, "a")
	onMax := genStr(maxContentLen, "a")
	aboveMax := genStr(maxContentLen+1, "a")

	_, errStr := CheckContent(onMin)
	if len(errStr) > 0 {
		t.Fatal("Expected no err string")
	}

	_, errStr = CheckContent(belowMin)
	if len(errStr) == 0 {
		t.Fatal("Expected an err string")
	}

	_, errStr = CheckContent(onMax)
	if len(errStr) > 0 {
		t.Fatal("Expected no err string")
	}

	_, errStr = CheckContent(aboveMax)
	if len(errStr) == 0 {
		t.Fatal("Expected an err string")
	}

	_, errStr = CheckContent("   a   ")
	if len(errStr) == 0 {
		t.Fatal("Expected an err string")
	}

	ret, errStr := CheckContent("\rxxz\r \r\n  \r")
	if len(errStr) > 0 {
		t.Fatal("Expected no err string")
	}
	if strings.ContainsAny(ret, "\r\n") {
		t.Fatal("Expected no return chars")
	}
	if strings.ContainsAny(ret, "\r") {
		t.Fatal("Expected no return chars")
	}

	ret, errStr = CheckContent("dog\n cat \n\n tiger \n\n\n\n\n bat")
	if len(errStr) > 0 {
		t.Fatal("Expected no err string")
	}
	if c := strings.Count(ret, "\n"); c != 4 {
		t.Fatalf("Expected 3 newlines, got %d", c)
	}
}
