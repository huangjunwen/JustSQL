package embed_db

import (
	"github.com/pingcap/tidb"
	"github.com/pingcap/tidb/ast"
	"github.com/pingcap/tidb/context"
	"github.com/pingcap/tidb/expression"
	"github.com/pingcap/tidb/kv"
	"github.com/pingcap/tidb/plan"
	"github.com/pingcap/tidb/sessionctx"
)

// Use TiDB as an embeded database to execute or compile SQLs.
type EmbedDB struct {
	Store kv.Storage
	Sess  tidb.Session
}

func NewEmbedDB(store_path string) (*EmbedDB, error) {
	var (
		store kv.Storage
		sess  tidb.Session
		err   error
	)

	// Use memory store by default
	if store_path == "" {
		store_path = "memory://"
	}

	if store, err = tidb.NewStore(store_path); err != nil {
		return nil, err
	}

	if _, err = tidb.BootstrapSession(store); err != nil {
		return nil, err
	}

	if sess, err = tidb.CreateSession(store); err != nil {
		return nil, err
	}

	return &EmbedDB{
		Store: store,
		Sess:  sess,
	}, nil

}

func (db *EmbedDB) Execute(src string) ([]ast.RecordSet, error) {
	return db.Sess.Execute(src)
}

func (db *EmbedDB) Compile(src string) ([]ast.StmtNode, error) {
	var (
		stmts []ast.StmtNode
		err   error
	)

	ctx := db.Sess.(context.Context)
	is := sessionctx.GetDomain(ctx).InfoSchema()

	// Parse SQL statements.
	if stmts, err = tidb.Parse(ctx, src); err != nil {
		return nil, err
	}

	// The loop is part of the process of tidb.Compile. Also refer expression/typeinferer_test.go
	for _, stmt := range stmts {
		// Resolve column names and table names, ResultSetNode types' ResultFields
		// are generated in this step.
		if err = plan.ResolveName(stmt, is, ctx); err != nil {
			return nil, err
		}

		// Validate the statement.
		if err = plan.Validate(stmt, false); err != nil {
			return nil, err
		}

		// Infer expression types, like 'count(*) AS cnt'
		if err = expression.InferType(ctx.GetSessionVars().StmtCtx, stmt); err != nil {
			return nil, err
		}

	}

	return stmts, nil

}
