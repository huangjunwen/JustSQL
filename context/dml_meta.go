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

	// Field name
	Name       string
	PascalName string

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
	name, err := resultFieldNameAsIdentifier(rf, false)
	if err != nil {
		return nil, err
	}
	if rf.TableAsName.L != "" {
		name = fmt.Sprintf("%s_%s", rf.TableAsName.L, name)
	}

	return &ResultFieldMeta{
		ResultField: rf,
		Name:        name,
		PascalName:  utils.PascalCase(name),
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

				// see github.com/pingcap/tidb/plan/resolve.go:handleTableSource
				switch s := r.Source.(type) {
				case *ast.TableName:
					name := r.AsName.L
					if name == "" {
						name = ctx.UniqueTableName(s.Schema.L, s.Name.L)
					}
					if name == "" {
						// Should not be here since it has been Compiled.
						panic("Table name is empty")
					}
					ret.TableMap[name] = len(ret.Tables)
					ret.Tables = append(ret.Tables, r)

				default:
					name := r.AsName.L
					if name == "" {
						// Should not be here since it has been Compiled.
						panic("Derived table name is empty")
					}
					ret.DerivedTableMap[name] = len(ret.Tables)
					ret.Tables = append(ret.Tables, r)
				}

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

	// Resolve name conflicts in result fields.
	names := make(map[string]*ResultFieldMeta)
	for _, rfm := range ret.ResultFields {
		name := rfm.Name
		pascal_name := rfm.PascalName
		for i := 1; ; i += 1 {
			if _, ok := names[name]; !ok {
				break
			}
			name = fmt.Sprintf("%s_%d", rfm.Name, i)
			pascal_name = fmt.Sprintf("%s_%d", rfm.PascalName, i)
		}

		rfm.Name = name
		rfm.PascalName = pascal_name
		names[name] = rfm
	}

	// Extract table source meta.
	if sources, err := NewSelectTableSourcesMeta(ctx, stmt); err != nil {
		return nil, err
	} else {
		ret.Sources = sources
	}

	return ret, nil

}

// This function expand all wildcards ("*") in a SELECT statement and return
// an new equivalent one. This is useful since "SELECT * ..." may lead to
// unpredictable error when table is altered.
func ExpandWildcard(ctx *Context, stmt *ast.SelectStmt) (*ast.SelectStmt, error) {

	// Check wildcard.
	has_wildcard := false
	for _, f := range stmt.Fields.Fields {
		if f.WildCard != nil {
			has_wildcard = true
			break
		}
	}
	if !has_wildcard {
		return stmt, nil
	}

	db := ctx.DB
	origin := stmt.Text()

	// Re-parse and re-compile. Since we need Offset of wildcards which
	// needs to be re-calculated.
	if stmts, err := db.Parse(origin); err != nil {
		return nil, err
	} else {
		stmt = stmts[0].(*ast.SelectStmt)
		if _, err := db.Compile(stmt); err != nil {
			return nil, err
		}
	}

	// Extract table source meta.
	sources, err := NewSelectTableSourcesMeta(ctx, stmt)
	if err != nil {
		return nil, err
	}

	// Iter fields and replace all wildcards.
	offset := 0
	parts := make([]string, 0)
	for i, f := range stmt.Fields.Fields {
		// Not a wildcard field.
		if f.WildCard == nil {
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
			return nil, fmt.Errorf("%s strings.HasPrefix(%+q, %+q) == false", err_prefix,
				origin[f.Offset:], field_text)
		}

		// Move offset.
		offset = f.Offset + len(field_text)

		// Expand wildcard.
		expan := []string{}

		// Qualified wildcard ("[db.]tbl.*")
		if table_name != "" {

			rfs := sources.TableResultFields(table_name)
			if rfs == nil {
				panic(fmt.Errorf("%s No result fields for table %+q", err_prefix, table_name))
			}

			for j, rf := range rfs {
				rf_name, err := resultFieldNameAsIdentifier(rf, true)
				if err != nil {
					return nil, fmt.Errorf("%s resultFieldNameAsIdentifier for %+q[%d]: %s",
						err_prefix, table_name, j, err)
				}
				expan = append(expan, table_name+"."+rf_name)
			}

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
				for k, rf := range table.GetResultFields() {
					rf_name, err := resultFieldNameAsIdentifier(rf, true)
					if err != nil {
						return nil, fmt.Errorf("%s resultFieldNameAsIdentifier for %+q[%d]: %s",
							err_prefix, table_name, k, err)
					}
					expan = append(expan, table_name+"."+rf_name)
				}
			}
		}

		parts = append(parts, strings.Join(expan, ", "))

	}

	parts = append(parts, origin[offset:])
	text := strings.Join(parts, "")

	// Second-re-parse and re-compile.
	if stmts, err := db.Parse(text); err != nil {
		return nil, err
	} else {
		stmt = stmts[0].(*ast.SelectStmt)
		if _, err := db.Compile(stmt); err != nil {
			return nil, err
		}
	}

	return stmt, nil

}

func resultFieldNameAsIdentifier(rf *ast.ResultField, exact bool) (string, error) {

	rf_name := rf.ColumnAsName.L
	if rf_name == "" {
		rf_name = rf.Column.Name.L
	}
	if rf_name == "" {
		return "", fmt.Errorf("Empty result field name")
	}
	if utils.IsIdent(rf_name) {
		return rf_name, nil
	}
	if exact {
		return "", fmt.Errorf("%+q is not a valid identifier", rf_name)
	} else {
		rf_name1 := utils.FindIdent(rf_name)
		if rf_name1 == "" {
			return "", fmt.Errorf("Can't find a valid identifier in %+q", rf_name)
		}
		return rf_name1, nil
	}

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
