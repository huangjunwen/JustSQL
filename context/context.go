package context

import (
	"fmt"
	"github.com/pingcap/tidb/model"
)

const (
	DEFAULT_default_db_name = "justsql"
	PLACEHOLDER             = "?"
	NAME_PLACEHOLDER        = ":"
)

// Runtime context.
type Context struct {
	// The embeded db.
	DB *EmbedDB

	// Default database name in embeded db.
	DefaultDBName string

	// Default database meta.
	// NOTE: Call ExtractDefaultDBMeta before using this field.
	DefaultDBMeta *DBMeta

	// File scopes.
	*Scopes

	// DB types and go types adpater.
	*TypeAdapter

	// SQL placeholders.
	Placeholder     string
	NamePlaceholder string
}

func NewContext(store_path, default_db_name string) (*Context, error) {

	db, err := NewEmbedDB(store_path)
	if err != nil {
		return nil, err
	}

	if default_db_name == "" {
		default_db_name = DEFAULT_default_db_name
	}
	db.MustExecute(fmt.Sprintf("DROP DATABASE IF EXISTS %s", default_db_name))
	db.MustExecute(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", default_db_name))
	db.MustExecute(fmt.Sprintf("USE %s", default_db_name))

	scopes := NewScopes()
	type_adapter := NewTypeAdapter(scopes)

	return &Context{
		DB:              db,
		DefaultDBName:   default_db_name,
		Scopes:          scopes,
		TypeAdapter:     type_adapter,
		Placeholder:     PLACEHOLDER,
		NamePlaceholder: NAME_PLACEHOLDER,
	}, nil

}

// Extract default database meta into context.
func (ctx *Context) ExtractDefaultDBMeta() error {

	db_meta, err := ctx.ExtractDBMeta(ctx.DefaultDBName)
	if err != nil {
		return err
	}
	ctx.DefaultDBMeta = db_meta
	return nil

}

// Extract database meta information.
func (ctx *Context) ExtractDBMeta(db_name string) (*DBMeta, error) {

	is := ctx.DB.Domain().InfoSchema()
	db_info, ok := is.SchemaByName(model.NewCIStr(db_name))
	if !ok {
		return nil, fmt.Errorf("Can't get DBInfo of %q", db_name)
	}

	return NewDBMeta(ctx, db_info)

}

func (ctx *Context) UniqueTableName(db_name, table_name string) string {

	if db_name != "" && db_name != ctx.DefaultDBName {
		return db_name + "." + table_name
	}
	return table_name

}
