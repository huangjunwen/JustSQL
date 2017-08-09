package render

import (
	"fmt"
	"github.com/huangjunwen/JustSQL/context"
	"github.com/huangjunwen/JustSQL/utils"
	ts "github.com/pingcap/tidb/util/types"
	"reflect"
	"strings"
	"text/template"
)

// --- Interfaces ---

// Indexer represents a random accessable array.
type Indexer interface {
	Len() int
	// Panic if out of range
	Index(i int) interface{}
}

// Appendable represents a mutable list of items.
type Appendable interface {
	Append(items ...interface{}) error
}

// StringsContainer represents a list of strings.
type StringsContainer interface {
	Strings() []string
}

// --- General helpers ---

// Return false only when v is nil.
func notNil(v interface{}) (res bool) {
	defer func() {
		var err error
		if utils.RecoverErr(&err) {
			res = false
		}
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

func first(val interface{}) (ret interface{}, err error) {
	defer func() {
		if utils.RecoverErr(&err) {
			ret = ""
		}
	}()
	switch v := val.(type) {
	case Indexer:
		ret = v.Index(0)
	default:
		ret = reflect.ValueOf(val).Index(0).Interface()
	}
	return
}

func last(val interface{}) (ret interface{}, err error) {
	defer func() {
		if utils.RecoverErr(&err) {
			ret = ""
		}
	}()
	switch v := val.(type) {
	case Indexer:
		ret = v.Index(v.Len() - 1)
	default:
		reflectVal := reflect.ValueOf(val)
		ret = reflectVal.Index(reflectVal.Len() - 1).Interface()
	}
	return
}

func append_(val interface{}, items ...interface{}) (ret string, err error) {
	defer func() {
		utils.RecoverErr(&err)
	}()
	switch v := val.(type) {
	case Appendable:
		err = v.Append(items...)
	default:
		itemValues := make([]reflect.Value, 0, len(items))
		for _, item := range items {
			itemValues = append(itemValues, reflect.ValueOf(item))
		}
		reflect.Append(reflect.ValueOf(val), itemValues...)
	}
	return
}

// --- String helpers ---

// StringList is just a list of strings.
type StringList struct {
	ss []string
}

func NewStringList() *StringList {
	return &StringList{
		ss: []string{},
	}
}

func (sl *StringList) Append(items ...interface{}) error {
	for _, item := range items {
		switch it := item.(type) {
		case string:
			sl.ss = append(sl.ss, it)
		case fmt.Stringer:
			sl.ss = append(sl.ss, it.String())
		default:
			return fmt.Errorf("stringList.Append: expect string but got %T", item)
		}
	}
	return nil
}

func (sl *StringList) Strings() []string {
	return sl.ss
}

func (sl *StringList) Len() int {
	return len(sl.ss)
}

func (sl *StringList) Index(i int) interface{} {
	return sl.ss[i]
}

// UniqueStringList is a list of unique strings.
type UniqueStringList struct {
	ss        []string
	smap      map[string]int
	converter func(string) string
	default_  string
}

func NewUniqueStringList(converter func(string) string, default_ string) *UniqueStringList {
	if converter == nil {
		converter = func(s string) string {
			return s
		}
	}
	return &UniqueStringList{
		ss:        []string{},
		smap:      map[string]int{},
		converter: converter,
		default_:  default_,
	}
}

func (usl *UniqueStringList) Append(items ...interface{}) error {
	for _, item := range items {

		var s string
		switch it := item.(type) {
		case string:
			s = it
		case fmt.Stringer:
			s = it.String()
		default:
			return fmt.Errorf("uniqueStringList.Append: expect string but got %T", item)
		}

		s = usl.converter(s)
		if s == "" {
			s = usl.default_
		}
		r := s
		for i := 1; ; i++ {
			if _, ok := usl.smap[r]; !ok {
				usl.smap[r] = len(usl.ss)
				usl.ss = append(usl.ss, r)
				break
			}
			r = fmt.Sprintf("%s%d", s, i)
		}
	}

	return nil
}

func (usl *UniqueStringList) Strings() []string {
	return usl.ss
}

func (usl *UniqueStringList) Len() int {
	return len(usl.ss)
}

func (usl *UniqueStringList) Index(i int) interface{} {
	return usl.ss[i]
}

func dup(s string, n int) []string {
	r := make([]string, n)
	for i := 0; i < len(r); i++ {
		r[i] = s
	}
	return r
}

func join(val interface{}, sep string) (string, error) {
	switch v := val.(type) {
	case []string:
		return strings.Join(v, sep), nil
	case StringsContainer:
		return strings.Join(v.Strings(), sep), nil
	default:
		return "", fmt.Errorf("join: expect []string or StringsContainer but got %T", v)
	}
}

// --- Source code helpers ---

// Import pkg into the renderred source code.
func buildImp(r *Renderer) func(string) *PkgName {
	return func(pkgPath string) *PkgName {
		return r.Scopes.CreatePkgName(pkgPath)
	}
}

func buildTypeName(r *Renderer) func(interface{}) (*TypeName, error) {
	return func(val interface{}) (*TypeName, error) {
		switch v := val.(type) {
		case *ts.FieldType:
			return r.TypeAdapter.AdaptType(v), nil
		case string:
			return r.Scopes.CreateTypeNameFromSpec(v), nil
		default:
			return nil, fmt.Errorf("typeName: not support %T as argument", v)
		}
	}
}

// --- Database helpers ---

type ColumnList struct {
	cols []*context.ColumnMeta
}

func NewColumnList() *ColumnList {
	return &ColumnList{
		cols: []*context.ColumnMeta{},
	}
}

func (cl *ColumnList) Len() int {
	return len(cl.cols)
}

func (cl *ColumnList) Index(i int) interface{} {
	return cl.cols[i]
}

func (cl *ColumnList) Append(items ...interface{}) error {
	for _, item := range items {
		if col, ok := item.(*context.ColumnMeta); !ok {
			return fmt.Errorf("columnList.Append: expect *context.ColumnMeta but got %T", item)
		} else {
			cl.cols = append(cl.cols, col)
		}
	}
	return nil
}

func (cl *ColumnList) Cols() []*context.ColumnMeta {
	return cl.cols
}

// Return names of columns.
func columnNames(val interface{}) []string {
	var cols []*context.ColumnMeta
	switch v := val.(type) {
	case []*context.ColumnMeta:
		cols = v
	case *ColumnList:
		cols = v.cols
	}
	r := make([]string, 0, len(cols))
	for _, col := range cols {
		r = append(r, col.Name)
	}
	return r
}

func buildDBName(r *Renderer) func() string {
	dbName := r.Context.DBName
	return func() string {
		return dbName
	}
}

func BuildExtraFuncs(r *Renderer) template.FuncMap {

	fnMap := template.FuncMap{
		// General helpers.
		"notNil": notNil,
		"first":  first,
		"last":   last,
		"append": append_,
		// String helpers.
		"pascal":           utils.PascalCase,
		"camel":            utils.CamelCase,
		"stringList":       NewStringList,
		"uniqueStringList": NewUniqueStringList,
		"split":            strings.Split,
		"dup":              dup,
		"join":             join,
		// Source code helpers.
		"imp":      buildImp(r),
		"typeName": buildTypeName(r),
		// Database helpers.
		"columnList":  NewColumnList,
		"columnNames": columnNames,
		// Context helpers.
		"dbname": buildDBName(r),
	}

	// Can be used for getting a function as variable.
	fnMap["fn"] = buildFn(fnMap)
	return fnMap

}
