package render

import (
	"fmt"
	"github.com/huangjunwen/JustSQL/annot"
	"github.com/huangjunwen/JustSQL/context"
	"github.com/pingcap/tidb/ast"
)

func handleTableMeta(r *Renderer, obj interface{}) (interface{}, error) {

	tableMeta, ok := obj.(*context.TableMeta)
	if !ok {
		return nil, fmt.Errorf("handleTableMeta: expect *context.TableMeta but got %T", obj)
	}

	// The 'dot' object to render TableMeta
	return map[string]interface{}{
		"Table": tableMeta,
	}, nil
}

func handleSelectStmt(r *Renderer, obj interface{}) (interface{}, error) {

	ctx := r.Context

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

	annotMeta, err := annot.NewAnnotMeta(stmtMeta.SelectStmt.Text())
	if err != nil {
		return nil, err
	}
	switch annotMeta.ReturnStyle {
	case annot.ReturnMany, annot.ReturnOne:
	case annot.ReturnUnknown:
		// Default return many for select.
		annotMeta.ReturnStyle = annot.ReturnMany
	default:
		return nil, fmt.Errorf("Wrapper function's return can't be %+q for SELECT ",
			annotMeta.ReturnStyle)
	}

	return map[string]interface{}{
		"OriginStmt": originStmtMeta,
		"Stmt":       stmtMeta,
		"Annot":      annotMeta,
	}, nil

}

func handleInsertStmt(r *Renderer, obj interface{}) (interface{}, error) {

	ctx := r.Context

	stmt, ok := obj.(*ast.InsertStmt)
	if !ok {
		return nil, fmt.Errorf("handleInsertStmt: expect *ast.InsertStmt but got %T", obj)
	}

	stmtMeta, err := context.NewInsertStmtMeta(ctx, stmt)
	if err != nil {
		return nil, err
	}

	annotMeta, err := annot.NewAnnotMeta(stmt.Text())
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"Stmt":  stmtMeta,
		"Annot": annotMeta,
	}, nil

}

func handleDeleteStmt(r *Renderer, obj interface{}) (interface{}, error) {

	ctx := r.Context

	stmt, ok := obj.(*ast.DeleteStmt)
	if !ok {
		return nil, fmt.Errorf("handleDeleteStmt: expect *ast.DeleteStmt but got %T", obj)
	}

	stmtMeta, err := context.NewDeleteStmtMeta(ctx, stmt)
	if err != nil {
		return nil, err
	}

	annotMeta, err := annot.NewAnnotMeta(stmt.Text())
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"Stmt":  stmtMeta,
		"Annot": annotMeta,
	}, nil

}

func handleUpdateStmt(r *Renderer, obj interface{}) (interface{}, error) {

	ctx := r.Context

	stmt, ok := obj.(*ast.UpdateStmt)
	if !ok {
		return nil, fmt.Errorf("handleUpdateStmt: expect *ast.UpdateStmt but got %T", obj)
	}

	stmtMeta, err := context.NewUpdateStmtMeta(ctx, stmt)
	if err != nil {
		return nil, err
	}

	annotMeta, err := annot.NewAnnotMeta(stmt.Text())
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"Stmt":  stmtMeta,
		"Annot": annotMeta,
	}, nil

}

func handleStandalone(r *Renderer, obj interface{}) (interface{}, error) {
	return nil, nil
}

func init() {
	RegistType("table", (*context.TableMeta)(nil), handleTableMeta)
	RegistType("select", (*ast.SelectStmt)(nil), handleSelectStmt)
	RegistType("insert", (*ast.InsertStmt)(nil), handleInsertStmt)
	RegistType("delete", (*ast.DeleteStmt)(nil), handleDeleteStmt)
	RegistType("update", (*ast.UpdateStmt)(nil), handleUpdateStmt)
	RegistType("standalone", (interface{})(nil), handleStandalone)
}
