package render

import (
	"github.com/huangjunwen/JustSQL/context"
	"text/template"
)

func buildFuncMap(ctx *context.Context) template.FuncMap {

	tctx := ctx.TypeContext

	// Import pkg (its path) and return a unique name.
	imp := func(pkg_path string) (string, error) {
		return tctx.CurrScope().UsePkg(pkg_path), nil
	}

	return template.FuncMap{
		"imp": imp,
	}

}
