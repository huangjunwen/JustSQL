package lex

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	// key[:val]
	kv_re     *regexp.Regexp = regexp.MustCompile(`^([A-z][0-9A-z_]*)(:(("[^"\\]*(?:\\.[^"\\]*)*")|([^\s:"]+)))?\s+`)
	escape_re *regexp.Regexp = regexp.MustCompile(`\\.`)
)

func parseKV(src string) func() (string, string, error) {
	remain := append([]byte(strings.TrimSpace(src)), ' ')
	return func() (string, string, error) {
		// drained
		if len(remain) == 0 {
			return "", "", nil
		}

		m := kv_re.FindSubmatch(remain)
		if m == nil {
			return "", "", fmt.Errorf("Illegal kv format near: %q", string(remain))
		}

		l := len(m[0])
		k := m[1]
		v := m[3]

		// if it is quoted. Unquote it, see: https://golang.org/ref/spec#Rune_literals
		if len(m[4]) != 0 {
			v = escape_re.ReplaceAllFunc(v[1:len(v)-1], func(x []byte) []byte {
				switch c := x[1]; c {
				default:
					return []byte{c}
				case 'a':
					return []byte{'\a'}
				case 'b':
					return []byte{'\b'}
				case 'f':
					return []byte{'\f'}
				case 'n':
					return []byte{'\n'}
				case 'r':
					return []byte{'\r'}
				case 't':
					return []byte{'\t'}
				case 'v':
					return []byte{'\v'}
				}
			})
		}

		remain = remain[l:]
		return string(k), string(v), nil
	}
}
