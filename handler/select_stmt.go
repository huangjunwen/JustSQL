package handler

import (
	"fmt"
	"github.com/huangjunwen/JustSQL/context"
	"github.com/pingcap/tidb/ast"
)

func handleSelectStmt(ctx *context.Context, obj interface{}) (interface{}, error) {

	origin_select_stmt, ok := obj.(*ast.SelectStmt)
	if !ok {
		return nil, fmt.Errorf("handleSelectStmt: expect *ast.SelectStmt but got %T", obj)
	}

	select_stmt, err := context.ExpandWildcard(ctx, origin_select_stmt)
	if err != nil {
		return nil, err
	}

	fn, err := NewDMLFunc(ctx, select_stmt)
	if err != nil {
		return nil, err
	}

	select_stmt_meta, err := context.NewSelectStmtMeta(ctx, select_stmt)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"Src":  origin_select_stmt.Text(),
		"Stmt": select_stmt_meta,
		"Func": fn,
	}, nil

}

func init() {
	RegistHandler((*ast.SelectStmt)(nil), handleSelectStmt)
}
