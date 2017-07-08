package annot

import (
	"fmt"
	"github.com/huangjunwen/JustSQL/utils"
	"regexp"
	"strings"
)

// Annotation is some extra information(k/v) stored in comments. Format:
//   primaryKey[:primaryVal] key[:val] ...
// Example:
//   bind:name slice type:"sql.NullString"
type Annot interface {
	SetPrimary(val string) error
	// Set non-primary key/val, if key == "", there will be no more
	// key/val pairs.
	Set(key, val string) error
}

// Parse annotation.
func ParseAnnot(src string) (Annot, error) {
	fn := parseAnnotString(src)

	// The first key/val is primary key/val
	primary_key, primary_val, err := fn()
	if err != nil {
		return nil, err
	}

	// Using primary key to choose type
	var ret Annot

	switch primary_key {
	default:
		return nil, fmt.Errorf("Unknown annotation type: %q", primary_key)
	case "bind":
		ret = &BindAnnot{}
	}

	// Set primary val
	err = ret.SetPrimary(primary_val)
	if err != nil {
		return nil, err
	}

	// Set non-primary key/val
	for {
		key, val, err := fn()
		if err != nil {
			return nil, err
		}

		err = ret.Set(key, val)
		if err != nil {
			return nil, err
		}

		if key == "" {
			break
		}
	}

	return ret, nil

}

var (
	annot_re  *regexp.Regexp = regexp.MustCompile(`^([A-z][0-9A-z_]*)(:(("[^"\\]*(?:\\.[^"\\]*)*")|([^\s:"]+)))?\s+`)
	escape_re *regexp.Regexp = regexp.MustCompile(`\\.`)
)

func parseAnnotString(src string) func() (string, string, error) {
	remain := append([]byte(strings.TrimSpace(src)), ' ')
	return func() (string, string, error) {
		// drained
		if len(remain) == 0 {
			return "", "", nil
		}

		m := annot_re.FindSubmatch(remain)
		if m == nil {
			return "", "", fmt.Errorf("Illegal annot kv format near: %q", string(remain))
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

// Annotations

// Annotation for bind param, format:
//   bind:id type:int slice
// where 'type' and 'slice' are optional
type BindAnnot struct {
	// bind param name
	Name string

	// (optinal) param type
	Type string

	// param is slice? Using in 'WHERE id IN ?'
	Slice bool
}

func (a *BindAnnot) SetPrimary(val string) error {
	if val == "" {
		return fmt.Errorf("bind: missing bind name")
	}
	if !utils.IsIdent(val) {
		return fmt.Errorf("bind: bind name %q is not a valid identifier", val)
	}
	a.Name = val
	return nil
}

func (a *BindAnnot) Set(key, val string) error {
	switch key {
	default:
		return fmt.Errorf("bind: unknown key %q", key)
	case "type":
		a.Type = val
	case "slice":
		a.Slice = true
	case "":
	}
	return nil
}
