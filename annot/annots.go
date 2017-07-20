package annot

import (
	"fmt"
	"github.com/huangjunwen/JustSQL/utils"
)

// --- FuncAnnot {{{

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

// --- }}}

// --- ArgAnnot {{{

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

// --- }}}

// --- BindAnnot {{{

type BindAnnot struct {
	// Name of the argument.
	Name string

	// Used for "IN (?)" binding.
	Multiple bool
}

func (a *BindAnnot) SetPrimary(val string) error {
	if val == "" {
		return fmt.Errorf("bind: missing arg name")
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
	case "multiple":
		a.Multiple = true
	case "":
		return nil
	}
	return nil
}

// --- }}}

// --- IfAnnot {{{

type IfAnnot struct {
	// If condition
	Cond string
}

func (a *IfAnnot) SetPrimary(val string) error {
	if val == "" {
		return fmt.Errorf("if: missing condition")
	}
	a.Cond = val
	return nil
}

func (a *IfAnnot) Set(key, val string) error {
	if key == "" {
		return nil
	}
	return fmt.Errorf("if: unknown option %+q", key)
}

// --- }}}

// --- ElifAnnot {{{

type ElifAnnot struct {
	// Else if condition
	Cond string
}

func (a *ElifAnnot) SetPrimary(val string) error {
	if val == "" {
		return fmt.Errorf("elif: missing condition")
	}
	a.Cond = val
	return nil
}

func (a *ElifAnnot) Set(key, val string) error {
	if key == "" {
		return nil
	}
	return fmt.Errorf("elif: unknown option %+q", key)
}

// --- }}}

// --- ElseAnnot {{{

type ElseAnnot struct {
}

func (a *ElseAnnot) SetPrimary(val string) error {
	if val != "" {
		return fmt.Errorf("else: expect not value")
	}
	return nil
}

func (a *ElseAnnot) Set(key, val string) error {
	if key == "" {
		return nil
	}
	return fmt.Errorf("else: unknown option %+q", key)
}

// --- }}}

// --- EndAnnot {{{

type EndAnnot struct {
}

func (a *EndAnnot) SetPrimary(val string) error {
	if val != "" {
		return fmt.Errorf("end: expect not value")
	}
	return nil
}

func (a *EndAnnot) Set(key, val string) error {
	if key == "" {
		return nil
	}
	return fmt.Errorf("end: unknown option %+q", key)
}

// --- }}}

func init() {
	RegistAnnot((*FuncAnnot)(nil), "func")
	RegistAnnot((*ArgAnnot)(nil), "arg", "param")
	RegistAnnot((*BindAnnot)(nil), "bind", "b")
	RegistAnnot((*IfAnnot)(nil), "if")
	RegistAnnot((*ElifAnnot)(nil), "elif")
	RegistAnnot((*ElseAnnot)(nil), "else")
	RegistAnnot((*EndAnnot)(nil), "end")
}
