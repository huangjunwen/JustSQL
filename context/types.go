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
	// Full import path, empty if it's builtin or in current module
	PkgPath string
	// Name for the package and type.
	PkgName  string
	TypeName string
}

// Return "PkgName.TypeName"
func (n *TypeName) String() string {
	if n.PkgName == "" {
		return n.TypeName
	}
	return fmt.Sprintf("%s.%s", n.PkgName, n.TypeName)
}

// Type associated information.
type TypeContext struct {
	// Map pkg path <-> pkg name
	pkgPath2Name map[string]string
	name2PkgPath map[string]string

	// ts.TypeToStr(*ts.FieldType.Tp) -> TypeName
	overrideAdaptTypes map[string]*TypeName
}

func NewTypeContext() *TypeContext {
	return &TypeContext{
		pkgPath2Name:       make(map[string]string),
		name2PkgPath:       make(map[string]string),
		overrideAdaptTypes: make(map[string]*TypeName),
	}
}

// Add and get a unique package name from its path.
func (tctx *TypeContext) AddPkg(pkg_path string) (string, error) {
	// Check exists.
	if name, ok := tctx.pkgPath2Name[pkg_path]; ok {
		return name, nil
	}
	// Special case for builtin.
	if pkg_path == "" {
		tctx.pkgPath2Name[""] = ""
		tctx.name2PkgPath[""] = ""
		return "", nil
	}
	// Try to use the base component as package name.
	base := path.Base(pkg_path)
	if !utils.IsIdent(base) {
		return "", fmt.Errorf("Bad package path: %q", pkg_path)
	}
	if _, ok := tctx.name2PkgPath[base]; !ok {
		tctx.pkgPath2Name[pkg_path] = base
		tctx.name2PkgPath[base] = pkg_path
		return base, nil
	}
	// Name conflict. Add a number suffix to resolve it.
	for i := 1; ; i += 1 {
		n := fmt.Sprintf("%s_%d", base, i)
		if _, ok := tctx.name2PkgPath[n]; !ok {
			tctx.pkgPath2Name[pkg_path] = n
			tctx.name2PkgPath[n] = pkg_path
			return n, nil
		}
	}
}

// Create TypeName from its package path and type name.
// Example:
//   tctx.CreateTypeName("sql", "NullString")
func (tctx *TypeContext) CreateTypeName(pkg_path, type_name string) (*TypeName, error) {
	if type_name == "" {
		return nil, fmt.Errorf("Missing type name")
	}

	pkg_name, err := tctx.AddPkg(pkg_path)
	if err != nil {
		return nil, err
	}

	return &TypeName{
		PkgPath:  pkg_path,
		PkgName:  pkg_name,
		TypeName: type_name,
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

// Override the adapt type for specific database type (tp_str).
// Example:
//   tn, err := tctx.CreateTypeName("github.com/go-sql-driver/mysql", "NullTime")
//   if err != nil {
//     ...
//   }
//   tctx.OverrideAdaptType("datetime", tn)
//   tctx.OverrideAdaptType("date", tn)
//   tctx.OverrideAdaptType("timestamp", tn)
func (tctx *TypeContext) OverrideAdaptType(tp_str string, tn *TypeName) error {
	tctx.overrideAdaptTypes[strings.ToLower(tp_str)] = tn
	return nil
}

// Change datetime/date/timestamp adapt type using 'mysql.NullTime' (generated code
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

// Find a type suitable to store database field type.
func (tctx *TypeContext) AdaptType(ft *ts.FieldType) (*TypeName, error) {
	// see: github.com/pingcap/tidb/mysql/type.go and github.com/pingcap/tidb/util/types/field_type.go
	cls := ft.ToClass()
	tp := ft.Tp
	flen := ft.Flen
	flag := ft.Flag
	nullable := !mysql.HasNotNullFlag(flag)
	unsigned := mysql.HasUnsignedFlag(flag)
	binary := mysql.HasBinaryFlag(flag)

	if tn, ok := tctx.overrideAdaptTypes[strings.ToLower(ts.TypeToStr(ft.Tp, ft.Charset))]; ok {
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
