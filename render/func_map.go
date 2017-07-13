package render

import (
	"github.com/huangjunwen/JustSQL/context"
	"github.com/pingcap/tidb/mysql"
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

// Return columns that need explicit values to insert.
func columnsForInsert(table *context.TableData) []*context.ColumnData {
	ret := make([]*context.ColumnData, 0)
	for _, col := range table.Columns {
		// Skip auto increment column
		if col.IsAutoIncrement {
			continue
		}
		// Skip time column with default now()
		switch tp := col.ColumnInfo.FieldType.Tp; tp {
		case mysql.TypeDate, mysql.TypeDatetime, mysql.TypeTimestamp:
			if col.DefaultValue == interface{}("CURRENT_TIMESTAMP") {
				continue
			}
		}
		ret = append(ret, col)
	}
	return ret
}

// Return columns that need explicit values to update.
func columnsForUpdate(table *context.TableData) []*context.ColumnData {
	ret := make([]*context.ColumnData, 0)
	for _, col := range table.Columns {
		// Skip ON UPDATE CURRENT_TIMESTAMP
		if col.IsOnUpdateNow {
			continue
		}
		ret = append(ret, col)
	}
	return ret
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
		"imp":                imp,
		"not_nil":            notNil,
		"columns_for_insert": columnsForInsert,
		"columns_for_update": columnsForUpdate,
		"column_name_list":   columnNameList,
		"placeholder":        placeholder,
		"placeholder_list":   placeholderList,
	}

}
