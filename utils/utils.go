package utils

import (
	"github.com/pingcap/tidb/model"
	"regexp"
	"strings"
	"unicode"
)

var ident_re *regexp.Regexp = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)

// Is it a valid identifier?
func IsIdent(s string) bool {
	return ident_re.MatchString(s)
}

var word_re *regexp.Regexp = regexp.MustCompile(`[^A-Za-z]*([A-Za-z])([A-Za-z0-9]*)`)

// Convert a string to pascal case. Example: "pascal_case" -> "PascalCase"
func PascalCase(s string) string {
	parts := []string{}
	for _, m := range word_re.FindAllStringSubmatch(s, -1) {
		parts = append(parts, strings.ToUpper(m[1]), m[2])
	}
	return strings.Join(parts, "")
}

// Convert a string to camel case. Example: "camel_case" -> "camelCase"
func CamelCase(s string) string {
	b := []byte(PascalCase(s))
	b[0] = byte(unicode.ToLower(rune(b[0])))
	return string(b)
}

// String and its variants. (CamelCase/...)
type Str struct {
	O          string // hello_world
	CamelCase  string // helloWorld
	PascalCase string // HelloWorld
}

func (s Str) String() string {
	return s.PascalCase
}

func NewStr(s string) Str {
	return Str{
		O:          s,
		CamelCase:  CamelCase(s),
		PascalCase: PascalCase(s),
	}
}

func NewStrFromCIStr(ci model.CIStr) Str {
	return NewStr(ci.O)
}
