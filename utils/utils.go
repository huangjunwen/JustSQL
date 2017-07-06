package utils

import (
	"regexp"
)

var (
	ident_re *regexp.Regexp = regexp.MustCompile(`^[A-z_][A-z0-9_]*$`)
)

// Is it a valid identifier?
func IsIdent(s string) bool {
	return ident_re.MatchString(s)
}
