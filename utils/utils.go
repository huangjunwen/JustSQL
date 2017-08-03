package utils

import (
	"regexp"
	"strings"
	"unicode"
)

var (
	identRe      *regexp.Regexp = regexp.MustCompile(`[^A-Za-z]*([A-Za-z])([A-Za-z0-9]*)`)
	exactIdentRe *regexp.Regexp = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)
)

// Is it a valid identifier?
func IsIdent(s string) bool {
	return exactIdentRe.MatchString(s)
}

// Find left most identifier.
func FindIdent(s string) string {
	m := identRe.FindStringSubmatch(s)
	if m == nil {
		return ""
	}
	return m[1] + m[2]
}

// Convert a string to pascal case. Example: "pascal_case" -> "PascalCase"
func PascalCase(s string) string {
	parts := []string{}
	for _, m := range identRe.FindAllStringSubmatch(s, -1) {
		parts = append(parts, strings.ToUpper(m[1]), strings.ToLower(m[2]))
	}
	return strings.Join(parts, "")
}

// Convert a string to camel case. Example: "camel_case" -> "camelCase"
func CamelCase(s string) string {
	b := []byte(PascalCase(s))
	b[0] = byte(unicode.ToLower(rune(b[0])))
	return string(b)
}
