package handler

import (
	"fmt"
	"github.com/huangjunwen/JustSQL/annot"
	"github.com/huangjunwen/JustSQL/context"
	"github.com/pingcap/tidb/ast"
	"strings"
)

type DMLFuncArg struct {
	// Name of the arg.
	Name string

	// Type of the arg
	AdaptType *context.TypeName

	// True if for "IN (?)"
	Multi bool
}

// Wrapper function for DML SQL.
type DMLFunc struct {
	ast.DMLNode

	// DMLNode.Text()
	Origin string

	// Comments (and annotations).
	Comments []annot.Comment

	// Wrapper name.
	Name string

	// Wrapper arguments.
	Args []*DMLFuncArg

	// Processed text.
	Query string

	// Return information.
	Return string
}

var no_name_cnt = 0

func NewDMLFunc(ctx *context.Context, node ast.DMLNode) (*DMLFunc, error) {

	origin := node.Text()
	ret := &DMLFunc{
		DMLNode: node,
		Origin:  origin,
		Args:    make([]*DMLFuncArg, 0),
	}

	comments, err := annot.ScanComment(origin)
	if err != nil {
		return nil, err
	}

	if len(comments) == 0 {
		no_name_cnt += 1
		ret.Name = fmt.Sprintf("NoName%d", no_name_cnt)
		ret.Query = origin
		return ret, nil
	}

	offset := 0
	parts := []string{}
	for i := 0; i < len(comments); i += 1 {
		comment := comments[i]

		// Append text before the comment.
		parts = append(parts, origin[offset:comment.Offset])

		// For different annotations.
		switch a := comment.Annot.(type) {
		case *annot.SubsAnnot:
			parts = append(parts, a.Content)

		case *annot.FuncAnnot:
			ret.Name = a.Name
			ret.Return = a.Return

		case *annot.ArgAnnot:
			ret.Args = append(ret.Args, &DMLFuncArg{
				Name:      a.Name,
				AdaptType: ctx.Scopes.CreateTypeNameFromSpec(a.Type),
				Multi:     a.Multi,
			})

		case *annot.BindAnnot:
			// Find the next comment.
			i += 1
			if i >= len(comments) {
				return nil, fmt.Errorf("bind: %q missing enclosure", a.Name)
			}
			parts = append(parts, fmt.Sprintf("%s%s", context.NAME_PLACEHOLDER, a.Name))
			comment = comments[i]

		default:
		}

		offset = comment.Offset + comment.Length
	}

	parts = append(parts, origin[offset:])

	ret.Comments = comments
	ret.Query = strings.Trim(strings.Join(parts, ""), " \t\n\r;")

	return ret, nil

}

func (fn *DMLFunc) HasMultiArg() bool {
	for _, arg := range fn.Args {
		if arg.Multi {
			return true
		}
	}
	return false
}
