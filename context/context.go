package context

// Runtime context.
type Context struct {
	DB                *EmbedDB
	DefaultSchemaName string
	*TypeContext
}

func NewContext(db_store_path string) (*Context, error) {
	db, err := NewEmbedDB(db_store_path)
	if err != nil {
		return nil, err
	}
	return &Context{
		DB:          db,
		TypeContext: NewTypeContext(),
	}, nil
}
