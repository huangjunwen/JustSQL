package handler

import (
	"fmt"
	"github.com/huangjunwen/JustSQL/context"
)

func handleTableMeta(ctx *context.Context, obj interface{}) (interface{}, error) {

	table_meta, ok := obj.(*context.TableMeta)
	if !ok {
		return nil, fmt.Errorf("handleTableMeta: expect *context.TableMeta but got %T", obj)
	}

	// The 'dot' object to render TableMeta
	return map[string]interface{}{
		"Table": table_meta,
	}, nil
}

func init() {
	RegistHandler((*context.TableMeta)(nil), handleTableMeta)
}
