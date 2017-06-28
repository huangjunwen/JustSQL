package tmpl

import (
	"fmt"
	"github.com/huangjunwen/JustSQL/embed_db"
	"github.com/pingcap/tidb/model"
)

func init() {
	// XXX
	defaultTmpl := `
	{{ .Table.Name.O }}
	  {{ range $idx, $col := .Table.Columns }}
	  {{ $col.Name.O }}
	  {{ end }}
	`
	contextFn := func(db *embed_db.EmbedDB, obj interface{}) (interface{}, error) {
		if _, ok := obj.(*model.TableInfo); !ok {
			return nil, fmt.Errorf("Expect *model.TableInfo but got %T", obj)
		}
		return map[string]interface{}{
			"Table": obj,
		}, nil
	}

	RegistRenderInfo((*model.TableInfo)(nil), "TableInfo", defaultTmpl, contextFn)
}
