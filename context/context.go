package context

import (
	"fmt"
	"github.com/pingcap/tidb/model"
)

const (
	DEFAULT_DB = "justsql"
)

// Runtime context.
type Context struct {
	// The embeded db.
	DB *EmbedDB

	// Default db name.
	DefaultDB string

	// Type things.
	*TypeContext

	// Extracted meta data.
	*DBMeta
}

func NewContext(db_store_path, default_db string) (*Context, error) {
	db, err := NewEmbedDB(db_store_path)
	if err != nil {
		return nil, err
	}

	if default_db == "" {
		default_db = DEFAULT_DB
	}
	db.MustExecute(fmt.Sprintf("DROP DATABASE IF EXISTS %s", default_db))
	db.MustExecute(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", default_db))
	db.MustExecute(fmt.Sprintf("USE %s", default_db))

	return &Context{
		DB:          db,
		DefaultDB:   default_db,
		TypeContext: NewTypeContext(),
	}, nil

}

// Extract database meta information into context.
func (ctx *Context) ExtractDBMeta() error {
	is := ctx.DB.Domain().InfoSchema()
	db_info, ok := is.SchemaByName(model.NewCIStr(ctx.DefaultDB))
	if !ok {
		return fmt.Errorf("Can't get DBInfo of %q", ctx.DefaultDB)
	}

	db_meta, err := NewDBMeta(ctx, db_info)
	if err != nil {
		return err
	}
	ctx.DBMeta = db_meta

	return nil
}
