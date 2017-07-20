package annot

import (
	"fmt"
	"github.com/huangjunwen/JustSQL/utils"
)

// Substitute content directly.
type ContentAnnot struct {
	Content string
}

func (a *ContentAnnot) SetPrimary(val string) error {
	a.Content = val
	return nil
}

func (a *ContentAnnot) Set(key, val string) error {
	if key == "" {
		return nil
	}
	return fmt.Errorf("content: unknown option %+q", key)
}

// Declare a wrapper function.
type FuncAnnot struct {
	// Function name.
	Name string
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
	return fmt.Errorf("func: unknown option %+q", key)
}

// Declare a function argument (potentially used in parameter binding).
type ArgAnnot struct {
	// Name of the argument.
	Name string

	// Type of the argument.
	Type string

	// True if for "IN (?)"
	Multi bool
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
	case "multi":
		a.Multi = true
	case "":
		return nil
	}
	return nil
}

func init() {
	RegistAnnot((*FuncAnnot)(nil), "func")
	RegistAnnot((*ArgAnnot)(nil), "arg", "param")
}
