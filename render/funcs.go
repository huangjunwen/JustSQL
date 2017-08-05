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

// Return the function of the name.
func buildFn(fnMap template.FuncMap) func(string) interface{} {
	return func(fnName string) interface{} {
		if fn, ok := fnMap[fnName]; ok {
			return fn
		}
		return nil
	}
}

// --- String helpers ---

// A list of strings.
type Strings []string

func (ss Strings) Add(s string) string {
	ss = append(ss, s)
	return ""
}

func (ss Strings) Last() string {
	l := len(ss)
	if l == 0 {
		return ""
	}
	return ss[l-1]
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
	Default    string
}

func NewUniqueStrings(converter func(string) string, dft string) *UniqueStrings {
	if converter == nil {
		converter = func(s string) string {
			return s
		}
	}
	return &UniqueStrings{
		Strings:    Strings{},
		StringsMap: map[string]int{},
		Converter:  converter,
		Default:    dft,
	}
}

func (uss *UniqueStrings) Add(s string) string {
	s = uss.Converter(s)
	if s == "" {
		s = uss.Default
	}
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

func (uss *UniqueStrings) Last() string {
	return uss.Strings.Last()
}

func repeatJoin(n int, s string, sep string) string {
	parts := make([]string, n)
	for i := 0; i < len(parts); i++ {
		parts[i] = s
	}
	return strings.Join(parts, sep)
}

// --- Source code helpers ---

// Import pkg into the renderred source code.
func buildImp(ctx *context.Context) func(string) *context.PkgName {
	return func(pkgPath string) *context.PkgName {
		return ctx.Scopes.CreatePkgName(pkgPath)
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

func BuildExtraFuncs(ctx *context.Context) template.FuncMap {

	fnMap := template.FuncMap{
		// General helpers.
		"notNil": notNil,
		// String helpers.
		"pascal":        utils.PascalCase,
		"camel":         utils.CamelCase,
		"strings":       NewStrings,
		"splitLines":    splitLines,
		"uniqueStrings": NewUniqueStrings,
		"repeatJoin":    repeatJoin,
		// Source code helpers.
		"imp": buildImp(ctx),
		// Database helpers.
		"columnNameList": columnNameList,
	}

	// Can be used to getting a function as variable.
	fnMap["fn"] = buildFn(fnMap)
	return fnMap

}
