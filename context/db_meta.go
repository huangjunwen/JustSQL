package context

import (
	"fmt"
	"github.com/huangjunwen/JustSQL/utils"
	"github.com/pingcap/tidb/model"
	"github.com/pingcap/tidb/mysql"
	"github.com/pingcap/tidb/util/types"
)

// This file contains xxxMeta types. They are used to store extracted
// meta information from tidb model.xxxInfo.

// All names used here should use the lower case version.

// DBMeta contains meta information of a database.
type DBMeta struct {
	*model.DBInfo

	Name   string
	Tables map[string]*TableMeta
}

func NewDBMeta(ctx *Context, db_info *model.DBInfo) (*DBMeta, error) {
	ret := &DBMeta{
		DBInfo: db_info,
		Name:   db_info.Name.L,
		Tables: make(map[string]*TableMeta),
	}
	for _, table_info := range db_info.Tables {
		table_meta, err := NewTableMeta(ctx, table_info)
		if err != nil {
			return nil, err
		}
		ret.Tables[table_meta.Name] = table_meta
	}
	return ret, nil
}

func (db *DBMeta) PascalName() string {
	return utils.PascalCase(db.Name)
}

// TableMeta contains meta information of a table.
type TableMeta struct {
	*model.TableInfo

	Name        string
	Columns     []*ColumnMeta
	Indices     []*IndexMeta
	ForeignKeys []*FKMeta

	// Shortcut
	primaryIndex  *IndexMeta
	autoIncColumn *ColumnMeta

	// column name -> column index
	columnByName map[string]*ColumnMeta
}

func NewTableMeta(ctx *Context, table_info *model.TableInfo) (*TableMeta, error) {
	ret := &TableMeta{
		TableInfo:    table_info,
		Name:         table_info.Name.L,
		Columns:      make([]*ColumnMeta, 0, len(table_info.Columns)),
		Indices:      make([]*IndexMeta, 0, len(table_info.Indices)),
		ForeignKeys:  make([]*FKMeta, 0, len(table_info.ForeignKeys)),
		columnByName: make(map[string]*ColumnMeta),
	}

	for _, column_info := range table_info.Columns {
		column_meta, err := NewColumnMeta(ctx, column_info)
		if err != nil {
			return nil, err
		}
		ret.Columns = append(ret.Columns, column_meta)
		ret.columnByName[column_meta.Name] = column_meta

		if column_meta.IsAutoInc {
			if ret.autoIncColumn != nil {
				panic(fmt.Errorf("Multiple auto increment columns found in table %q", ret.Name))
			}
			ret.autoIncColumn = column_meta
		}
	}

	if index_meta := NewIndexMetaFromPKHandler(ctx, table_info); index_meta != nil {
		ret.Indices = append(ret.Indices, index_meta)
		ret.primaryIndex = index_meta
	}

	for _, index_info := range table_info.Indices {
		index_meta, err := NewIndexMeta(ctx, index_info)
		if err != nil {
			return nil, err
		}
		ret.Indices = append(ret.Indices, index_meta)
		if index_meta.Primary {
			if ret.primaryIndex != nil {
				panic(fmt.Errorf("Multiple primary index found in table %q", ret.Name))
			}
			ret.primaryIndex = index_meta
		}
	}

	for _, fk_info := range table_info.ForeignKeys {
		fk_meta, err := NewFKMeta(ctx, fk_info)
		if err != nil {
			return nil, err
		}
		ret.ForeignKeys = append(ret.ForeignKeys, fk_meta)
	}

	return ret, nil
}

func (t *TableMeta) PascalName() string {
	return utils.PascalCase(t.Name)
}

// Return primary key columns if exists.
func (t *TableMeta) PrimaryColumns() []*ColumnMeta {
	if t.primaryIndex == nil {
		return nil
	}
	ret := make([]*ColumnMeta, 0, len(t.primaryIndex.ColumnIndices))
	for _, col_idx := range t.primaryIndex.ColumnIndices {
		ret = append(ret, t.Columns[col_idx])
	}
	return ret
}

// Return non-primary columns if exists.
func (t *TableMeta) NonPrimaryColumns() []*ColumnMeta {
	if t.primaryIndex == nil {
		return t.Columns
	}
	is_primary := make([]bool, len(t.Columns))
	for _, col_idx := range t.primaryIndex.ColumnIndices {
		is_primary[col_idx] = true
	}

	ret := make([]*ColumnMeta, 0, len(t.Columns)-len(t.primaryIndex.ColumnIndices))
	for i, b := range is_primary {
		if b {
			continue
		}
		ret = append(ret, t.Columns[i])
	}
	return ret

}

