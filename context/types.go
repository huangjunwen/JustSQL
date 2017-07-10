package context

import (
	"fmt"
	"github.com/huangjunwen/JustSQL/utils"
	"github.com/pingcap/tidb/mysql"
	ts "github.com/pingcap/tidb/util/types"
	"path"
	"strings"
)

// TypeName represents a Go type in literal.
type TypeName struct {
	// In which scope (file) this type is used.
	Scope string
	// Full import path, empty if it's builtin or in current package.
	PkgPath string
	// Name for the package and type.
	PkgName  string
	TypeName string
}

// Type associated information for a (file) scope.
type typeScope struct {
	// Scope name.
	scopeName string

	// Pkg path -> pkg name (can be '_')
	pkgPaths map[string]string

	// Pkg name -> pkg path. NOTE: len(pkgNames) <= len(pkgPaths) since maybe
	// some packages are imported only.
	pkgNames map[string]string
}

// Type associated information.
type TypeContext struct {
	// ts.TypeToStr(*ts.FieldType.Tp) -> TypeName
	overridedAdaptTypes map[string]*TypeName

	currScope *typeScope
	scopes    map[string]*typeScope
}

// Return "PkgName.TypeName"
func (tn *TypeName) String() string {
	if tn.PkgName == "" {
		return tn.TypeName
	}
	return fmt.Sprintf("%s.%s", tn.PkgName, tn.TypeName)
}

func NewTypeContext() *TypeContext {
	ret := &TypeContext{
		overridedAdaptTypes: make(map[string]*TypeName),
		currScope:           nil,
		scopes:              make(map[string]*typeScope),
	}
	return ret
}

func (tctx *TypeContext) checkScope() *typeScope {
	if tctx.currScope == nil {
		panic(fmt.Errorf("currScope == nil. Please EnterScope first."))
	}
	return tctx.currScope
}

// Return all (file) scope names.
func (tctx *TypeContext) ListScope() []string {
	ret := make([]string, 0, len(tctx.scopes))
	for name, _ := range tctx.scopes {
		ret = append(ret, name)
	}
	return ret
}

// Switch to a (file) scope.
func (tctx *TypeContext) SwitchScope(scope_name string) {
	if scope, ok := tctx.scopes[scope_name]; ok {
		tctx.currScope = scope
		return
	}

	curr := &typeScope{
		scopeName: scope_name,
		pkgPaths:  make(map[string]string),
		pkgNames:  make(map[string]string),
	}
	tctx.scopes[scope_name] = curr
	tctx.currScope = curr
}

// Add pakcage to current (file) scope and return a unique package
// name. If the package is imported only (not used), then return '_'.
func (tctx *TypeContext) AddPkg(pkg_path string, import_only bool) (string, error) {
	scope := tctx.checkScope()

	// Check exists.
	if name, ok := scope.pkgPaths[pkg_path]; ok {
		return name, nil
	}

	// For builtin or current package.
	if pkg_path == "" {
		return "", nil
	}

	// If import_only == true, only add to pkgPaths
	if import_only {
		scope.pkgPaths[pkg_path] = "_"
		return "_", nil
	}

	// Try to use the base component as package name.
	base := path.Base(pkg_path)
	if !utils.IsIdent(base) {
		return "", fmt.Errorf("Bad package path: %q", pkg_path)
	}

	name := base
	i := 0
	for {
		if _, ok := scope.pkgNames[name]; !ok {
			scope.pkgPaths[pkg_path] = name
			scope.pkgNames[name] = pkg_path
			return name, nil
		}
		// Name conflict. Add a number suffix.
		i += 1
		name = fmt.Sprintf("%s_%d", base, i)
	}
}

// List all packages in current (file) scope. Returns a list of [pkg_path, pkg_name].
func (tctx *TypeContext) ListPkg() [][]string {
	scope := tctx.checkScope()
	ret := make([][]string, 0, len(scope.pkgPaths))
	for pkg_path, pkg_name := range scope.pkgPaths {
		ret = append(ret, []string{
			pkg_path,
			pkg_name,
		})
	}
	return ret
}

// Create TypeName from its package path and type name in current (file) scope.
// Example:
//   tctx.CreateTypeName("sql", "NullString")
func (tctx *TypeContext) CreateTypeName(pkg_path, type_name string) (*TypeName, error) {
	if type_name == "" {
		return nil, fmt.Errorf("Missing type name")
	}

	pkg_name, err := tctx.AddPkg(pkg_path, false)
	if err != nil {
		return nil, err
	}

	return &TypeName{
		Scope:    tctx.currScope.scopeName,
		PkgPath:  pkg_path,
		PkgName:  pkg_name,
		TypeName: type_name,
	}, nil
}

// Create TypeName from colon-seperated spec in current (file) scope:
//   [full_pkg_path:]type
// Example:
//   "[]byte"
//   "sql:NullString"
//   "github.com/go-sql-driver/mysql:NullTime"
func (tctx *TypeContext) CreateTypeNameFromSpec(s string) (*TypeName, error) {
	var pkg_path, type_name string
	i := strings.LastIndex(s, ":")
	if i < 0 {
		pkg_path = ""
		type_name = s
	} else {
		pkg_path = s[:i]
		type_name = s[i+1:]
	}

	return tctx.CreateTypeName(pkg_path, type_name)
}

