package render

import (
	"fmt"
	"github.com/huangjunwen/JustSQL/context"
	"github.com/huangjunwen/JustSQL/utils"
	"reflect"
	"strings"
	"text/template"
)

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

func splitLines(s string) []string {
	return strings.Split(s, "\n")
}

// Return 'col1, col2, col3, ...'
func columnNameList(cols []*context.ColumnMeta) string {
	parts := make([]string, 0, len(cols))
	for _, col := range cols {
		parts = append(parts, col.Name)
	}
	return strings.Join(parts, ", ")
}

// Convert a string to its pascal case. If the string is empty ("")
// it will return "Empty_"
func pascalNoEmpty(s string) string {
	ret := utils.PascalCase(s)
	if ret == "" {
		return "Empty_"
	}
	return ret
}

// A set of unqiue pascal names.
type UniqueNames struct {
	Names map[string]int
	Last  string
}

func NewUniqueNames() *UniqueNames {
	return &UniqueNames{
		Names: make(map[string]int),
	}
}

func (un *UniqueNames) Add(name string) string {
	name = utils.PascalCase(name)
	if name == "" {
		name = "NoName"
	}
	ret := name
	for i := 1; ; i += 1 {
		if _, ok := un.Names[ret]; !ok {
			un.Names[ret] = len(un.Names)
			un.Last = ret
			return ""
		}
		ret = fmt.Sprintf("%s%d", name, i)
	}
}

type StringArr []string

func NewStringArr() *StringArr {
	return &StringArr{}
}

func (a *StringArr) Push(s string) string {
	*a = append(*a, s)
	return ""
}

func BuildExtraFuncs(ctx *context.Context) template.FuncMap {

	scopes := ctx.Scopes

	imp := func(pkg_path string) *context.PkgName {
		return scopes.CreatePkgName(pkg_path)
	}

	placeholder := func() string {
		return ctx.Placeholder
	}

	// Return '?, ?, ?, ...'
	placeholderList := func(n int) string {
		if n <= 0 {
			return ""
		} else if n == 1 {
			return placeholder()
		}
		s := strings.Repeat(placeholder()+", ", n)
		return s[:len(s)-2]
	}

	return template.FuncMap{
		"imp":              imp,
		"not_nil":          notNil,
		"split_lines":      splitLines,
		"column_name_list": columnNameList,
		"placeholder":      placeholder,
		"placeholder_list": placeholderList,
		"pascal":           pascalNoEmpty,
		"unique_names":     NewUniqueNames,
		"string_arr":       NewStringArr,
	}

}