// Return auto increment column if exists.
func (t *TableMeta) AutoIncColumn() *ColumnMeta {
	return t.autoIncColumn
}

// Retrive column by its name.
func (t *TableMeta) ColumnByName(name string) *ColumnMeta {
	ret, ok := t.columnByName[name]
	if !ok {
		return nil
	}
	return ret
}

// ColumnMeta contains meta data of a column.
type ColumnMeta struct {
	*model.ColumnInfo

	Name   string
	Offset int

	// Column field type.
	Type types.FieldType

	// Is it enum/set type?
	IsEnum bool
	IsSet  bool

	// Element list if IsEnum == true or IsSet == true
	Elems []string

	// Flags of this column
	IsNotNULL     bool
	IsAutoInc     bool
	IsOnUpdateNow bool

	DefaultValue interface{}

	// Go type to store this field.
	AdaptType *TypeName
}

func NewColumnMeta(ctx *Context, column_info *model.ColumnInfo) (*ColumnMeta, error) {
	ret := &ColumnMeta{
		ColumnInfo: column_info,
		Name:       column_info.Name.L,
		Offset:     column_info.Offset,
		Type:       column_info.FieldType,
	}

	tp := &ret.Type
	ret.AdaptType = ctx.TypeAdapter.AdaptType(tp)

	if tp.Tp == mysql.TypeEnum {
		ret.IsEnum = true
		ret.Elems = tp.Elems
	} else if tp.Tp == mysql.TypeSet {
		ret.IsSet = true
		ret.Elems = tp.Elems
	}

	ret.IsNotNULL = mysql.HasNotNullFlag(tp.Flag)
	ret.IsAutoInc = mysql.HasAutoIncrementFlag(tp.Flag)
	ret.IsOnUpdateNow = mysql.HasOnUpdateNowFlag(tp.Flag)

	ret.DefaultValue = column_info.DefaultValue

	return ret, nil
}

func (c *ColumnMeta) PascalName() string {
	return utils.PascalCase(c.Name)
}

// IndexMeta contains meta data of an index.
type IndexMeta struct {
	*model.IndexInfo

	Name          string
	Unique        bool
	Primary       bool
	ColumnIndices []int
}

func NewIndexMeta(ctx *Context, index_info *model.IndexInfo) (*IndexMeta, error) {
	ret := &IndexMeta{
		IndexInfo:     index_info,
		Name:          index_info.Name.L,
		Unique:        index_info.Unique,
		Primary:       index_info.Primary,
		ColumnIndices: make([]int, 0, len(index_info.Columns)),
	}
	for _, column := range index_info.Columns {
		ret.ColumnIndices = append(ret.ColumnIndices, column.Offset)
	}
	return ret, nil
}

// see: https://github.com/pingcap/tidb/issues/3746
func NewIndexMetaFromPKHandler(ctx *Context, table_info *model.TableInfo) *IndexMeta {
	if !table_info.PKIsHandle {
		return nil
	}
	column_info := table_info.GetPkColInfo()
	return &IndexMeta{
		Name:          "primary",
		Unique:        true,
		Primary:       true,
		ColumnIndices: []int{column_info.Offset},
	}
}

func (i *IndexMeta) PascalName() string {
	return utils.PascalCase(i.Name)
}

// FKMeta contains meta data of a foreign key.
type FKMeta struct {
	*model.FKInfo

	Name         string
	ColNames     []string
	RefTableName string
	RefColNames  []string
}

func NewFKMeta(ctx *Context, fk_info *model.FKInfo) (*FKMeta, error) {
	ret := &FKMeta{
		Name:         fk_info.Name.L,
		FKInfo:       fk_info,
		ColNames:     make([]string, 0, len(fk_info.Cols)),
		RefTableName: fk_info.RefTable.L,
		RefColNames:  make([]string, 0, len(fk_info.RefCols)),
	}
	for _, col := range fk_info.Cols {
		ret.ColNames = append(ret.ColNames, col.L)
	}
	for _, refcol := range fk_info.RefCols {
		ret.RefColNames = append(ret.RefColNames, refcol.L)
	}
	return ret, nil
}

func (fk *FKMeta) PascalName() string {
	return utils.PascalCase(fk.Name)
}
