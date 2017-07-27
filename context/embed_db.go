package context

import (
	"fmt"
	"github.com/pingcap/tidb"
	"github.com/pingcap/tidb/ast"
	"github.com/pingcap/tidb/context"
	"github.com/pingcap/tidb/domain"
	"github.com/pingcap/tidb/kv"
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

// DB context.
func (db *EmbedDB) Ctx() context.Context {
	return db.Sess.(context.Context)
}

// Get Tidb domain (storage space).
func (db *EmbedDB) Domain() *domain.Domain {
	return sessionctx.GetDomain(db.Ctx())
}

// Parse SQLs.
func (db *EmbedDB) Parse(src string) ([]ast.StmtNode, error) {
	ret, err := tidb.Parse(db.Ctx(), src)
	if err != nil {
		return nil, fmt.Errorf("Parse(%+q): %s", src, err)
	}
	return ret, nil
}

// Compile stmt.
func (db *EmbedDB) Compile(stmt ast.StmtNode) (ast.Statement, error) {
	ret, err := tidb.Compile(db.Ctx(), stmt)
	if err != nil {
		return nil, fmt.Errorf("Compile(%+q): %s", stmt.Text(), err)
	}
	return ret, nil
}

// Execute some SQLs.
func (db *EmbedDB) Execute(src string) ([]ast.RecordSet, error) {
	ret, err := db.Sess.Execute(src)
	if err != nil {
		return nil, fmt.Errorf("Execute(%+q): %s", src, err)
	}
	return ret, nil
}

// Execute some SQLs and Panic if error.
func (db *EmbedDB) MustExecute(src string) {
	_, err := db.Execute(src)
	if err != nil {
		panic(err)
	}
}
