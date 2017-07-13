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

func testPascalCase(t *testing.T, s string, expect string) {
	r := PascalCase(s)
	if r != expect {
		t.Errorf("%q: %q != %q\n", s, r, expect)
	}
}

func TestCamelCase(t *testing.T) {
	testCamelCase(t, "hello", "hello")
	testCamelCase(t, "hello   world", "helloWorld")
	testCamelCase(t, "  hello   world", "helloWorld")
	testCamelCase(t, "_hello___world", "helloWorld")
}

func TestPascalCase(t *testing.T) {
	testPascalCase(t, "hello", "Hello")
	testPascalCase(t, "hello   world", "HelloWorld")
	testPascalCase(t, "  hello   world", "HelloWorld")
	testPascalCase(t, "_hello___world", "HelloWorld")
}
