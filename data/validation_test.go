package data

import "testing"

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
}
