package context

import (
	"fmt"
	"github.com/huangjunwen/JustSQL/utils"
	"github.com/pingcap/tidb/ast"
	"github.com/pingcap/tidb/mysql"
	"github.com/pingcap/tidb/util/types"
	"strings"
)

// ResultFieldMeta contains meta information of a result field.
type ResultFieldMeta struct {
	*ast.ResultField

	// Field name, maybe contain non-identifier chars: "now()"
	Name string

	// Field type.
	Type types.FieldType

	// Go type to store this field.
	AdaptType *TypeName

	// If this is a field from table column.
	// The following will have values.
	Table  *TableMeta
	Column *ColumnMeta
}

func NewResultFieldMeta(ctx *Context, rf *ast.ResultField) (*ResultFieldMeta, error) {

	db_meta := ctx.DefaultDBMeta

	// Is it from a real table?
	var table *TableMeta = nil
	var column *ColumnMeta = nil
	table, _ = db_meta.Tables[rf.Table.Name.L]
	if table != nil {
		column = table.Columns[rf.Column.Offset]
	}

	// Determin the name.
	name, err := resultFieldName(rf, false)
	if err != nil {
		return nil, err
	}

	return &ResultFieldMeta{
		ResultField: rf,
		Name:        name,
		Type:        rf.Column.FieldType,
		AdaptType:   ctx.TypeAdapter.AdaptType(&rf.Column.FieldType),
		Table:       table,
		Column:      column,
	}, nil

}

func (rf *ResultFieldMeta) IsEnum() bool {
	return rf.Type.Tp == mysql.TypeEnum
}

func (rf *ResultFieldMeta) IsSet() bool {
	return rf.Type.Tp == mysql.TypeSet
}

// SelectTableSourcesMeta contains meta information of the 'FROM' part
// of SELECT statment.
type SelectTableSourcesMeta struct {
	// The same as in github.com/pingcap/tidb/plan/resolve.go:resolverContext
	// NOTE: normal table names (alias) can not have same names, derived table
	// alias names can not have same names, but normal table and derived table
	// can have same names.
	Tables          []*ast.TableSource
	TableMap        map[string]int
	DerivedTableMap map[string]int
}

func NewSelectTableSourcesMeta(ctx *Context, stmt *ast.SelectStmt) (*SelectTableSourcesMeta, error) {

	if err := ensureSelectStmtCompiled(ctx, stmt); err != nil {
		return nil, err
	}

	ret := &SelectTableSourcesMeta{
		Tables:          make([]*ast.TableSource, 0),
		TableMap:        make(map[string]int),
		DerivedTableMap: make(map[string]int),
	}

	var collect func(*ast.Join) error

	collect = func(j *ast.Join) error {
		// Left then right
		rss := []ast.ResultSetNode{
			j.Left,
			j.Right,
		}
		for _, rs := range rss {
			if rs == nil {
				continue
			}
			switch r := rs.(type) {
			case *ast.TableSource:

				var (
					name       string
					is_derived bool = false
				)
				// see github.com/pingcap/tidb/plan/resolve.go:handleTableSource
				switch s := r.Source.(type) {
				case *ast.TableName:
					name = r.AsName.L
					if name == "" {
						name = ctx.UniqueTableName(s.Schema.L, s.Name.L)
					}

				default:
					name = r.AsName.L
					is_derived = true
				}

				if name == "" {
					return fmt.Errorf("[bug?] No name for table source[%d]", len(ret.Tables))
				}
				// see https://github.com/pingcap/tidb/issues/3908
				if !is_derived {
					if _, ok := ret.TableMap[name]; ok {
						return fmt.Errorf("[bug?] Duplicate normal table name %+q", name)
					}
					ret.TableMap[name] = len(ret.Tables)
				} else {
					if _, ok := ret.DerivedTableMap[name]; ok {
						return fmt.Errorf("[bug?] Duplicate derived table name %+q", name)
					}
					ret.DerivedTableMap[name] = len(ret.Tables)
				}
				ret.Tables = append(ret.Tables, r)

			case *ast.Join:
				if err := collect(r); err != nil {
					return err
				}

			default:
				return fmt.Errorf("Not supported type %T in collect", r)
			}
		}

		return nil
	}

	if err := collect(stmt.From.TableRefs); err != nil {
		return nil, err
	}

	return ret, nil

}

