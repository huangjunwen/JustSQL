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
	Type *types.FieldType
}

func NewResultFieldMeta(ctx *Context, rf *ast.ResultField) (*ResultFieldMeta, error) {

	var err error

	// Determin the name.
	name, err := resultFieldName(rf, false)
	if err != nil {
		return nil, err
	}

	return &ResultFieldMeta{
		ResultField: rf,
		Name:        name,
		Type:        &rf.Column.FieldType,
	}, nil

}

func (rf *ResultFieldMeta) IsEnum() bool {
	return rf.Type.Tp == mysql.TypeEnum
}

func (rf *ResultFieldMeta) IsSet() bool {
	return rf.Type.Tp == mysql.TypeSet
}

func (rf *ResultFieldMeta) Elems() []string {
	return rf.Type.Elems
}

func (rf *ResultFieldMeta) IsNotNULL() bool {
	return mysql.HasNotNullFlag(rf.Type.Flag)
}

func (rf *ResultFieldMeta) IsAutoInc() bool {
	return mysql.HasAutoIncrementFlag(rf.Type.Flag)
}

func (rf *ResultFieldMeta) IsOnUpdateNow() bool {
	return mysql.HasOnUpdateNowFlag(rf.Type.Flag)
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
	TableSources  []*ast.TableSource
	TableRefNames []string

	// Not nil if TableSources[i] is normal table.
	TableMetas []*TableMeta

	// Map table ref name to its index.
	// Similar to github.com/tidb/plan/resolver.go:resolverContext,
	// but only use one map to enforce name uniqueness for both normal tables and derived tables.
	TableRefNameMap map[string]int
}

