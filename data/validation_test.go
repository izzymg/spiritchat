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
