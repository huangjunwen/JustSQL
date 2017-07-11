package context

import (
	"github.com/huangjunwen/JustSQL/utils"
	"github.com/pingcap/tidb/model"
	"github.com/pingcap/tidb/mysql"
)

// This file contains xxxData types. They are used to store extracted
// meta data from tidb model.xxxInfo.

// DBData contains meta data of a database.
type DBData struct {
	*model.DBInfo

	Name   utils.Str
	Tables map[string]*TableData
}

func NewDBData(ctx *Context, dbinfo *model.DBInfo) (*DBData, error) {
	ret := &DBData{
		DBInfo: dbinfo,
		Name:   utils.NewStrFromCIStr(dbinfo.Name),
		Tables: make(map[string]*TableData),
	}
	for _, tableinfo := range dbinfo.Tables {
		tabledata, err := NewTableData(ctx, tableinfo)
		if err != nil {
			return nil, err
		}
		ret.Tables[tabledata.Name.O] = tabledata
	}
	return ret, nil
}

// TableData contains meta data of a table.
type TableData struct {
	*model.TableInfo

	Name        utils.Str
	Columns     []*ColumnData
	Indices     []*IndexData
	ForeignKeys []*FKData
	Primary     *IndexData

	// column name -> column index
	columnIndex map[string]int
}

func NewTableData(ctx *Context, tableinfo *model.TableInfo) (*TableData, error) {
	ret := &TableData{
		TableInfo:   tableinfo,
		Name:        utils.NewStrFromCIStr(tableinfo.Name),
		Columns:     make([]*ColumnData, 0, len(tableinfo.Columns)),
		Indices:     make([]*IndexData, 0, len(tableinfo.Indices)),
		ForeignKeys: make([]*FKData, 0, len(tableinfo.ForeignKeys)),
		columnIndex: make(map[string]int),
	}

	for i, columninfo := range tableinfo.Columns {
		columndata, err := NewColumnData(ctx, columninfo)
		if err != nil {
			return nil, err
		}
		ret.Columns = append(ret.Columns, columndata)
		ret.columnIndex[columndata.Name.O] = i
	}

	for _, indexinfo := range tableinfo.Indices {
		indexdata, err := NewIndexData(ctx, indexinfo)
		if err != nil {
			return nil, err
		}
		ret.Indices = append(ret.Indices, indexdata)
		if indexdata.Primary {
			ret.Primary = indexdata
		}
	}

	for _, fkinfo := range tableinfo.ForeignKeys {
		fkdata, err := NewFKData(ctx, fkinfo)
		if err != nil {
			return nil, err
		}
		ret.ForeignKeys = append(ret.ForeignKeys, fkdata)
	}

	return ret, nil
}

// Retrive column by its name.
func (t *TableData) ColumnByName(name string) *ColumnData {
	i, ok := t.columnIndex[name]
	if !ok {
		return nil
	}
	return t.Columns[i]
}

// ColumnData contains meta data of a column.
type ColumnData struct {
	*model.ColumnInfo

	Name   utils.Str
	Offset int

	Type *TypeName

	// Is it enum/set type?
	IsEnum bool
	IsSet  bool

	// Element list if IsEnum == true or IsSet == true
	Elems []string
}

func NewColumnData(ctx *Context, columninfo *model.ColumnInfo) (*ColumnData, error) {
	ret := &ColumnData{
		ColumnInfo: columninfo,
		Name:       utils.NewStrFromCIStr(columninfo.Name),
		Offset:     columninfo.Offset,
	}

	tp, err := ctx.TypeContext.AdaptType(&columninfo.FieldType)
	if err != nil {
		return nil, err
	}
	ret.Type = tp

	if columninfo.FieldType.Tp == mysql.TypeEnum {
		ret.IsEnum = true
		ret.Elems = columninfo.FieldType.Elems
	} else if columninfo.FieldType.Tp == mysql.TypeSet {
		ret.IsSet = true
		ret.Elems = columninfo.FieldType.Elems
	}

	return ret, nil
}

// IndexData contains meta data of an index.
type IndexData struct {
	*model.IndexInfo

	Name          utils.Str
	Unique        bool
	Primary       bool
	ColumnIndices []int
}

func NewIndexData(ctx *Context, indexinfo *model.IndexInfo) (*IndexData, error) {
	ret := &IndexData{
		IndexInfo:     indexinfo,
		Name:          utils.NewStrFromCIStr(indexinfo.Name),
		Unique:        indexinfo.Unique,
		Primary:       indexinfo.Primary,
		ColumnIndices: make([]int, 0, len(indexinfo.Columns)),
	}
	for _, column := range indexinfo.Columns {
		ret.ColumnIndices = append(ret.ColumnIndices, column.Offset)
	}
	return ret, nil
}

// FKData contains meta data of a foreign key.
type FKData struct {
	*model.FKInfo

	Name         utils.Str
	ColNames     []string
	RefTableName string
	RefColNames  []string
}

func NewFKData(ctx *Context, fkinfo *model.FKInfo) (*FKData, error) {
	ret := &FKData{
		FKInfo:       fkinfo,
		ColNames:     make([]string, 0, len(fkinfo.Cols)),
		RefTableName: fkinfo.RefTable.O,
		RefColNames:  make([]string, 0, len(fkinfo.RefCols)),
	}
	for _, col := range fkinfo.Cols {
		ret.ColNames = append(ret.ColNames, col.O)
	}
	for _, refcol := range fkinfo.RefCols {
		ret.RefColNames = append(ret.RefColNames, refcol.O)
	}
	return ret, nil
}