// Return a list of result field of the table or nil if not exists.
// NOTE: if table_name is both normal table name and derived table name,
// the result will be the combination of both.
// see github.com/pingcap/tidb/plan/resolve.go:createResultFields
func (s *SelectTableSourcesMeta) TableResultFields(table_name string) []*ast.ResultField {

	tab_idx1, ok1 := s.TableMap[table_name]
	tab_idx2, ok2 := s.DerivedTableMap[table_name]
	if !ok1 && !ok2 {
		return nil
	}
	ret := []*ast.ResultField{}
	if ok1 {
		ret = s.Tables[tab_idx1].Source.GetResultFields()
	}
	if ok2 {
		ret = append(ret, s.Tables[tab_idx2].Source.GetResultFields()...)
	}
	return ret

}

func (s *SelectTableSourcesMeta) Has(table_name string) bool {
	_, ok := s.TableMap[table_name]
	return ok
}

func (s *SelectTableSourcesMeta) HasDerived(table_name string) bool {
	_, ok := s.DerivedTableMap[table_name]
	return ok
}

// SelectStmtMeta contains meta information of a SELECT statement.
type SelectStmtMeta struct {
	*ast.SelectStmt

	ResultFields []*ResultFieldMeta

	Sources *SelectTableSourcesMeta
}

func NewSelectStmtMeta(ctx *Context, stmt *ast.SelectStmt) (*SelectStmtMeta, error) {

	if err := ensureSelectStmtCompiled(ctx, stmt); err != nil {
		return nil, err
	}

	rfs := stmt.GetResultFields()
	ret := &SelectStmtMeta{
		SelectStmt:   stmt,
		ResultFields: make([]*ResultFieldMeta, 0, len(rfs)),
	}

	// Extract result fields meta.
	for _, rf := range rfs {
		rfm, err := NewResultFieldMeta(ctx, rf)
		if err != nil {
			return nil, err
		}
		ret.ResultFields = append(ret.ResultFields, rfm)
	}

	// Extract table source meta.
	if sources, err := NewSelectTableSourcesMeta(ctx, stmt); err != nil {
		return nil, err
	} else {
		ret.Sources = sources
	}

	return ret, nil

}

type WildcardExpansion struct {
	// The table source's name.
	TableSourceName string

	// Portion of the expansion in result fields.
	ResultFieldOffset int
	ResultFieldNum    int
}

// WildcardExpansions contain information of a wildcard expansion.
type WildcardExpansions struct {
	Expansions []*WildcardExpansion

	// Map index of result field -> index of Expansions, -1 means not in wildcard expansion.
	resultFieldIndex []int
}

func (e *WildcardExpansions) Add(table_source_name string, result_field_offset int, result_field_num int) {

	if result_field_offset < 0 {
		panic(fmt.Errorf("result_field_offset < 0"))
	}
	if result_field_num <= 0 {
		panic(fmt.Errorf("result_field_num <= 0"))
	}
	if e.Expansions == nil {
		e.Expansions = make([]*WildcardExpansion, 0)
	}
	if e.resultFieldIndex == nil {
		e.resultFieldIndex = make([]int, 0)
	}

	// Resize.
	old_len := len(e.resultFieldIndex)
	new_len := result_field_offset + result_field_num
	for i := 0; i < new_len-old_len; i += 1 {
		e.resultFieldIndex = append(e.resultFieldIndex, -1)
	}

	// Mark.
	idx := len(e.Expansions)
	for i := result_field_offset; i < result_field_offset+result_field_num; i += 1 {
		if e.resultFieldIndex[i] != -1 {
			panic(fmt.Errorf("e.resultFieldIndex[%d] == %d != -1", i, e.resultFieldIndex[i]))
		}
		e.resultFieldIndex[i] = idx
	}
	e.Expansions = append(e.Expansions, &WildcardExpansion{
		TableSourceName:   table_source_name,
		ResultFieldOffset: result_field_offset,
		ResultFieldNum:    result_field_num,
	})

}

// Return the wildcard table source name of the i-th result field or "" if the result
// field is not from a wildcard expansion.
func (e *WildcardExpansions) TableSourceName(i int) string {

	if i < 0 || i >= len(e.resultFieldIndex) {
		return ""
	}
	idx := e.resultFieldIndex[i]
	if idx == -1 {
		return ""
	}
	return e.Expansions[idx].TableSourceName

}

// Return the offset inside one wildcard of the i-th result field or -1 if the result
// field is not from a wildcard expansion.
func (e *WildcardExpansions) Offset(i int) int {

	if i < 0 || i >= len(e.resultFieldIndex) {
		return -1
	}
	idx := e.resultFieldIndex[i]
	if idx == -1 {
		return -1
	}
	return i - e.Expansions[idx].ResultFieldOffset

}

