package annot

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

// Annot (annotation) is some extra information(k/v) stored in comments.
// Format (except SubsAnnot):
//   primaryKey[:primaryVal] key[:val] ...
// Example:
//   func:HelloWorld attribute:"one two three"
type Annot interface {
	SetPrimary(val string) error
	// Set non-primary key/val, if key == "", there will be no more
	// key/val pairs.
	Set(key string, val string) error
}

// Annotation primary key -> Annot type
var annotMap = make(map[string]reflect.Type)

// RegistAnnot register annotation type. obj should be a pointer.
func RegistAnnot(obj Annot, names ...string) {

	typ := reflect.TypeOf(obj)
	if typ.Kind() != reflect.Ptr {
		panic(fmt.Errorf("Expect ptr value, but got %s", typ.Kind()))
	}

	typ = typ.Elem()
	for _, name := range names {
		annotMap[name] = typ
	}

}

// ParseAnnot parse annotation from a string.
func ParseAnnot(src string) (Annot, error) {

	src = strings.TrimSpace(src)

	// Special case for SubsAnnot
	if len(src) <= 0 {
		return &SubsAnnot{
			Content: "",
		}, nil
	} else if src[0] == '$' {
		return &SubsAnnot{
			Content: strings.TrimSpace(src[1:]),
		}, nil
	}

	// Normal case
	fn := parseAnnotString(src)

	// The first key/val is primary key/val
	primaryKey, primaryVal, err := fn()
	if err != nil {
		return nil, err
	}

	// Using primary key to choose type
	typ, ok := annotMap[primaryKey]
	if !ok {
		return nil, fmt.Errorf("Unknown annotation %+q", primaryKey)
	}
	ret := reflect.New(typ).Interface().(Annot)

	// Set primary val
	err = ret.SetPrimary(primaryVal)
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
	annotRe  *regexp.Regexp = regexp.MustCompile(`^([A-Za-z][0-9A-Za-z_]*)(:(("[^"\\]*(?:\\.[^"\\]*)*")|([^\s:"]+)))?\s+`)
	escapeRe *regexp.Regexp = regexp.MustCompile(`\\.`)
)

func parseAnnotString(src string) func() (string, string, error) {

	// Append a space at the end to match annotRe
	remain := append([]byte(strings.TrimSpace(src)), ' ')

	return func() (string, string, error) {
		// drained
		if len(remain) == 0 {
			return "", "", nil
		}

		m := annotRe.FindSubmatch(remain)
		if m == nil {
			return "", "", fmt.Errorf("Illegal annot kv format near: %q", string(remain))
		}

		l := len(m[0])
		k := m[1]
		v := m[3]

		// if it is quoted. Unquote it, see: https://golang.org/ref/spec#Rune_literals
		if len(m[4]) != 0 {
			v = escapeRe.ReplaceAllFunc(v[1:len(v)-1], func(x []byte) []byte {
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
