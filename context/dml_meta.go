package context

import (
	"fmt"
	"github.com/huangjunwen/JustSQL/utils"
	"github.com/pingcap/tidb/ast"
	"github.com/pingcap/tidb/mysql"
	"github.com/pingcap/tidb/util/types"
)

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
	// The following fields have values.
	Table  *TableMeta
	Column *ColumnMeta
}

func newResultFieldMeta(ctx *Context, rf *ast.ResultField) (*ResultFieldMeta, error) {

	db_meta, err := ctx.DBMeta()
	if err != nil {
		return nil, err
	}

	// Is it from a real table?
	var table *TableMeta = nil
	var column *ColumnMeta = nil
	table, _ = db_meta.Tables[rf.Table.Name.L]
	if table != nil {
		column = table.Columns[rf.Column.Offset]
	}

	// Determin the name.
	name := rf.ColumnAsName.L
	if name == "" {
		name = rf.Column.Name.L
	}
	if name == "" {
		return nil, fmt.Errorf("No name for *ast.ResultField")
	} else if !utils.IsIdent(name) {
		name = utils.FindIdent(name)
		if name == "" {
			return nil, fmt.Errorf(
				"Can't find a valid identifier in %+q, you can add 'AS alias'", name)
		}
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

type SelectStmtMeta struct {
	*ast.SelectStmt

	ResultFields []*ResultFieldMeta
}

func NewSelectStmtMeta(ctx *Context, stmt *ast.SelectStmt) (*SelectStmtMeta, error) {

	rfs := stmt.GetResultFields()
	ret := &SelectStmtMeta{
		SelectStmt:   stmt,
		ResultFields: make([]*ResultFieldMeta, 0, len(rfs)),
	}
	for _, rf := range rfs {
		rfm, err := newResultFieldMeta(ctx, rf)
		if err != nil {
			return nil, err
		}
		ret.ResultFields = append(ret.ResultFields, rfm)
	}

	return ret, nil

}
