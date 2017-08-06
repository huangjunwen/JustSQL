package annot

import (
	"fmt"
	"github.com/huangjunwen/JustSQL/utils"
)

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

// Declare a bind option. Defualt to ':'.
type BindOptAnnot struct {
	// Prefix of named placeholder in query string.
	NamePrefix string
}

func (a *BindOptAnnot) SetPrimary(val string) error {
	if val != "" {
		return fmt.Errorf("bindOpt: expect no primary value but got %+q", val)
	}
	return nil
}

func (a *BindOptAnnot) Set(key, val string) error {
	switch key {
	default:
		return fmt.Errorf("bindOpt: unknown option %+q", key)
	case "namePrefix":
		if val == "" {
			return fmt.Errorf("bindOpt: got empty setting for namePrefix")
		}
		a.NamePrefix = val
	case "":
		return nil
	}
	return nil
}

func init() {
	RegistAnnot((*FuncAnnot)(nil), "func")
	RegistAnnot((*ArgAnnot)(nil), "arg", "param")
	RegistAnnot((*BindAnnot)(nil), "bind")
}
