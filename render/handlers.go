package render

import (
	"fmt"
	"github.com/huangjunwen/JustSQL/annot"
	"github.com/huangjunwen/JustSQL/context"
	"github.com/pingcap/tidb/ast"
)

func handleTableMeta(ctx *context.Context, obj interface{}) (interface{}, error) {

	tableMeta, ok := obj.(*context.TableMeta)
	if !ok {
		return nil, fmt.Errorf("handleTableMeta: expect *context.TableMeta but got %T", obj)
	}

	// The 'dot' object to render TableMeta
	return map[string]interface{}{
		"Table": tableMeta,
	}, nil
}

func handleSelectStmt(ctx *context.Context, obj interface{}) (interface{}, error) {

	originStmt, ok := obj.(*ast.SelectStmt)
	if !ok {
		return nil, fmt.Errorf("handleSelectStmt: expect *ast.SelectStmt but got %T", obj)
	}

	originStmtMeta, err := context.NewSelectStmtMeta(ctx, originStmt)
	if err != nil {
		return nil, err
	}

	stmtMeta, err := originStmtMeta.ExpandWildcard(ctx)
	if err != nil {
		return nil, err
	}

	fnMeta, err := annot.NewWrapperFuncMeta(ctx, stmtMeta.SelectStmt.Text())
	if err != nil {
		return nil, err
	}
	switch fnMeta.Return {
	case annot.ReturnMany, annot.ReturnOne:
	case annot.ReturnUnknown:
		// Default return many for select.
		fnMeta.Return = annot.ReturnMany
	default:
		return nil, fmt.Errorf("Wrapper function's return can't be %+q for SELECT ", fnMeta.Return)
	}

	return map[string]interface{}{
		"OriginStmt": originStmtMeta,
		"Stmt":       stmtMeta,
		"Func":       fnMeta,
	}, nil

}

func init() {
	RegistType("table", (*context.TableMeta)(nil), handleTableMeta)
	RegistType("select", (*ast.SelectStmt)(nil), handleSelectStmt)
}
