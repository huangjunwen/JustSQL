package render

import (
	"fmt"
	"github.com/huangjunwen/JustSQL/context"
	"github.com/huangjunwen/JustSQL/utils"
	"reflect"
	"strings"
	"text/template"
)

// --- General helpers ---

// Import pkg into the renderred source code.
func buildImp(ctx *context.Context) func(string) *context.PkgName {
	return func(pkgPath string) *context.PkgName {
		return ctx.Scopes.CreatePkgName(pkgPath)
	}
}

// Return false only when v is nil.
func notNil(v interface{}) (res bool) {
	defer func() {
		if r := recover(); r != nil {
			res = false
		}
		return
	}()
	res = !reflect.ValueOf(v).IsNil()
	return
}

// --- String helpers ---

// Create a function that return pascal case identifier of a string or
// empty if the string contains no identifier.
func pascalFunc(empty string) func(string) string {
	return func(s string) string {
		s = utils.PascalCase(s)
		if s == "" {
			return empty
		}
		return s
	}
}

// A list of strings.
type Strings []string

func (ss Strings) Add(s string) string {
	ss = append(ss, s)
	return ""
}

func NewStrings() Strings {
	return Strings{}
}

func splitLines(s string) Strings {
	return strings.Split(s, "\n")
}

// A set of unique strings.
type UniqueStrings struct {
	Strings
	StringsMap map[string]int
	Converter  func(string) string
}

func NewUniqueStrings(converter func(string) string) *UniqueStrings {
	if converter == nil {
		converter = func(s string) string {
			return s
		}
	}
	return &UniqueNames{
		Strings:    Strings{},
		StringsMap: map[string]int{},
		Converter:  converter,
	}
}

func (uss *UniqueStrings) Add(s string) string {
	s = uss.Converter(s)
	result := s
	for i := 1; ; i++ {
		if _, ok := uss.StringsMap[result]; !ok {
			uss.StringsMap[result] = len(uss.Strings)
			uss.Strings.Add(result)
			return ""
		}
		result = fmt.Sprintf("%s%d", s, i)
	}
}

// --- Database helpers ---

// Return 'col1, col2, col3, ...'
func columnNameList(cols []*context.ColumnMeta) string {
	parts := make([]string, 0, len(cols))
	for _, col := range cols {
		parts = append(parts, col.Name)
	}
	return strings.Join(parts, ", ")
}

// Return SQL parameter binding placeholder.
func buildPh(ctx *context.Context) func() string {
	return func() string {
		return ctx.Placeholder
	}
}

// Return a list of SQL parameters binding placeholders.
func buildPhList(ctx *context.Context) func(int) string {
	return func(n int) string {
		if n <= 0 {
			return ""
		} else if n == 1 {
			return ctx.Placeholder
		}
		s := strings.Repeat(ctx.Placeholder+", ", n)
		return s[:len(s)-2]
	}
}

func BuildExtraFuncs(ctx *context.Context) template.FuncMap {

	return template.FuncMap{
		// General helpers.
		"imp":    buildImp(ctx),
		"notNil": notNil,
		// Strings helpers.
		"pascalFunc":    pascalFunc,
		"strings":       NewStrings,
		"splitLines":    splitLines,
		"uniqueStrings": NewUniqueStrings,
		// Database helpers.
		"columnNameList": columnNameList,
		"ph":             buildPh(ctx),
		"phList":         buildPhList(ctx),
	}

}
