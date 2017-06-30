package comment

import (
	"errors"
	"strings"
)

// SQL comment
type Comment struct {
	// Source string
	Src string
	// Offset and length in original source (including '/*' '*/' '#' '--')
	Offset, Length int
	// Stripped comment content
	Content string
}

func ScanComment(src string) ([]Comment, error) {
	i := 0
	comments := make([]Comment, 0)

	// Move to the next char c
	moveTo := func(c byte) bool {
		// i should < len(s.src) at this point
		i += 1
		for ; i < len(src); i += 1 {
			if src[i] == c {
				return true
			}
		}
		return false
	}

	addComment := func(offset int, length int, content string) {
		comments = append(comments, Comment{
			Src:     src,
			Offset:  offset,
			Length:  length,
			Content: strings.TrimSpace(content),
		})
	}

	for ; i < len(src); i += 1 {
		switch c := src[i]; c {
		default:
			continue

		case '#':
			offset := i
			if moveTo('\n') {
				addComment(offset, i+1-offset, src[offset+1:i+1])
			} else {
				addComment(offset, i-offset, src[offset+1:i])
			}

		case '-':
			if i+1 >= len(src) || src[i+1] != '-' {
				continue
			}
			offset := i
			if moveTo('\n') {
				addComment(offset, i+1-offset, src[offset+2:i+1])
			} else {
				addComment(offset, i-offset, src[offset+2:i])
			}

		case '/':
			if i+1 >= len(src) || src[i+1] != '*' {
				continue
			}
			offset := i
			for {
				if !moveTo('*') || i >= len(src)-1 {
					return nil, errors.New("Missing '*/'")
				}
				if src[i+1] == '/' {
					i += 1
					break
				}
			}
			addComment(offset, i+1-offset, src[offset+2:i-1])
		}

	}
	return comments, nil
}
