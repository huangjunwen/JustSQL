package types

import (
	"fmt"
	"github.com/huangjunwen/JustSQL/utils"
	//"github.com/pingcap/tidb/mysql"
	//ts "github.com/pingcap/tidb/util/types"
	"path"
	"strings"
)

// TypeName represents a Go type in literal.
type TypeName struct {
	// Full import path, empty if it's builtin
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
}

func NewTypeEnv() *TypeEnv {
	return &TypeEnv{
		pkgPath2Name: make(map[string]string),
		name2PkgPath: make(map[string]string),
	}
}

// Get a unique package name from its path.
func (env *TypeEnv) PkgName(pkg_path string) (string, error) {
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

// Format:
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

	if type_name == "" {
		return nil, fmt.Errorf("Missing type name after ':'")
	}

	pkg_name, err := env.PkgName(pkg_path)
	if err != nil {
		return nil, err
	}

	return &TypeName{
		PkgPath:  pkg_path,
		PkgName:  pkg_name,
		TypeName: type_name,
	}, nil

}
