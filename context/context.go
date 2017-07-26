package context

import (
	"fmt"
	"github.com/pingcap/tidb/model"
)

const (
	DEFAULT_DB_NAME  = "justsql"
	PLACEHOLDER      = "?"
	NAME_PLACEHOLDER = ":"
)

// Runtime context.
type Context struct {
	// The embeded db.
	DB *EmbedDB

	// Database name in embeded db. Currently JustSQL only support single database.
	DBName string

	// File scopes.
	*Scopes

	// DB types and go types adpater.
	*TypeAdapter

	// Extracted meta data.
	dbMeta *DBMeta
}

func NewContext(db_store_path, db_name string) (*Context, error) {
	db, err := NewEmbedDB(db_store_path)
	if err != nil {
		return nil, err
	}

	if db_name == "" {
		db_name = DEFAULT_DB_NAME
	}
	db.MustExecute(fmt.Sprintf("DROP DATABASE IF EXISTS %s", db_name))
	db.MustExecute(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", db_name))
	db.MustExecute(fmt.Sprintf("USE %s", db_name))

	scopes := NewScopes()
	type_adapter := NewTypeAdapter(scopes)

	return &Context{
		DB:          db,
		DBName:      db_name,
		Scopes:      scopes,
		TypeAdapter: type_adapter,
	}, nil

}

func (ctx *Context) DBMeta() (*DBMeta, error) {
	if ctx.dbMeta == nil {
		if err := ctx.ExtractDBMeta(); err != nil {
			return nil, err
		}
	}
	return ctx.dbMeta, nil
}

// Extract database meta information into context.
func (ctx *Context) ExtractDBMeta() error {
	is := ctx.DB.Domain().InfoSchema()
	db_info, ok := is.SchemaByName(model.NewCIStr(ctx.DBName))
	if !ok {
		return fmt.Errorf("Can't get DBInfo of %q", ctx.DBName)
	}

	db_meta, err := NewDBMeta(ctx, db_info)
	if err != nil {
		return err
	}

	ctx.dbMeta = db_meta
	return nil
}

func (ctx *Context) UniqueTableName(schema, table string) string {
	if schema != "" && schema != ctx.DBName {
		return schema + "." + table
	}
	return table
}