// This function expand all wildcards ("*") in a SELECT statement and return
// an new equivalent one. This is useful since "SELECT * ..." may lead to
// unpredictable error when table is altered.
func ExpandWildcard(ctx *Context, stmt *ast.SelectStmt) (*ast.SelectStmt, *WildcardExpansions, error) {

	// Check wildcard.
	has_wildcard := false
	for _, f := range stmt.Fields.Fields {
		if f.WildCard != nil {
			has_wildcard = true
			break
		}
	}
	if !has_wildcard {
		return stmt, nil, nil
	}

	db := ctx.DB
	origin := stmt.Text()

	// Re-parse and re-compile. Since we need Offset of wildcards which
	// needs to be re-calculated.
	if stmts, err := db.Parse(origin); err != nil {
		return nil, nil, err
	} else {
		stmt = stmts[0].(*ast.SelectStmt)
		if _, err := db.Compile(stmt); err != nil {
			return nil, nil, err
		}
	}

	// Extract table source meta.
	sources, err := NewSelectTableSourcesMeta(ctx, stmt)
	if err != nil {
		return nil, nil, err
	}

	expansions := new(WildcardExpansions)

	// Iter fields and replace all wildcards.
	offset := 0
	parts := make([]string, 0)
	result_field_offset := 0
	for i, f := range stmt.Fields.Fields {
		// Not a wildcard field.
		if f.WildCard == nil {
			result_field_offset += 1
			continue
		}

		err_prefix := fmt.Sprintf("Expan wildcard field[%d]:", i)
		table_name := ctx.UniqueTableName(f.WildCard.Schema.L, f.WildCard.Table.L)

		// Save part before this wildcard field.
		parts = append(parts, origin[offset:f.Offset])

		// Calculate this wildcard field length. XXX: f.Text() return "" so i need to
		// construct the field text myself -_-
		field_text := "*"
		if table_name != "" {
			field_text = table_name + ".*"
		}

		if !strings.HasPrefix(origin[f.Offset:], field_text) {
			return nil, nil, fmt.Errorf("%s strings.HasPrefix(%+q, %+q) == false", err_prefix,
				origin[f.Offset:], field_text)
		}

		// Move offset.
		offset = f.Offset + len(field_text)

		// Expand wildcard.
		expan_parts := []string{}

		// Qualified wildcard ("[db.]tbl.*")
		if table_name != "" {

			rfs := sources.TableResultFields(table_name)
			if rfs == nil {
				panic(fmt.Errorf("%s No result fields for table %+q", err_prefix, table_name))
			}

			for j, rf := range rfs {
				rf_name, err := resultFieldName(rf, true)
				if err != nil {
					return nil, nil, fmt.Errorf("%s resultFieldName for %+q[%d]: %s",
						err_prefix, table_name, j, err)
				}
				expan_parts = append(expan_parts, table_name+"."+rf_name)
			}

			expansions.Add(table_name, result_field_offset, len(rfs))
			result_field_offset += len(rfs)

		} else {

			// Unqualified wildcard ("*")
			r_table_map := make(map[int]string)
			for k, v := range sources.TableMap {
				r_table_map[v] = k
			}
			for k, v := range sources.DerivedTableMap {
				r_table_map[v] = k
			}

			for j, table := range sources.Tables {
				table_name = r_table_map[j]
				rfs := table.GetResultFields()
				for k, rf := range rfs {
					rf_name, err := resultFieldName(rf, true)
					if err != nil {
						return nil, nil, fmt.Errorf("%s resultFieldName for %+q[%d]: %s",
							err_prefix, table_name, k, err)
					}
					expan_parts = append(expan_parts, table_name+"."+rf_name)
				}
				expansions.Add(table_name, result_field_offset, len(rfs))
				result_field_offset += len(rfs)
			}
		}

		parts = append(parts, strings.Join(expan_parts, ", "))

	}

	parts = append(parts, origin[offset:])
	text := strings.Join(parts, "")

	// Second-re-parse and re-compile.
	if stmts, err := db.Parse(text); err != nil {
		return nil, nil, err
	} else {
		stmt = stmts[0].(*ast.SelectStmt)
		if _, err := db.Compile(stmt); err != nil {
			return nil, nil, err
		}
	}

	return stmt, expansions, nil

}

func resultFieldName(rf *ast.ResultField, as_identifier bool) (string, error) {

	rf_name := rf.ColumnAsName.L
	if rf_name == "" {
		rf_name = rf.Column.Name.L
	}
	if rf_name == "" {
		return "", fmt.Errorf("Empty result field name")
	}
	if !as_identifier {
		return rf_name, nil
	}
	if utils.IsIdent(rf_name) {
		return rf_name, nil
	}
	return "", fmt.Errorf("%+q is not a valid identifier", rf_name)

}

func ensureSelectStmtCompiled(ctx *Context, stmt *ast.SelectStmt) error {

	db := ctx.DB
	if stmt.GetResultFields() == nil {
		if _, err := db.Compile(stmt); err != nil {
			return err
		}
	}

	return nil

}
