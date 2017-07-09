package utils

import (
	"regexp"
	"strings"
)

var ident_re *regexp.Regexp = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// Is it a valid identifier?
func IsIdent(s string) bool {
	return ident_re.MatchString(s)
}

var word_re *regexp.Regexp = regexp.MustCompile(`[^A-Za-z]*([A-Za-z])([A-Za-z0-9]*)`)

// Convert a string to camel case.
func CamelCase(s string) string {
	parts := []string{}
	for _, m := range word_re.FindAllStringSubmatch(s, -1) {
		parts = append(parts, strings.ToUpper(m[1]), m[2])
	}
	return strings.Join(parts, "")
}
