package context

import (
	"github.com/pingcap/tidb"
	"github.com/pingcap/tidb/ast"
	"github.com/pingcap/tidb/context"
	"github.com/pingcap/tidb/domain"
	"github.com/pingcap/tidb/expression"
	"github.com/pingcap/tidb/kv"
	"github.com/pingcap/tidb/parser"
	"github.com/pingcap/tidb/plan"
	"github.com/pingcap/tidb/sessionctx"
)

// Use TiDB as an embeded database to execute or parse SQLs.
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
		store_path = tidb.EngineGoLevelDBMemory
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

// Get Tidb domain (storage space).
func (db *EmbedDB) Domain() *domain.Domain {
	return sessionctx.GetDomain(db.Sess.(context.Context))
}

// Parse SQLs.
func (db *EmbedDB) Parse(src string) ([]ast.StmtNode, error) {
	ctx := db.Sess.(context.Context)
	p := parser.New()
	p.SetSQLMode(ctx.GetSessionVars().SQLMode)
	charset, collation := ctx.GetSessionVars().GetCharsetInfo()
	return p.Parse(src, charset, collation)
}

// Resolve names/types in SQL statement.
func (db *EmbedDB) Resolve(stmt ast.StmtNode) error {
	ctx := db.Sess.(context.Context)
	is := db.Domain().InfoSchema()

	// This is part of tidb.Compile. Also refer expression/typeinferer_test.go

	// Resolve column names and table names, ResultSetNode types' ResultFields
	// are generated in this step.
	if err := plan.ResolveName(stmt, is, ctx); err != nil {
		return err
	}
	// Validate the statement.
	if err := plan.Validate(stmt, false); err != nil {
		return err
	}
	// Infer expression types, like 'count(*) AS cnt'
	if err := expression.InferType(ctx.GetSessionVars().StmtCtx, stmt); err != nil {
		return err
	}
	return nil
}

// Execute some SQLs.
func (db *EmbedDB) Execute(src string) ([]ast.RecordSet, error) {
	return db.Sess.Execute(src)
}

// Execute some SQLs and Panic if error.
func (db *EmbedDB) MustExecute(src string) {
	_, err := db.Execute(src)
	if err != nil {
		panic(err)
	}
}
