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

	Name       string
	PascalName string

	Tables map[string]*TableMeta
}

func NewDBMeta(ctx *Context, dbInfo *model.DBInfo) (*DBMeta, error) {
	ret := &DBMeta{
		DBInfo:     dbInfo,
		Name:       dbInfo.Name.L,
		PascalName: utils.PascalCase(dbInfo.Name.L),
		Tables:     make(map[string]*TableMeta),
	}
	for _, tableInfo := range dbInfo.Tables {
		tableMeta, err := NewTableMeta(ctx, ret, tableInfo)
		if err != nil {
			return nil, err
		}
		ret.Tables[tableMeta.Name] = tableMeta
	}
	return ret, nil
}

// TableMeta contains meta information of a table.
type TableMeta struct {
	*model.TableInfo

	DB *DBMeta

	Name       string
	PascalName string

	Columns     []*ColumnMeta
	Indices     []*IndexMeta
	ForeignKeys []*FKMeta

	// Shortcut
	primaryIndex  *IndexMeta
	autoIncColumn *ColumnMeta

	// column name -> column index
	columnByName map[string]*ColumnMeta
}

func NewTableMeta(ctx *Context, dbMeta *DBMeta, tableInfo *model.TableInfo) (*TableMeta, error) {
	ret := &TableMeta{
		TableInfo:    tableInfo,
		DB:           dbMeta,
		Name:         tableInfo.Name.L,
		PascalName:   utils.PascalCase(tableInfo.Name.L),
		Columns:      make([]*ColumnMeta, 0, len(tableInfo.Columns)),
		Indices:      make([]*IndexMeta, 0, len(tableInfo.Indices)),
		ForeignKeys:  make([]*FKMeta, 0, len(tableInfo.ForeignKeys)),
		columnByName: make(map[string]*ColumnMeta),
	}

	for _, columnInfo := range tableInfo.Columns {
		columnMeta, err := NewColumnMeta(ctx, ret, columnInfo)
		if err != nil {
			return nil, err
		}
		ret.Columns = append(ret.Columns, columnMeta)
		ret.columnByName[columnMeta.Name] = columnMeta

		if columnMeta.IsAutoInc() {
			if ret.autoIncColumn != nil {
				panic(fmt.Errorf("Multiple auto increment columns found in table %q", ret.Name))
			}
			ret.autoIncColumn = columnMeta
		}
	}

	if indexMeta := NewIndexMetaFromPKHandler(ctx, ret); indexMeta != nil {
		ret.Indices = append(ret.Indices, indexMeta)
		ret.primaryIndex = indexMeta
	}

	for _, indexInfo := range tableInfo.Indices {
		indexMeta, err := NewIndexMeta(ctx, ret, indexInfo)
		if err != nil {
			return nil, err
		}
		ret.Indices = append(ret.Indices, indexMeta)
		if indexMeta.Primary {
			if ret.primaryIndex != nil {
				panic(fmt.Errorf("Multiple primary index found in table %q", ret.Name))
			}
			ret.primaryIndex = indexMeta
		}
	}

	for _, fkInfo := range tableInfo.ForeignKeys {
		fkMeta, err := NewFKMeta(ctx, ret, fkInfo)
		if err != nil {
			return nil, err
		}
		ret.ForeignKeys = append(ret.ForeignKeys, fkMeta)
	}

	return ret, nil
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

	Table *TableMeta

	Name       string
	PascalName string

	Offset int

	// Column field type.
	Type *types.FieldType
}

func NewColumnMeta(ctx *Context, tableMeta *TableMeta, columnInfo *model.ColumnInfo) (*ColumnMeta, error) {
	return &ColumnMeta{
		ColumnInfo: columnInfo,
		Table:      tableMeta,
		Name:       columnInfo.Name.L,
		PascalName: utils.PascalCase(columnInfo.Name.L),
		Offset:     columnInfo.Offset,
		Type:       &columnInfo.FieldType,
	}, nil
}

func (c *ColumnMeta) IsEnum() bool {
	return c.Type.Tp == mysql.TypeEnum
}

func (c *ColumnMeta) IsSet() bool {
	return c.Type.Tp == mysql.TypeSet
}

func (c *ColumnMeta) Elems() []string {
	return c.Type.Elems
}

func (c *ColumnMeta) IsNotNULL() bool {
	return mysql.HasNotNullFlag(c.Type.Flag)
}

func (c *ColumnMeta) IsAutoInc() bool {
	return mysql.HasAutoIncrementFlag(c.Type.Flag)
}

func (c *ColumnMeta) IsOnUpdateNow() bool {
	return mysql.HasOnUpdateNowFlag(c.Type.Flag)
}

func (c *ColumnMeta) DefaultValue() interface{} {
	return c.ColumnInfo.DefaultValue
}

// IndexMeta contains meta data of an index.
type IndexMeta struct {
	// Nil if it is created from PKHandler.
	*model.IndexInfo

	Table *TableMeta

	Name       string
	PascalName string

	Unique        bool
	Primary       bool
	ColumnIndices []int
}

func NewIndexMeta(ctx *Context, tableMeta *TableMeta, indexInfo *model.IndexInfo) (*IndexMeta, error) {
	ret := &IndexMeta{
		IndexInfo:     indexInfo,
		Table:         tableMeta,
		Name:          indexInfo.Name.L,
		PascalName:    utils.PascalCase(indexInfo.Name.L),
		Unique:        indexInfo.Unique,
		Primary:       indexInfo.Primary,
		ColumnIndices: make([]int, 0, len(indexInfo.Columns)),
	}
	for _, column := range indexInfo.Columns {
		ret.ColumnIndices = append(ret.ColumnIndices, column.Offset)
	}
	return ret, nil
}

// see: https://github.com/pingcap/tidb/issues/3746
func NewIndexMetaFromPKHandler(ctx *Context, tableMeta *TableMeta) *IndexMeta {
	tableInfo := tableMeta.TableInfo
	if !tableInfo.PKIsHandle {
		return nil
	}
	columnInfo := tableInfo.GetPkColInfo()
	return &IndexMeta{
		Table:         tableMeta,
		Name:          "primary",
		Unique:        true,
		Primary:       true,
		ColumnIndices: []int{columnInfo.Offset},
	}
}

// FKMeta contains meta data of a foreign key.
type FKMeta struct {
	*model.FKInfo

	Table *TableMeta

	Name       string
	PascalName string

	ColNames     []string
	RefTableName string
	RefColNames  []string
}

func NewFKMeta(ctx *Context, tableMeta *TableMeta, fkInfo *model.FKInfo) (*FKMeta, error) {
	ret := &FKMeta{
		FKInfo:       fkInfo,
		Table:        tableMeta,
		Name:         fkInfo.Name.L,
		PascalName:   utils.PascalCase(fkInfo.Name.L),
		ColNames:     make([]string, 0, len(fkInfo.Cols)),
		RefTableName: fkInfo.RefTable.L,
		RefColNames:  make([]string, 0, len(fkInfo.RefCols)),
	}
	for _, col := range fkInfo.Cols {
		ret.ColNames = append(ret.ColNames, col.L)
	}
	for _, refcol := range fkInfo.RefCols {
		ret.RefColNames = append(ret.RefColNames, refcol.L)
	}
	return ret, nil
}
