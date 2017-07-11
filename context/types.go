package context

import (
	"fmt"
	"github.com/pingcap/tidb/mysql"
	ts "github.com/pingcap/tidb/util/types"
	"path"
	"regexp"
	"strings"
)

// TypeName represents a Go type in literal.
type TypeName struct {
	// Context in which this TypeName is created.
	typeContext *TypeContext

	// Full import path, empty if it's builtin or in current package.
	PkgPath string

	// Name of the type.
	TypeName string
}

// Return "PkgName.TypeName". NOTE: PkgName is dynamicly determined by
// current scope. For example, "github.com/go-sql-driver/mysql:NullTime" maybe
// render as "mysql.NullTime" in one scope or "mysql_1.NullTime" in another
// scope due to name conflict.
func (tn *TypeName) String() string {
	if tn.PkgPath == "" {
		return tn.TypeName
	}
	pkg_name := tn.typeContext.currScope.UsePkg(tn.PkgPath)
	return fmt.Sprintf("%s.%s", pkg_name, tn.TypeName)
}

// Mainly used to resolve package names conflict in (file) scope.
type TypeScope struct {
	// Scope name.
	scopeName string

	// Pkg path -> pkg name.
	// If pkg name == "", means the package is not used yet (maybe it is
	// import-only package, like github.com/go-sql-driver/mysql)
	pkgPaths map[string]string

	// Pkg name -> pkg path. NOTE: len(pkgNames) <= len(pkgPaths)
	pkgNames map[string]string
}

func NewTypeScope(name string) *TypeScope {
	return &TypeScope{
		scopeName: name,
		pkgPaths:  make(map[string]string),
		pkgNames:  make(map[string]string),
	}
}

// Import a package into the (file) scope.
func (scope *TypeScope) ImportPkg(pkg_path string) {
	// Do nothing for builtin or current package.
	if pkg_path == "" {
		return
	}

	// Add to pkgPaths.
	if _, ok := scope.pkgPaths[pkg_path]; !ok {
		scope.pkgPaths[pkg_path] = ""
	}

}

var ident_re *regexp.Regexp = regexp.MustCompile(`^([A-Za-z][A-Za-z0-9_]*)`)

// Use package and return a unique pakcage name in the (file) scope.
func (scope *TypeScope) UsePkg(pkg_path string) string {
	// Builtin or current package.
	if pkg_path == "" {
		return ""
	}

	// pkg_name already determined.
	pkg_name, ok := scope.pkgPaths[pkg_path]
	if pkg_name != "" {
		return pkg_name
	}

	// Import if not yet.
	if !ok {
		scope.ImportPkg(pkg_path)
	}

	// Then determin package name.
	base := strings.ToLower(ident_re.FindString(path.Base(pkg_path)))
	if base == "" {
		base = "pkg"
	}

	// Resolve name conflict.
	pkg_name = base
	i := 0
	for {
		if _, ok := scope.pkgNames[pkg_name]; !ok {
			scope.pkgPaths[pkg_path] = pkg_name
			scope.pkgNames[pkg_name] = pkg_path
			return pkg_name
		}
		// Name conflict. Add a number suffix.
		i += 1
		pkg_name = fmt.Sprintf("%s_%d", base, i)
	}
}

// List (pkg_path, pkg_name) in the (file) scope. pkg_name will be "_"
// if the package is import-only.
func (scope *TypeScope) ListPkg() [][]string {
	ret := make([][]string, 0, len(scope.pkgPaths))
	for pkg_path, pkg_name := range scope.pkgPaths {
		if pkg_name == "" {
			pkg_name = "_"
		}
		ret = append(ret, []string{
			pkg_path,
			pkg_name,
		})
	}
	return ret
}

// Type associated information.
type TypeContext struct {
	// ts.TypeToStr(*ts.FieldType.Tp) -> TypeName
	overridedAdaptTypes map[string]*TypeName

	currScope *TypeScope
	scopes    map[string]*TypeScope
}

func NewTypeContext() *TypeContext {
	ret := &TypeContext{
		overridedAdaptTypes: make(map[string]*TypeName),
		currScope:           nil,
		scopes:              make(map[string]*TypeScope),
	}
	// Switch to default scope.
	ret.SwitchScope("")
	return ret
}

// Return current (file) scope.
func (tctx *TypeContext) CurrScope() *TypeScope {
	return tctx.currScope
}

// Switch to a (file) scope.
func (tctx *TypeContext) SwitchScope(scope_name string) {
	if scope, ok := tctx.scopes[scope_name]; ok {
		tctx.currScope = scope
		return
	}

	curr := NewTypeScope(scope_name)
	tctx.scopes[scope_name] = curr
	tctx.currScope = curr
}

// Create TypeName from its package path and type name.
// Example:
//   tctx.CreateTypeName("sql", "NullString")
func (tctx *TypeContext) CreateTypeName(pkg_path, type_name string) (*TypeName, error) {
	return &TypeName{
		typeContext: tctx,
		PkgPath:     pkg_path,
		TypeName:    type_name,
	}, nil
}

// Create TypeName from colon-seperated spec:
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

// Main method of TypeContext. Find a type suitable to store a db field data.
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
					return tctx.CreateTypeName("database/sql", "NullBool")
				} else {
					return tctx.CreateTypeName("", "bool")
				}
			}

			if nullable {
				return tctx.CreateTypeName("database/sql", "NullInt64")
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
					return tctx.CreateTypeName("database/sql", "NullBool")
				} else {
					return tctx.CreateTypeName("", "bool")
				}
			}

			if nullable {
				return tctx.CreateTypeName("database/sql", "NullInt64")
			}

			if unsigned {
				return tctx.CreateTypeName("", "uint8")
			} else {
				return tctx.CreateTypeName("", "int8")
			}

		case mysql.TypeShort: // smallint
			if nullable {
				return tctx.CreateTypeName("database/sql", "NullInt64")
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
				return tctx.CreateTypeName("database/sql", "NullInt64")
			}

			if unsigned {
				return tctx.CreateTypeName("", "uint32")
			} else {
				return tctx.CreateTypeName("", "int32")
			}

		case mysql.TypeLonglong: // bigint
			if nullable {
				return tctx.CreateTypeName("database/sql", "NullInt64")
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
			return tctx.CreateTypeName("database/sql", "NullFloat64")
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
			return tctx.CreateTypeName("database/sql", "NullFloat64")
		}
		return tctx.CreateTypeName("", "float64")

	case ts.ClassString:
		if binary {
			return tctx.CreateTypeName("", "[]byte")
		}
		if nullable {
			return tctx.CreateTypeName("database/sql", "NullString")
		}
		return tctx.CreateTypeName("", "string")

	}

	return nil, fmt.Errorf("Unknown type %x", ft.Tp)
}