func NewTableRefsMeta(ctx *Context, refs *ast.TableRefsClause) (*TableRefsMeta, error) {

	ret := &TableRefsMeta{
		TableSources:    make([]*ast.TableSource, 0),
		TableRefNames:   make([]string, 0),
		TableMetas:      make([]*TableMeta, 0),
		TableRefNameMap: make(map[string]int),
	}

	// Refs can be nil: "SELECT 3"
	if refs == nil {
		return ret, nil
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
					tableMeta    *TableMeta
				)

				switch s := r.Source.(type) {
				case *ast.TableName:
					tableRefName = r.AsName.L
					if tableRefName == "" {
						tableRefName = ctx.UniqueTableName(s.Schema.L, s.Name.L)
					}
					if dbMeta, err := ctx.GetDBMeta(s.Schema.L); err != nil {
						return err
					} else {
						tableMeta = dbMeta.Tables[s.Name.L]
					}

				default:
					tableRefName = r.AsName.L
				}

				if tableRefName == "" {
					return fmt.Errorf("[bug?] No name for table source[%d]", len(ret.TableSources))
				}
				if _, ok := ret.TableRefNameMap[tableRefName]; ok {
					return fmt.Errorf("Duplicate table name/alias %+q, plz checkout JustSQL's document",
						tableRefName)
				}
				ret.TableRefNameMap[tableRefName] = len(ret.TableSources)
				ret.TableSources = append(ret.TableSources, r)
				ret.TableRefNames = append(ret.TableRefNames, tableRefName)
				ret.TableMetas = append(ret.TableMetas, tableMeta)

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
	for tableRefName, _ := range ret.TableRefNameMap {

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

	idx, ok := t.TableRefNameMap[tableRefName]
	if !ok {
		return nil
	}
	return t.TableSources[idx].Source.GetResultFields()

}

// TableMeta return *TableMeta if tableRefName references a normal table or nil
// if tableRefName references a derived table.
func (t *TableRefsMeta) TableMeta(tableRefName string) *TableMeta {

	idx, ok := t.TableRefNameMap[tableRefName]
	if !ok {
		return nil
	}
	return t.TableMetas[idx]

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

func (f *FieldListMeta) addWildcard(tableRefName string, resultFieldOffset int, resultFieldNum int) error {

	errPrefix := fmt.Sprintf("addWildcard(%+q, %d, %d)", tableRefName, resultFieldOffset,
		resultFieldNum)

	// Check index boundary.
	if resultFieldOffset < 0 {
		return fmt.Errorf("%s: resultFieldOffset==%d", errPrefix, resultFieldOffset)
	}
	if resultFieldNum <= 0 {
		return fmt.Errorf("%s: resultFieldNum==%d", errPrefix, resultFieldNum)
	}
	if resultFieldOffset+resultFieldNum > len(f.ResultFieldToWildcard) {
		return fmt.Errorf("%s: %d+%d>%d", errPrefix, resultFieldOffset, resultFieldNum,
			len(f.ResultFieldToWildcard))
	}

	// Mark.
	idx := len(f.Wildcards)
	for i := resultFieldOffset; i < resultFieldOffset+resultFieldNum; i++ {
		// Wildcard overlapped?
		if f.ResultFieldToWildcard[i] != -1 {
			return fmt.Errorf("%s: f.ResultFieldToWildcard[%d] == %d", errPrefix,
				i, f.ResultFieldToWildcard[i])
		}
		f.ResultFieldToWildcard[i] = idx
	}

	// Save wildcard meta.
	f.Wildcards = append(f.Wildcards, WildcardMeta{
		TableRefName:      tableRefName,
		ResultFieldOffset: resultFieldOffset,
		ResultFieldNum:    resultFieldNum,
	})

	return nil
}

// NewFieldListMeta create FieldListMeta from the field list and table refs of a SELECT statement.
func NewFieldListMeta(ctx *Context, stmt *ast.SelectStmt, tableRefsMeta *TableRefsMeta) (*FieldListMeta, error) {

	if err := ensureSelectStmtCompiled(ctx, stmt); err != nil {
		return nil, err
	}

	rfsn := len(stmt.GetResultFields())
	ret := &FieldListMeta{
		Wildcards:             make([]WildcardMeta, 0),
		ResultFieldToWildcard: make([]int, rfsn),
	}
	for i := 0; i < rfsn; i++ {
		ret.ResultFieldToWildcard[i] = -1
	}

	// Also see github.com/pingcap/tidb/plan/resolver.go createResultFields
	offset := 0
	for _, f := range stmt.Fields.Fields {
		if f.WildCard == nil {
			// Each one non-wildcard field -> one result field.
			offset += 1
			continue
		}
		// Qualified wildcard ("[db.]tbl.*")
		tableRefName := ctx.UniqueTableName(f.WildCard.Schema.L, f.WildCard.Table.L)
		if tableRefName != "" {
			rfs := tableRefsMeta.GetResultFields(tableRefName)
			if err := ret.addWildcard(tableRefName, offset, len(rfs)); err != nil {
				return nil, err
			}
			offset += len(rfs)
			continue
		}
		// Unqualified wildcard ("*")
		for i, table := range tableRefsMeta.TableSources {
			tableRefName := tableRefsMeta.TableRefNames[i]
			rfs := table.GetResultFields()
			if err := ret.addWildcard(tableRefName, offset, len(rfs)); err != nil {
				return nil, err
			}
			offset += len(rfs)
		}

	}

	if offset != rfsn {
		return nil, fmt.Errorf("[bug?] Field list expanded length(%d) != Result field length(%d)",
			offset, rfsn)
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

// WildcardColumnOffset return the offset inside wildcard of the i-th result field
// or -1 if the result field is not inside wildcard.
func (f *FieldListMeta) WildcardColumnOffset(i int) int {

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

	ret := &SelectStmtMeta{
		SelectStmt: stmt,
	}

	var err error

	// Execute the statement to verify it and also extract result fields meta.
	rs, err := ctx.DB.Execute(stmt.Text())
	if err != nil {
		return nil, err
	}
	if len(rs) < 1 {
		return nil, fmt.Errorf("NewSelectStmtMeta: No RecordSet return")
	}
	rfs, err := rs[0].Fields()
	if err != nil {
		return nil, err
	}
	ret.ResultFields = make([]*ResultFieldMeta, 0, len(rfs))
	for _, rf := range rfs {
		rfm, err := NewResultFieldMeta(ctx, rf)
		if err != nil {
			return nil, err
		}
		ret.ResultFields = append(ret.ResultFields, rfm)
	}

	// To extract more information from the SELECT statement itself, it needs to
	// be compiled.
	if err = ensureSelectStmtCompiled(ctx, stmt); err != nil {
		return nil, err
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

	// Some rough checks.
	// BUG: "SELECT u.*, justsql.u.* FROM user u"
	if len(ret.FieldList.ResultFieldToWildcard) != len(ret.ResultFields) {
		return nil, fmt.Errorf("[bug] Different result fields num %d != %d",
			len(ret.FieldList.ResultFieldToWildcard), len(ret.ResultFields))
	}

	return ret, nil

}

// This function expand all wildcards ("*") in a SELECT statement and return
// a new equivalent one. This is useful since "SELECT * ..." may lead to
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
	offset := 0
	for n, field := range stmt.Fields.Fields {

		if field.WildCard == nil {
			continue
		}

		errPrefix := fmt.Sprintf("Expand wildcard field[%d]:", n)

		// Save part before this wildcard field.
		parts = append(parts, text[offset:field.Offset])

		// Calculate this wildcard field length to move offset.
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
		offset = field.Offset + len(fieldText)

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
			for i, table := range s.TableRefs.TableSources {

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

	parts = append(parts, text[offset:])
	text = strings.Join(parts, "")

	// Second-re-parse and re-compile.
	if stmts, err := db.Parse(text); err != nil {
		return nil, err
	} else {
		stmt = stmts[0].(*ast.SelectStmt)
	}

	// Create new stmt meta.
	ret, err := NewSelectStmtMeta(ctx, stmt)
	if err != nil {
		return nil, err
	}

	// Rough checks.
	if len(s.ResultFields) != len(ret.ResultFields) {
		return nil, fmt.Errorf("[bug] Wildcard expanded different result fields num %d != %d",
			len(s.ResultFields), len(ret.ResultFields))
	}
	return ret, nil

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
