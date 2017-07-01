package lex

import (
	"errors"
	"strings"
)

// Use comments inside SQL to store some extra information.
type Comment struct {
	// Offset and length in original source (including '/*' '*/' '#' '--' '\n')
	Offset, Length int
	// Stripped comment content
	Content string
}

// Scan SQL comments ('#...', '--...', '/*...*/').
func ScanComment(src string) ([]Comment, error) {
	i := 0
	l := len(src)
	comments := make([]Comment, 0)

	// Find the next char c, return found or not. 'i' will be set
	// to the next char of c.
	find := func(c byte) bool {
		for ; i < l; i += 1 {
			if src[i] == c {
				i += 1
				return true
			}
		}
		return false
	}

	addComment := func(offset int, length int, content string) {
		comments = append(comments, Comment{
			Offset:  offset,
			Length:  length,
			Content: strings.TrimSpace(content),
		})
	}

	for i < l {
		switch c := src[i]; c {
		default:
			i += 1
			continue

		// # ... \n
		case '#':
			offset := i
			find('\n')
			addComment(offset, i-offset, src[offset+1:i])

		// -- ... \n
		case '-':
			if i+1 >= l || src[i+1] != '-' {
				i += 1
				continue
			}
			offset := i
			find('\n')
			addComment(offset, i-offset, src[offset+2:i])

		// /* ... */
		case '/':
			if i+1 >= l || src[i+1] != '*' {
				i += 1
				continue
			}
			offset := i
			i += 2
			for {
				if !find('*') || i >= l {
					return nil, errors.New("Missing '*/'")
				}
				if src[i] == '/' {
					i += 1
					break
				}
			}
			addComment(offset, i-offset, src[offset+2:i-2])
		}

	}
	return comments, nil
}
