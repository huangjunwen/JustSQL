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

	// Is it from a real table?
	var table *TableMeta = nil
	var column *ColumnMeta = nil
	if rf.Table.Name.L != "" {
		db_meta, err := ctx.GetDBMeta(rf.DBName.L)
		if err != nil {
			return nil, err
		}
		table = db_meta.Tables[rf.Table.Name.L]
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

// TableRefsMeta contains meta information table refs (table sources).
type TableRefsMeta struct {
	// Similar to github.com/pingcap/tidb/plan/resolve.go:resolverContext

	// List of table sources and its reference name. Reference name can be:
	//   - tbl_name
	//   - db_name.tbl_name
	//   - alias
	Tables        []*ast.TableSource
	TableRefNames []string

	// Map table ref name to its index.
	// NOTE: normal table names (alias) can not have same names; derived table
	// alias names can not have same names; but normal table and derived table
	// can have same names.
	TableMap        map[string]int
	DerivedTableMap map[string]int
}

func NewTableRefsMeta(ctx *Context, refs *ast.TableRefsClause) (*TableRefsMeta, error) {

	ret := &TableRefsMeta{
		Tables:          make([]*ast.TableSource, 0),
		TableRefNames:   make([]string, 0),
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
					table_ref_name string
					is_derived     bool = false
				)
				// see github.com/pingcap/tidb/plan/resolve.go:handleTableSource
				switch s := r.Source.(type) {
				case *ast.TableName:
					table_ref_name = r.AsName.L
					if table_ref_name == "" {
						table_ref_name = ctx.UniqueTableName(s.Schema.L, s.Name.L)
					}

				default:
					table_ref_name = r.AsName.L
					is_derived = true
				}

				if table_ref_name == "" {
					return fmt.Errorf("[bug?] No name for table source[%d]", len(ret.Tables))
				}
				// see https://github.com/pingcap/tidb/issues/3908
				if !is_derived {
					if _, ok := ret.TableMap[table_ref_name]; ok {
						return fmt.Errorf("[bug?] Duplicate normal table ref name %+q", table_ref_name)
					}
					ret.TableMap[table_ref_name] = len(ret.Tables)
				} else {
					if _, ok := ret.DerivedTableMap[table_ref_name]; ok {
						return fmt.Errorf("[bug?] Duplicate derived table ref name %+q", table_ref_name)
					}
					ret.DerivedTableMap[table_ref_name] = len(ret.Tables)
				}
				ret.Tables = append(ret.Tables, r)
				ret.TableRefNames = append(ret.TableRefNames, table_ref_name)

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

	if err := collect(refs.TableRefs); err != nil {
		return nil, err
	}

	return ret, nil

}

// Return a list of result field of the table or nil if not exists.
// NOTE: if table_ref_name is both normal table name and derived table name,
// the result will be the combination of both.
// see github.com/pingcap/tidb/plan/resolve.go:createResultFields
func (s *TableRefsMeta) GetResultFields(table_ref_name string) []*ast.ResultField {

	tab_idx1, ok1 := s.TableMap[table_ref_name]
	tab_idx2, ok2 := s.DerivedTableMap[table_ref_name]
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

func (s *TableRefsMeta) Has(table_ref_name string) bool {
	_, ok := s.TableMap[table_ref_name]
	return ok
}

func (s *TableRefsMeta) HasDerived(table_ref_name string) bool {
	_, ok := s.DerivedTableMap[table_ref_name]
	return ok
}

// WildcardMeta contain information for a wildcard.
// NOTE: For unqualified wildcard "*", serveral WildcardMeta will be
// generated.
type WildcardMeta struct {
	// Table ref name of wildcard.
	TableRefName string

	// Portion of the wildcard in result fields.
	ResultFieldOffset int
	ResultFieldNum    int
}

// FieldListMeta contain field list meta information in SELECT statement.
type FieldListMeta struct {
	// Wildcard information.
	Wildcards []WildcardMeta

	// Map index of result field -> index in wildcards or -1 if the result field
	// is not in a wildcard.
	ResultFieldToWildcard []int
}

func (f *FieldListMeta) addWildcard(table_ref_name string, result_field_offset int, result_field_num int) {

	idx := len(f.Wildcards)
	for i := result_field_offset; i < result_field_offset+result_field_num; i++ {
		if f.ResultFieldToWildcard[i] != -1 {
			panic(fmt.Errorf("f.ResultFieldToWildcard[%d] == %d != -1", i, f.ResultFieldToWildcard[i]))
		}
		f.ResultFieldToWildcard[i] = idx
	}
	f.Wildcards = append(f.Wildcards, WildcardMeta{
		TableRefName:      table_ref_name,
		ResultFieldOffset: result_field_offset,
		ResultFieldNum:    result_field_num,
	})

}

// NewFieldListMeta create FieldListMeta from the field list and table refs of a SELECT statement.
func NewFieldListMeta(ctx *Context, stmt *ast.SelectStmt, table_refs_meta *TableRefsMeta) (*FieldListMeta, error) {

	ret := &FieldListMeta{
		Wildcards:             make([]WildcardMeta, 0),
		ResultFieldToWildcard: make([]int, len(stmt.GetResultFields())),
	}
	for i := 0; i < len(stmt.GetResultFields()); i++ {
		ret.ResultFieldToWildcard[i] = -1
	}

	offset := 0
	for _, f := range stmt.Fields.Fields {
		if f.WildCard == nil {
			offset += 1
			continue
		}
		// Qualified wildcard ("[db.]tbl.*")
		table_ref_name := ctx.UniqueTableName(f.WildCard.Schema.L, f.WildCard.Table.L)
		if table_ref_name != "" {
			rfs := table_refs_meta.GetResultFields(table_ref_name)
			if len(rfs) == 0 {
				return nil, fmt.Errorf("[bug?] TableRefsMeta.GetResultFields(%+q) == nil", table_ref_name)
			}
			ret.addWildcard(table_ref_name, offset, len(rfs))
			offset += len(rfs)
			continue
		}
		// Unqualified wildcard ("*")
		for i, table := range table_refs_meta.Tables {
			table_ref_name := table_refs_meta.TableRefNames[i]
			rfs := table.GetResultFields()
			if len(rfs) == 0 {
				return nil, fmt.Errorf("[bug?] Table[%+q].GetResultFields() == nil", table_ref_name)
			}
			ret.addWildcard(table_ref_name, offset, len(rfs))
			offset += len(rfs)
		}

	}

	if offset != len(stmt.GetResultFields()) {
		return nil, fmt.Errorf("[bug?] Field list expanded length(%d) != Result field length(%d)",
			offset, len(stmt.GetResultFields()))
	}

	return ret, nil
}

// WildcardTableRefName return the wildcard table ref name of the i-th result field
// or "" if the result field is not inside wildcard.
func (f *FieldListMeta) WildcardTableRefName(i int) string {

	if i < 0 || i >= len(f.ResultFieldToWildcard) {
		return ""
	}
	idx := f.ResultFieldToWildcard[i]
	if idx < 0 {
		return ""
	}
	return f.Wildcards[idx].TableRefName

}

// WildcardOffset return the offset inside wildcard of the i-th result field
// or -1 if the result field is not inside wildcard.
func (f *FieldListMeta) WildcardOffset(i int) int {

	if i < 0 || i >= len(f.ResultFieldToWildcard) {
		return -1
	}
	idx := f.ResultFieldToWildcard[i]
	if idx < 0 {
		return -1
	}
	return i - f.Wildcards[idx].ResultFieldOffset

}

// SelectStmtMeta contains meta information of a SELECT statement.
type SelectStmtMeta struct {
	*ast.SelectStmt

	ResultFields []*ResultFieldMeta
	TableRefs    *TableRefsMeta
	FieldList    *FieldListMeta
}

// NewSelectStmtMeta create SelectStmtMeta from *ast.SelectStmt.
func NewSelectStmtMeta(ctx *Context, stmt *ast.SelectStmt) (*SelectStmtMeta, error) {

	if err := ensureSelectStmtCompiled(ctx, stmt); err != nil {
		return nil, err
	}

	ret := &SelectStmtMeta{
		SelectStmt: stmt,
	}

	// Extract result fields meta.
	rfs := stmt.GetResultFields()
	ret.ResultFields = make([]*ResultFieldMeta, 0, len(rfs))
	for _, rf := range rfs {
		rfm, err := NewResultFieldMeta(ctx, rf)
		if err != nil {
			return nil, err
		}
		ret.ResultFields = append(ret.ResultFields, rfm)
	}

	// Extract table refs meta.
	if refsm, err := NewTableRefsMeta(ctx, stmt.From); err != nil {
		return nil, err
	} else {
		ret.TableRefs = refsm
	}

	// Extract wildcards meta.
	if flm, err := NewFieldListMeta(ctx, stmt, ret.TableRefs); err != nil {
		return nil, err
	} else {
		ret.FieldList = flm
	}

	return ret, nil

}

// This function expand all wildcards ("*") in a SELECT statement and return
// an new equivalent one. This is useful since "SELECT * ..." may lead to
// unpredictable error when table is altered.
func (s *SelectStmtMeta) ExpandWildcard(ctx *Context) (*SelectStmtMeta, error) {

	if len(s.FieldList.Wildcards) == 0 {
		return s, nil
	}

	db := ctx.DB
	text := s.SelectStmt.Text()

	// Re-parse and re-compile. Since we need Offset of wildcards which
	// needs to be re-calculated.
	var stmt *ast.SelectStmt
	if stmts, err := db.Parse(text); err != nil {
		return nil, err
	} else {
		stmt = stmts[0].(*ast.SelectStmt)
		if _, err := db.Compile(stmt); err != nil {
			return nil, err
		}
	}

	parts := []string{}
	text_offset := 0
	for n, field := range stmt.Fields.Fields {

		if field.WildCard == nil {
			continue
		}

		err_prefix := fmt.Sprintf("Expand wildcard field[%d]:", n)

		// Save part before this wildcard field.
		parts = append(parts, text[text_offset:field.Offset])

		// Calculate this wildcard field length to move text_offset.
		// XXX: field.Text() return "" so i need to construct the field text myself -_-
		field_text := "*"
		if field.WildCard.Table.O != "" {
			field_text = field.WildCard.Table.O + ".*"
			if field.WildCard.Schema.O != "" {
				field_text = field.WildCard.Schema.O + "." + field_text
			}
		}
		if !strings.HasPrefix(text[field.Offset:], field_text) {
			return nil, fmt.Errorf("%s strings.HasPrefix(%+q, %+q) == false", err_prefix,
				text[field.Offset:], field_text)
		}
		text_offset = field.Offset + len(field_text)

		// Expand wildcard.
		table_ref_name := ctx.UniqueTableName(field.WildCard.Schema.L, field.WildCard.Table.L)
		expand_parts := []string{}

		if table_ref_name != "" {

			// Qualified wildcard ("[db.]tbl.*")
			rfs := s.TableRefs.GetResultFields(table_ref_name)
			if rfs == nil {
				panic(fmt.Errorf("%s No result fields for table %+q", err_prefix, table_ref_name))
			}

			for i, rf := range rfs {
				rf_name, err := resultFieldName(rf, true)
				if err != nil {
					return nil, fmt.Errorf("%s resultFieldName for %+q[%d]: %s",
						err_prefix, table_ref_name, i, err)
				}
				expand_parts = append(expand_parts, table_ref_name+"."+rf_name)
			}

		} else {

			// Unqualified wildcard ("*")
			for i, table := range s.TableRefs.Tables {

				table_ref_name = s.TableRefs.TableRefNames[i]
				rfs := table.GetResultFields()

				for j, rf := range rfs {
					rf_name, err := resultFieldName(rf, true)
					if err != nil {
						return nil, fmt.Errorf("%s resultFieldName for %+q[%d]: %s",
							err_prefix, table_ref_name, j, err)
					}
					expand_parts = append(expand_parts, table_ref_name+"."+rf_name)
				}
			}
		}

		parts = append(parts, strings.Join(expand_parts, ", "))

	}

	parts = append(parts, text[text_offset:])
	text = strings.Join(parts, "")

	// Second-re-parse and re-compile.
	if stmts, err := db.Parse(text); err != nil {
		return nil, err
	} else {
		stmt = stmts[0].(*ast.SelectStmt)
	}

	return NewSelectStmtMeta(ctx, stmt)

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
