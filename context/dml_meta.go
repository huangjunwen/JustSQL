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

	var table *TableMeta = nil
	var column *ColumnMeta = nil

	// Real TableInfo has name.
	if rf.Table.Name.L != "" {
		dbMeta, err := ctx.GetDBMeta(rf.DBName.L)
		if err != nil {
			return nil, err
		}
		table = dbMeta.Tables[rf.Table.Name.L]
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

// TableRefsMeta contains meta information of table references (e.g. 'FROM' part of SELECT statement).
//
// CASE 1:
//  MySQL allows a normal table and a derived table sharing a same name:
//     "SELECT a.* FROM t AS a JOIN (SELECT 1) AS a;"
//   "a.*" expands to all columns of the two tables.
//   Also see comments of github.com/pingcap/tidb/plan/resolver.go:handleTableSource
//
// CASE 2:
//   "SELECT user.* FROM user, mysql.user" is also allowed, "user.*" also expands to all
//   columns of the two tables. This may lead to confusion.
//
// Due to the reasons above, JustSQL enforce table referernce names' uniqueness, even in different
// database ("mysql.user" and "user"). Thus the above SQLs should modified to something like:
//   "SELECT a1.*, a2.* FROM t AS a1 JOIN (SELECT 1) AS a2;
//   "SELECT u1.*, u2.* FROM user u1, mysql.user u2"
type TableRefsMeta struct {
	// List of table sources and its reference name. Reference name can be the following:
	//   - tbl_name
	//   - other_database.tbl_name
	//   - alias
	Tables         []*ast.TableSource
	TableRefNames  []string
	TableIsDerived []bool

	// Map table ref name to its index.
	// Similar to github.com/tidb/plan/resolver.go:resolverContext,
	// but only use one map to enforce name uniqueness for normal tables and derived tables.
	TableMap map[string]int
}

func NewTableRefsMeta(ctx *Context, refs *ast.TableRefsClause) (*TableRefsMeta, error) {

	ret := &TableRefsMeta{
		Tables:         make([]*ast.TableSource, 0),
		TableRefNames:  make([]string, 0),
		TableIsDerived: make([]bool, 0),
		TableMap:       make(map[string]int),
	}

	var collect func(*ast.Join) error

	collect = func(j *ast.Join) error {
		// Left then right
		for _, rs := range [2]ast.ResultSetNode{j.Left, j.Right} {
			if rs == nil {
				continue
			}
			switch r := rs.(type) {
			case *ast.TableSource:

				var (
					tableRefName string
					isDerived    bool = false
				)

				switch s := r.Source.(type) {
				case *ast.TableName:
					tableRefName = r.AsName.L
					if tableRefName == "" {
						tableRefName = ctx.UniqueTableName(s.Schema.L, s.Name.L)
					}

				default:
					tableRefName = r.AsName.L
					isDerived = true
				}

				if tableRefName == "" {
					return fmt.Errorf("[bug?] No name for table source[%d]", len(ret.Tables))
				}
				if _, ok := ret.TableMap[tableRefName]; ok {
					return fmt.Errorf("Duplicate table name/alias %+q, plz checkout JustSQL's document",
						tableRefName)
				}
				ret.TableMap[tableRefName] = len(ret.Tables)
				ret.Tables = append(ret.Tables, r)
				ret.TableRefNames = append(ret.TableRefNames, tableRefName)
				ret.TableIsDerived = append(ret.TableIsDerived, isDerived)

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

	// Check name duplication for CASE 2
	names := map[string]string{}
	for tableRefName, _ := range ret.TableMap {

		parts := strings.Split(tableRefName, ".")
		name := ""
		switch len(parts) {
		case 1:
			name = tableRefName
		case 2:
			name = parts[1]
		default:
			panic(fmt.Errorf("Invalid table ref name %+q", tableRefName))
		}

		if existName, ok := names[name]; ok {
			return nil, fmt.Errorf("Plz add alias to table ref name %+q or %+q", existName,
				tableRefName)
		}
		names[name] = tableRefName

	}
	return ret, nil

}

// GetResultFields return a list of result field of the table or nil if not exists.
func (t *TableRefsMeta) GetResultFields(tableRefName string) []*ast.ResultField {

	idx, ok := t.TableMap[tableRefName]
	if !ok {
		return nil
	}
	return t.Tables[idx].Source.GetResultFields()

}

func (t *TableRefsMeta) IsNormalTable(tableRefName string) bool {
	i, ok := t.TableMap[tableRefName]
	if !ok {
		return false
	}
	return !t.TableIsDerived[i]
}

func (t *TableRefsMeta) IsDerivedTable(tableRefName string) bool {
	i, ok := t.TableMap[tableRefName]
	if !ok {
		return false
	}
	return t.TableIsDerived[i]
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

func (f *FieldListMeta) addWildcard(tableRefName string, resultFieldOffset int, resultFieldNum int) {

	idx := len(f.Wildcards)
	for i := resultFieldOffset; i < resultFieldOffset+resultFieldNum; i++ {
		if f.ResultFieldToWildcard[i] != -1 {
			panic(fmt.Errorf("f.ResultFieldToWildcard[%d] == %d != -1", i, f.ResultFieldToWildcard[i]))
		}
		f.ResultFieldToWildcard[i] = idx
	}
	f.Wildcards = append(f.Wildcards, WildcardMeta{
		TableRefName:      tableRefName,
		ResultFieldOffset: resultFieldOffset,
		ResultFieldNum:    resultFieldNum,
	})

}

// NewFieldListMeta create FieldListMeta from the field list and table refs of a SELECT statement.
func NewFieldListMeta(ctx *Context, stmt *ast.SelectStmt, tableRefsMeta *TableRefsMeta) (*FieldListMeta, error) {

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
		tableRefName := ctx.UniqueTableName(f.WildCard.Schema.L, f.WildCard.Table.L)
		if tableRefName != "" {
			rfs := tableRefsMeta.GetResultFields(tableRefName)
			if len(rfs) == 0 {
				return nil, fmt.Errorf("[bug?] TableRefsMeta.GetResultFields(%+q) == nil", tableRefName)
			}
			ret.addWildcard(tableRefName, offset, len(rfs))
			offset += len(rfs)
			continue
		}
		// Unqualified wildcard ("*")
		for i, table := range tableRefsMeta.Tables {
			tableRefName := tableRefsMeta.TableRefNames[i]
			rfs := table.GetResultFields()
			if len(rfs) == 0 {
				return nil, fmt.Errorf("[bug?] Table[%+q].GetResultFields() == nil", tableRefName)
			}
			ret.addWildcard(tableRefName, offset, len(rfs))
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
	textOffset := 0
	for n, field := range stmt.Fields.Fields {

		if field.WildCard == nil {
			continue
		}

		errPrefix := fmt.Sprintf("Expand wildcard field[%d]:", n)

		// Save part before this wildcard field.
		parts = append(parts, text[textOffset:field.Offset])

		// Calculate this wildcard field length to move textOffset.
		// XXX: field.Text() return "" so i need to construct the field text myself -_-
		fieldText := "*"
		if field.WildCard.Table.O != "" {
			fieldText = field.WildCard.Table.O + ".*"
			if field.WildCard.Schema.O != "" {
				fieldText = field.WildCard.Schema.O + "." + fieldText
			}
		}
		if !strings.HasPrefix(text[field.Offset:], fieldText) {
			return nil, fmt.Errorf("%s strings.HasPrefix(%+q, %+q) == false", errPrefix,
				text[field.Offset:], fieldText)
		}
		textOffset = field.Offset + len(fieldText)

		// Expand wildcard.
		tableRefName := ctx.UniqueTableName(field.WildCard.Schema.L, field.WildCard.Table.L)
		expandParts := []string{}

		if tableRefName != "" {

			// Qualified wildcard ("[db.]tbl.*")
			rfs := s.TableRefs.GetResultFields(tableRefName)
			if rfs == nil {
				panic(fmt.Errorf("%s No result fields for table %+q", errPrefix, tableRefName))
			}

			for i, rf := range rfs {
				rfName, err := resultFieldName(rf, true)
				if err != nil {
					return nil, fmt.Errorf("%s resultFieldName for %+q[%d]: %s",
						errPrefix, tableRefName, i, err)
				}
				expandParts = append(expandParts, tableRefName+"."+rfName)
			}

		} else {

			// Unqualified wildcard ("*")
			for i, table := range s.TableRefs.Tables {

				tableRefName = s.TableRefs.TableRefNames[i]
				rfs := table.GetResultFields()

				for j, rf := range rfs {
					rfName, err := resultFieldName(rf, true)
					if err != nil {
						return nil, fmt.Errorf("%s resultFieldName for %+q[%d]: %s",
							errPrefix, tableRefName, j, err)
					}
					expandParts = append(expandParts, tableRefName+"."+rfName)
				}
			}
		}

		parts = append(parts, strings.Join(expandParts, ", "))

	}

	parts = append(parts, text[textOffset:])
	text = strings.Join(parts, "")

	// Second-re-parse and re-compile.
	if stmts, err := db.Parse(text); err != nil {
		return nil, err
	} else {
		stmt = stmts[0].(*ast.SelectStmt)
	}

	return NewSelectStmtMeta(ctx, stmt)

}

// InsertStmtMeta contains meta information of a INSERT statement.
type InsertStmtMeta struct {
	*ast.InsertStmt
}

// NewInsertStmtMeta create InsertStmtMeta from *ast.InsertStmt.
func NewInsertStmtMeta(ctx *Context, stmt *ast.InsertStmt) (*InsertStmtMeta, error) {
	return &InsertStmtMeta{
		InsertStmt: stmt,
	}, nil
}

// DeleteStmtMeta contains meta information of a DELETE statement.
type DeleteStmtMeta struct {
	*ast.DeleteStmt
}

// NewDeleteStmtMeta create DeleteStmtMeta from *ast.DeleteStmt.
func NewDeleteStmtMeta(ctx *Context, stmt *ast.DeleteStmt) (*DeleteStmtMeta, error) {
	return &DeleteStmtMeta{
		DeleteStmt: stmt,
	}, nil
}

// UpdateStmtMeta contains meta information of a UPDATE statement.
type UpdateStmtMeta struct {
	*ast.UpdateStmt
}

// NewUpdateStmtMeta create UpdateStmtMeta from *ast.UpdateStmt.
func NewUpdateStmtMeta(ctx *Context, stmt *ast.UpdateStmt) (*UpdateStmtMeta, error) {
	return &UpdateStmtMeta{
		UpdateStmt: stmt,
	}, nil
}

func resultFieldName(rf *ast.ResultField, asIdentifier bool) (string, error) {

	rfName := rf.ColumnAsName.L
	if rfName == "" {
		rfName = rf.Column.Name.L
	}
	if rfName == "" {
		return "", fmt.Errorf("Empty result field name")
	}
	if !asIdentifier {
		return rfName, nil
	}
	if utils.IsIdent(rfName) {
		return rfName, nil
	}
	return "", fmt.Errorf("%+q is not a valid identifier", rfName)

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
