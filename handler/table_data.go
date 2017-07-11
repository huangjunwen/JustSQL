package handler

import (
	"fmt"
	"github.com/huangjunwen/JustSQL/context"
)

func handleTableData(ctx *context.Context, obj interface{}) (interface{}, error) {
	table, ok := obj.(*context.TableData)
	if !ok {
		return nil, fmt.Errorf("handleTableData: expect *context.TableData but got %T", obj)
	}

	db, err := ctx.DBData()
	if err != nil {
		return nil, err
	}

	// The 'dot' object to render TableData
	return map[string]interface{}{
		"DB":    db,
		"Table": table,
	}, nil
}

func init() {
	RegistHandler((*context.TableData)(nil), handleTableData)
}
