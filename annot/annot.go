package annot

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

// Annotation is some extra information(k/v) stored in comments. Format:
//   primaryKey[:primaryVal] key[:val] ...
// Example:
//   bind:name multi
type Annot interface {
	SetPrimary(val string) error
	// Set non-primary key/val, if key == "", there will be no more
	// key/val pairs.
	Set(key, val string) error
}

// Annotation name -> Annot type
var annot_map = make(map[string]reflect.Type)

// Regist annotation. obj should be a pointer.
func RegistAnnot(obj Annot, names ...string) {

	typ := reflect.TypeOf(obj)
	if typ.Kind() != reflect.Ptr {
		panic(fmt.Errorf("Expect ptr value, but got %s", typ.Kind()))
	}

	typ = typ.Elem()
	for _, name := range names {
		annot_map[name] = typ
	}

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
	typ, ok := annot_map[primary_key]
	if !ok {
		return nil, fmt.Errorf("Unknown annotation %+q", primary_key)
	}
	ret := reflect.New(typ).Interface().(Annot)

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
	annot_re  *regexp.Regexp = regexp.MustCompile(`^([A-Za-z][0-9A-Za-z_]*)(:(("[^"\\]*(?:\\.[^"\\]*)*")|([^\s:"]+)))?\s+`)
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