// Override the adapt type for specific database field type.
// Example:
//   tn, err := tctx.CreateTypeName("github.com/go-sql-driver/mysql", "NullTime")
//   if err != nil {
//     ...
//   }
//   tctx.OverrideAdaptType("datetime", tn)
//   tctx.OverrideAdaptType("date", tn)
//   tctx.OverrideAdaptType("timestamp", tn)
func (tctx *TypeContext) OverrideAdaptType(tp_str string, tn *TypeName) error {
	tctx.overridedAdaptTypes[strings.ToLower(tp_str)] = tn
	return nil
}

// Use 'mysql.NullTime' for datetime/date/timestamp field type (generated code
// depends on "github.com/go-sql-driver/mysql")
func (tctx *TypeContext) UseMySQLNullTime() error {
	tn, err := tctx.CreateTypeName("github.com/go-sql-driver/mysql", "NullTime")
	if err != nil {
		return err
	}
	tctx.OverrideAdaptType("datetime", tn)
	tctx.OverrideAdaptType("date", tn)
	tctx.OverrideAdaptType("timestamp", tn)
	return nil
}

// Main method of TypeContext. Find a type suitable to store a db field data in current (file) scope.
func (tctx *TypeContext) AdaptType(ft *ts.FieldType) (*TypeName, error) {
	// see: github.com/pingcap/tidb/mysql/type.go and github.com/pingcap/tidb/util/types/field_type.go
	cls := ft.ToClass()
	tp := ft.Tp
	flen := ft.Flen
	flag := ft.Flag
	nullable := !mysql.HasNotNullFlag(flag)
	unsigned := mysql.HasUnsignedFlag(flag)
	binary := mysql.HasBinaryFlag(flag)

	if tn, ok := tctx.overridedAdaptTypes[strings.ToLower(ts.TypeToStr(ft.Tp, ft.Charset))]; ok {
		return tn, nil
	}

	switch cls {
	case ts.ClassInt:
		switch tp {
		case mysql.TypeBit: // bit
			if flen == 1 {
				if nullable {
					return tctx.CreateTypeName("sql", "NullBool")
				} else {
					return tctx.CreateTypeName("", "bool")
				}
			}

			if nullable {
				return tctx.CreateTypeName("sql", "NullInt64")
			}

			if flen <= 8 {
				return tctx.CreateTypeName("", "uint8")
			} else if flen <= 16 {
				return tctx.CreateTypeName("", "uint16")
			} else if flen <= 32 {
				return tctx.CreateTypeName("", "uint32")
			} else {
				return tctx.CreateTypeName("", "uint64")
			}

		case mysql.TypeTiny: // tinyint
			// tinyint(1) also means bool
			if flen == 1 {
				if nullable {
					return tctx.CreateTypeName("sql", "NullBool")
				} else {
					return tctx.CreateTypeName("", "bool")
				}
			}

			if nullable {
				return tctx.CreateTypeName("sql", "NullInt64")
			}

			if unsigned {
				return tctx.CreateTypeName("", "uint8")
			} else {
				return tctx.CreateTypeName("", "int8")
			}

		case mysql.TypeShort: // smallint
			if nullable {
				return tctx.CreateTypeName("sql", "NullInt64")
			}

			if unsigned {
				return tctx.CreateTypeName("", "uint16")
			} else {
				return tctx.CreateTypeName("", "int16")
			}

		case mysql.TypeInt24: // mediumint
			fallthrough

		case mysql.TypeLong: // int
			if nullable {
				return tctx.CreateTypeName("sql", "NullInt64")
			}

			if unsigned {
				return tctx.CreateTypeName("", "uint32")
			} else {
				return tctx.CreateTypeName("", "int32")
			}

		case mysql.TypeLonglong: // bigint
			if nullable {
				return tctx.CreateTypeName("sql", "NullInt64")
			}

			if unsigned {
				return tctx.CreateTypeName("", "uint64")
			} else {
				return tctx.CreateTypeName("", "int64")
			}

		case mysql.TypeYear:
			// 16-bit is enough for yyyy
			return tctx.CreateTypeName("", "uint16")
		}

	case ts.ClassReal:
		if nullable {
			return tctx.CreateTypeName("sql", "NullFloat64")
		}
		switch tp {
		case mysql.TypeFloat: // float
			return tctx.CreateTypeName("", "float32")
		case mysql.TypeDouble: // double
			return tctx.CreateTypeName("", "float64")
		}

	// NOTE: it is STRONGLY recommended to use precise type to store decimal.
	case ts.ClassDecimal:
		if nullable {
			return tctx.CreateTypeName("sql", "NullFloat64")
		}
		return tctx.CreateTypeName("", "float64")

	case ts.ClassString:
		if binary {
			return tctx.CreateTypeName("", "[]byte")
		}
		if nullable {
			return tctx.CreateTypeName("sql", "NullString")
		}
		return tctx.CreateTypeName("", "string")

	}

	return nil, fmt.Errorf("Unknown type %x", ft.Tp)
}
