package annot

import (
	"fmt"
	"github.com/huangjunwen/JustSQL/context"
	"strings"
)

// WrapperFuncArgMeta contains wrapper function argument meta information.
type WrapperFuncArgMeta struct {
	// Name of the arg.
	Name string

	// Type of the arg.
	AdaptType *context.TypeName
}

// WrapperFuncArgMeta contains wrapper function meta information.
type WrapperFuncMeta struct {
	// Source query text.
	SrcQuery string

	// Processed query text (e.g. comments stripped).
	Query string

	// Comments (and annotations) in SrcQuery.
	Comments []Comment

	// Wrapper name.
	Name string

	// Wrapper arguments.
	Args []WrapperFuncArgMeta

	// Return style.
	ReturnStyle

	// Has "IN (?)" binding?
	HasInBinding bool

	// Prefix of named placeholder.
	BindingNamePrefix string

	// Internal fields.
	_bindingProcessed bool
}

var noNameCnt int = 0

// NewWrapperFuncMeta gather wrapper meta from source query's comments (annotations).
func NewWrapperFuncMeta(ctx *context.Context, srcQuery string) (*WrapperFuncMeta, error) {

	ret := &WrapperFuncMeta{
		SrcQuery: srcQuery,
		Comments: make([]Comment, 0),
		Args:     make([]WrapperFuncArgMeta, 0),
	}

	comments, err := ScanComment(srcQuery)
	if err != nil {
		return nil, err
	}
	ret.Comments = comments

	parts := []string{}
	offset := 0
	for i := 0; i < len(comments); i++ {
		comment := comments[i]

		// Append text before the comment.
		parts = append(parts, srcQuery[offset:comment.Offset])

		// For different annotations.
		switch a := comment.Annot.(type) {
		case *SubsAnnot:
			parts = append(parts, a.Content)

		case *FuncAnnot:
			ret.Name = a.Name
			ret.ReturnStyle = a.ReturnStyle

		case *ArgAnnot:
			ret.Args = append(ret.Args, WrapperFuncArgMeta{
				Name:      a.Name,
				AdaptType: ctx.Scopes.CreateTypeNameFromSpec(a.Type),
			})

		case *BindOptAnnot:
			if ret._bindingProcessed {
				return nil, fmt.Errorf("bindOpt: should take place before bind")
			}
			if ret.BindingNamePrefix != "" {
				return nil, fmt.Errorf("bindOpt: namePrefix already set as %+q", ret.BindingNamePrefix)
			}
			ret.BindingNamePrefix = a.NamePrefix

		case *BindAnnot:
			// Find the next comment.
			i += 1
			if i >= len(comments) {
				return nil, fmt.Errorf("bind: %q missing enclosure", a.Name)
			}
			namePrefix := ret.BindingNamePrefix
			if namePrefix == "" {
				namePrefix = ":"
			}
			parts = append(parts, fmt.Sprintf("%s%s", namePrefix, a.Name))
			if a.In {
				ret.HasInBinding = true
			}
			ret._bindingProcessed = true
			comment = comments[i]

		default:
		}

		offset = comment.Offset + comment.Length
	}

	parts = append(parts, srcQuery[offset:])
	ret.Query = strings.Trim(strings.Join(parts, ""), " \t\n\r;")

	if ret.Name == "" {
		noNameCnt += 1
		ret.Name = fmt.Sprintf("NoName%d", noNameCnt)
	}

	return ret, nil

}
