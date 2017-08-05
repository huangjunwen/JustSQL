package context

import (
	"fmt"
	"github.com/pingcap/tidb/model"
)

const (
	DefaultDBName = "justsql"
)

// Context contains global runtime information of justsql.
type Context struct {
	// The embeded db.
	DB *EmbedDB

	// Default database name in embeded db.
	DBName string

	// Database name -> cached DBMeta
	CachedDBMeta map[string]*DBMeta

	// File scopes.
	*Scopes

	// DB types and go types adapter.
	*TypeAdapter
}

// NewContext create new Context.
func NewContext(storePath, dbName string) (*Context, error) {

	db, err := NewEmbedDB(storePath)
	if err != nil {
		return nil, err
	}

	if dbName == "" {
		dbName = DefaultDBName
	}
	db.MustExecute(fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName))
	db.MustExecute(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", dbName))
	db.MustExecute(fmt.Sprintf("USE %s", dbName))

	scopes := NewScopes()
	typeAdapter := NewTypeAdapter(scopes)

	return &Context{
		DB:           db,
		DBName:       dbName,
		CachedDBMeta: make(map[string]*DBMeta),
		Scopes:       scopes,
		TypeAdapter:  typeAdapter,
	}, nil

}

// ClearCachedDBMeta clears all cached DBMeta.
func (ctx *Context) ClearCachedDBMeta() {
	ctx.CachedDBMeta = make(map[string]*DBMeta)
}

// GetDBMeta get DBMeta of the given db name or use cached one.
func (ctx *Context) GetDBMeta(dbName string) (*DBMeta, error) {

	if ret, ok := ctx.CachedDBMeta[dbName]; ok {
		return ret, nil
	}
	is := ctx.DB.Domain().InfoSchema()
	dbInfo, ok := is.SchemaByName(model.NewCIStr(dbName))
	if !ok {
		return nil, fmt.Errorf("Can't get DBInfo of %q", dbName)
	}

	dbMeta, err := NewDBMeta(ctx, dbInfo)
	if err != nil {
		return nil, err
	}

	ctx.CachedDBMeta[dbName] = dbMeta
	return dbMeta, nil

}

// UniqueTableName == github.com/pingcap/tidb/plan/resolver.go nameResolver.tableUniqueName
func (ctx *Context) UniqueTableName(dbName, tableName string) string {

	if dbName != "" && dbName != ctx.DBName {
		return dbName + "." + tableName
	}
	return tableName

}
