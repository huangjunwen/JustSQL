package types

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

type TypeEnv struct {
	// Map full pkg path <-> unique name
	pkgPath2Name map[string]string
	name2PkgPath map[string]string

	// ts.TypeToStr(*ts.FieldType.Tp) -> TypeName
	overrideAdaptTypes map[string]*TypeName
}

func NewTypeEnv() *TypeEnv {
	return &TypeEnv{
		pkgPath2Name:       make(map[string]string),
		name2PkgPath:       make(map[string]string),
		overrideAdaptTypes: make(map[string]*TypeName),
	}
}

// Add and get a unique package name from its path.
func (env *TypeEnv) AddPkg(pkg_path string) (string, error) {
	// Check exists.
	if name, ok := env.pkgPath2Name[pkg_path]; ok {
		return name, nil
	}
	// Special case for builtin.
	if pkg_path == "" {
		env.pkgPath2Name[""] = ""
		env.name2PkgPath[""] = ""
		return "", nil
	}
	// Try to use the base component as package name.
	base := path.Base(pkg_path)
	if !utils.IsIdent(base) {
		return "", fmt.Errorf("Bad package path: %q", pkg_path)
	}
	if _, ok := env.name2PkgPath[base]; !ok {
		env.pkgPath2Name[pkg_path] = base
		env.name2PkgPath[base] = pkg_path
		return base, nil
	}
	// Name conflict. Add a number suffix to resolve it.
	for i := 1; ; i += 1 {
		n := fmt.Sprintf("%s_%d", base, i)
		if _, ok := env.name2PkgPath[n]; !ok {
			env.pkgPath2Name[pkg_path] = n
			env.name2PkgPath[n] = pkg_path
			return n, nil
		}
	}
}

// Create TypeName from its package path and type name.
// Example:
//   env.CreateTypeName("sql", "NullString")
func (env *TypeEnv) CreateTypeName(pkg_path, type_name string) (*TypeName, error) {
	if type_name == "" {
		return nil, fmt.Errorf("Missing type name after ':'")
	}

	pkg_name, err := env.AddPkg(pkg_path)
	if err != nil {
		return nil, err
	}

	return &TypeName{
		PkgPath:  pkg_path,
		PkgName:  pkg_name,
		TypeName: type_name,
	}, nil
}

// Create TypeName from colon seperated format:
//   [full_pkg_path:]type
// Example:
//   "[]byte"
//   "sql:NullString"
//   "github.com/go-sql-driver/mysql:NullTime"
func (env *TypeEnv) ParseTypeName(s string) (*TypeName, error) {
	var pkg_path, type_name string
	i := strings.LastIndex(s, ":")
	if i < 0 {
		pkg_path = ""
		type_name = s
	} else {
		pkg_path = s[:i]
		type_name = s[i+1:]
	}

	return env.CreateTypeName(pkg_path, type_name)
}

// Override the adapt type for specific database type (tp_str).
// Example:
//   tn, err := env.ParseTypeName("github.com/go-sql-driver/mysql:NullTime")
//   if err != nil {
//     ...
//   }
//   env.OverrideAdaptType("datetime", tn)
//   env.OverrideAdaptType("date", tn)
//   env.OverrideAdaptType("timestamp", tn)
func (env *TypeEnv) OverrideAdaptType(tp_str string, tn *TypeName) {
	env.overrideAdaptTypes[tp_str] = tn
}

// Find a type suitable to store database field type.
func (env *TypeEnv) AdaptFieldType(ft *ts.FieldType) (*TypeName, error) {
	// see: github.com/pingcap/tidb/mysql/type.go and github.com/pingcap/tidb/util/types/field_type.go
	cls := ft.ToClass()
	tp := ft.Tp
	flen := ft.Flen
	flag := ft.Flag
	nullable := !mysql.HasNotNullFlag(flag)
	unsigned := mysql.HasUnsignedFlag(flag)
	binary := mysql.HasBinaryFlag(flag)

	if tn, ok := env.overrideAdaptTypes[ts.TypeToStr(ft.Tp, ft.Charset)]; ok {
		return tn, nil
	}

	switch cls {
	case ts.ClassInt:
		// Bit is special. Can be up to 64-bit and bit(1) means bool
		if tp == mysql.TypeBit {
			if flen == 1 {
				if nullable {
					return env.CreateTypeName("sql", "NullBool")
				} else {
					return env.CreateTypeName("", "bool")
				}
			}

			if nullable {
				return env.CreateTypeName("sql", "NullInt64")
			}
			if flen <= 8 {
				return env.CreateTypeName("", "uint8")
			} else if flen <= 16 {
				return env.CreateTypeName("", "uint16")
			} else if flen <= 32 {
				return env.CreateTypeName("", "uint32")
			} else {
				return env.CreateTypeName("", "uint64")
			}
		}

		if nullable {
			return env.CreateTypeName("sql", "NullInt64")
		}

		switch tp {
		case mysql.TypeTiny: // tinyint
			if unsigned {
				return env.CreateTypeName("", "uint8")
			} else {
				return env.CreateTypeName("", "int8")
			}
		case mysql.TypeShort: // smallint
			if unsigned {
				return env.CreateTypeName("", "uint16")
			} else {
				return env.CreateTypeName("", "int16")
			}
		case mysql.TypeInt24: // mediumint
			fallthrough
		case mysql.TypeLong: // int
			if unsigned {
				return env.CreateTypeName("", "uint32")
			} else {
				return env.CreateTypeName("", "int32")
			}
		case mysql.TypeLonglong: // bigint
			if unsigned {
				return env.CreateTypeName("", "uint64")
			} else {
				return env.CreateTypeName("", "int64")
			}
		case mysql.TypeYear:
			// 16-bit is enough for yyyy
			return env.CreateTypeName("", "uint16")
		}

	case ts.ClassReal:
		if nullable {
			return env.CreateTypeName("sql", "NullFloat64")
		}
		switch tp {
		case mysql.TypeFloat: // float
			return env.CreateTypeName("", "float32")
		case mysql.TypeDouble: // double
			return env.CreateTypeName("", "float64")
		}

	// NOTE: it is STRONGLY recommended to use precise type to store decimal.
	case ts.ClassDecimal:
		if nullable {
			return env.CreateTypeName("sql", "NullFloat64")
		}
		return env.CreateTypeName("", "float64")

	case ts.ClassString:
		if binary {
			return env.CreateTypeName("", "[]byte")
		}
		if nullable {
			return env.CreateTypeName("sql", "NullString")
		}
		return env.CreateTypeName("", "string")

	}

	return nil, fmt.Errorf("Unknown type %x", ft.Tp)
}
