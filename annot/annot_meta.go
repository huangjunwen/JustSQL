package annot

import (
	"fmt"
	"strings"
)

// Global settings.
var (
	BindingNamePrefix string = ":"
)

// AnnotMeta contains meta information of annotations.
type AnnotMeta struct {
	// Source query text.
	SrcText string

	// Processed query text (e.g. comments stripped and annotations processed).
	Text string

	// Comments (and annotations) in SrcQuery.
	Comments []Comment

	// Function name (from FuncAnnot).
	FuncName string

	// Function arguments (from ArgAnnot).
	Args []*ArgAnnot

	// Return style.
	ReturnStyle

	// List of binding name.
	Bindings []string

	// "IN (?)" bindings (index of Bindings)
	InBindings []int
}

var noNameCnt int = 0

// NewAnnotMeta gather wrapper meta from source query's comments (annotations).
func NewAnnotMeta(src string) (*AnnotMeta, error) {

	ret := &AnnotMeta{
		SrcText:    src,
		Args:       make([]*ArgAnnot, 0),
		Bindings:   make([]string, 0),
		InBindings: make([]int, 0),
	}

	comments, err := ScanComment(src)
	if err != nil {
		return nil, err
	}
	ret.Comments = comments

	parts := []string{}
	offset := 0
	for i := 0; i < len(comments); i++ {
		comment := comments[i]

		// Append text before the comment.
		parts = append(parts, src[offset:comment.Offset])

		// For different annotations.
		switch a := comment.Annot.(type) {
		case *BindOptAnnot:
			BindingNamePrefix = a.NamePrefix

		case *SubsAnnot:
			parts = append(parts, a.Content)

		case *FuncAnnot:
			ret.FuncName = a.Name
			ret.ReturnStyle = a.ReturnStyle

		case *ArgAnnot:
			ret.Args = append(ret.Args, a)

		case *BindAnnot:
			// Find the next comment.
			i += 1
			if i >= len(comments) {
				return nil, fmt.Errorf("bind: %q missing enclosure", a.Name)
			}
			parts = append(parts, BindingNamePrefix+a.Name)
			if a.In {
				ret.InBindings = append(ret.InBindings, len(ret.Bindings))
			}
			ret.Bindings = append(ret.Bindings, a.Name)
			comment = comments[i]

		default:
		}

		offset = comment.Offset + comment.Length
	}

	parts = append(parts, src[offset:])
	ret.Text = strings.Trim(strings.Join(parts, ""), " \t\n\r;")

	if ret.FuncName == "" {
		noNameCnt += 1
		ret.FuncName = fmt.Sprintf("NoName%d", noNameCnt)
	}

	return ret, nil

}
