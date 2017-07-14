package render

import (
	"github.com/huangjunwen/JustSQL/context"
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

// Return 'col1, col2, col3, ...'
func columnNameList(cols []*context.ColumnData) string {
	parts := make([]string, 0, len(cols))
	for _, col := range cols {
		parts = append(parts, col.Name.O)
	}
	return strings.Join(parts, ", ")
}

func placeholder() string {
	return "?"
}

// Return '?, ?, ?, ...'
func placeholderList(n int) string {
	if n <= 0 {
		return ""
	} else if n == 1 {
		return placeholder()
	}
	s := strings.Repeat(placeholder()+", ", n)
	return s[:len(s)-2]
}

func buildExtraFuncs(ctx *context.Context) template.FuncMap {

	tctx := ctx.TypeContext

	// Import pkg (its path) and return a unique name.
	imp := func(pkg_path string) (string, error) {
		return tctx.CurrScope().UsePkg(pkg_path), nil
	}

	return template.FuncMap{
		"imp":              imp,
		"not_nil":          notNil,
		"column_name_list": columnNameList,
		"placeholder":      placeholder,
		"placeholder_list": placeholderList,
	}

}
