package utils

import (
	"testing"
)

func testCamelCase(t *testing.T, s string, expect string) {
	r := CamelCase(s)
	if r != expect {
		t.Errorf("%q: %q != %q\n", s, r, expect)
	}
}

func TestCamelCase(t *testing.T) {
	testCamelCase(t, "hello", "Hello")
	testCamelCase(t, "hello   world", "HelloWorld")
	testCamelCase(t, "  hello   world", "HelloWorld")
	testCamelCase(t, "_hello___world", "HelloWorld")
}
