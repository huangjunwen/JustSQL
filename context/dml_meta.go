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

func NewResultFieldMeta(ctx *Context, rf *ast.ResultField) (*ResultFieldMeta, error) {

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

type SelectStmtMeta struct {
	*ast.SelectStmt

	ResultFields []*ResultFieldMeta

	Sources *SelectTableSourcesMeta
}

func NewSelectStmtMeta(ctx *Context, stmt *ast.SelectStmt) (*SelectStmtMeta, error) {

	// Create result fields meta.
	rfs := stmt.GetResultFields()
	ret := &SelectStmtMeta{
		SelectStmt:   stmt,
		ResultFields: make([]*ResultFieldMeta, 0, len(rfs)),
	}
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

	// Create table source meta.
	if sources, err := NewSelectTableSourcesMeta(ctx, stmt); err != nil {
		return nil, err
	} else {
		ret.Sources = sources
	}

	return ret, nil

}
