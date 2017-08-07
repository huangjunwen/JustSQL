package annot

import (
	"fmt"
	"github.com/huangjunwen/JustSQL/utils"
	"strings"
)

// Global settings.
var (
	BindNamePrefix string = ":"
)

// Declare a global setting annotation.
type SettingAnnot struct{}

func (a *SettingAnnot) SetPrimary(val string) error {
	if val != "" {
		return fmt.Errorf("setting: expect no primary value but got %+q", val)
	}
	return nil
}

func (a *SettingAnnot) Set(key, val string) error {
	switch key {
	default:
		return fmt.Errorf("setting: unknown option %+q", key)
	case "bindNamePrefix":
		if val == "" {
			return fmt.Errorf("setting: got empty setting for bindNamePrefix")
		}
		BindNamePrefix = val
	case "":
		return nil
	}
	return nil
}

// SubsAnnot declares a block of content for substitution directly.
type SubsAnnot struct {
	Content string
}

func (a *SubsAnnot) SetPrimary(val string) error {
	a.Content = val
	return nil
}

func (a *SubsAnnot) Set(key, val string) error {
	if key == "" {
		return nil
	}
	return fmt.Errorf("content: unknown option %+q", key)
}

type ReturnStyle string

const (
	ReturnUnknown = ReturnStyle("")
	// The following are used by SELECT
	ReturnMany = ReturnStyle("many")
	ReturnOne  = ReturnStyle("one")
)

// FuncAnnot declares a wrapper function for a SQL.
type FuncAnnot struct {
	// Function name.
	Name string

	// Return style.
	ReturnStyle
}

func (a *FuncAnnot) SetPrimary(val string) error {
	if val == "" {
		return fmt.Errorf("func: missing func name")
	}
	if !utils.IsIdent(val) {
		return fmt.Errorf("func: func name %+q is not a valid identifier", val)
	}
	a.Name = val
	return nil
}

func (a *FuncAnnot) Set(key, val string) error {
	if key == "" {
		return nil
	}
	if key == "return" {
		switch val {
		case "many":
			a.ReturnStyle = ReturnMany
		case "one":
			a.ReturnStyle = ReturnOne
		default:
			return fmt.Errorf("func: unknwon return type %+q", val)
		}
		return nil
	}
	return fmt.Errorf("func: unknown option %+q", key)
}

// ArgAnnot declares a function argument (maybe used in parameter binding).
type ArgAnnot struct {
	// Name of the argument.
	Name string

	// Type of the argument.
	Type string
}

func (a *ArgAnnot) SetPrimary(val string) error {
	if val == "" {
		return fmt.Errorf("arg: missing arg name")
	}
	if !utils.IsIdent(val) {
		return fmt.Errorf("arg: arg name %+q is not a valid identifier", val)
	}
	a.Name = val
	return nil
}

func (a *ArgAnnot) Set(key, val string) error {
	switch key {
	default:
		return fmt.Errorf("arg: unknown option %+q", key)
	case "type":
		a.Type = val
	case "":
		return nil
	}
	return nil
}

// Declare a binding.
type BindAnnot struct {
	// Bind arg name.
	Name string

	// The binding is used for "IN (?, ?, ?)"
	In bool
}

func (a *BindAnnot) SetPrimary(val string) error {
	if val == "" {
		return fmt.Errorf("bind: missing bind name")
	}
	if !utils.IsIdent(val) {
		return fmt.Errorf("bind: bind name %+q is not a valid identifier", val)
	}
	a.Name = val
	return nil
}

func (a *BindAnnot) Set(key, val string) error {
	switch key {
	default:
		return fmt.Errorf("bind: unknown option %+q", key)
	case "in":
		a.In = true
	case "":
		return nil
	}
	return nil
}

// Regist Annotations.
func init() {
	RegistAnnot((*SettingAnnot)(nil), "setting")
	RegistAnnot((*FuncAnnot)(nil), "func")
	RegistAnnot((*ArgAnnot)(nil), "arg", "param")
	RegistAnnot((*BindAnnot)(nil), "bind")
}

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
			parts = append(parts, BindNamePrefix+a.Name)
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
