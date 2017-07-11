package handler

import (
	"fmt"
	"github.com/huangjunwen/JustSQL/context"
)

// The 'dot' object for renderring TableData
type TableDataDot struct {
	DB    *context.DBData
	Table *context.TableData
}

func handleTableData(ctx *context.Context, obj interface{}) (interface{}, error) {
	table, ok := obj.(*context.TableData)
	if !ok {
		return nil, fmt.Errorf("handleTableData: expect *context.TableData but got %T", obj)
	}

	db, err := ctx.DBData()
	if err != nil {
		return nil, err
	}

	return &TableDataDot{
		DB:    db,
		Table: table,
	}, nil
}

func init() {
	RegistHandler((*context.TableData)(nil), handleTableData)
}
