package handler

import (
	"fmt"
	"github.com/huangjunwen/JustSQL/context"
	"github.com/pingcap/tidb/ast"
)

func handleSelectStmt(ctx *context.Context, obj interface{}) (interface{}, error) {

	select_stmt, ok := obj.(*ast.SelectStmt)
	if !ok {
		return nil, fmt.Errorf("handleSelectStmt: expect *ast.SelectStmt but got %T", obj)
	}

	if _, err := ctx.DB.Compile(select_stmt); err != nil {
		return nil, err
	}

	fn, err := NewDMLFunc(ctx, select_stmt)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"Stmt": select_stmt,
		"Func": fn,
	}, nil

}

func init() {
	RegistHandler((*ast.SelectStmt)(nil), handleSelectStmt)
}
