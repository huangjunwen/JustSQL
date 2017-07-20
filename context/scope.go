package context

import (
	"fmt"
	"path"
	"regexp"
	"strings"
)

// Used to store information in a (file) scope.
type Scope struct {
	// Scope name.
	scopeName string

	// Pkg path -> pkg name.
	pkgPaths map[string]string

	// Pkg name -> pkg used.
	pkgNames map[string]bool
}

func NewScope(name string) *Scope {
	return &Scope{
		scopeName: name,
		pkgPaths:  make(map[string]string),
		pkgNames:  make(map[string]bool),
	}
}

var ident_re *regexp.Regexp = regexp.MustCompile(`^([A-Za-z][A-Za-z0-9_]*)`)

// Import a package into the (file) scope and return a unique pkg name.
func (scope *Scope) ImportPkg(pkg_path string) string {
	// Do nothing for builtin or current package.
	if pkg_path == "" {
		return ""
	}

	// Already imported.
	if pkg_name, ok := scope.pkgPaths[pkg_path]; ok {
		return pkg_name
	}

	// Determin pkg name.
	base := strings.ToLower(ident_re.FindString(path.Base(pkg_path)))
	if base == "" {
		base = "pkg"
	}

	// Resolve name conflict.
	pkg_name := base
	i := 0
	for {
		if _, ok := scope.pkgNames[pkg_name]; !ok {
			break
		}
		// Name conflict. Add a number suffix.
		i += 1
		pkg_name = fmt.Sprintf("%s_%d", base, i)
	}

	// Store and return.
	scope.pkgPaths[pkg_path] = pkg_name
	scope.pkgNames[pkg_name] = false

	return pkg_name
}

// Import (if not yet) package then use it.
func (scope *Scope) UsePkg(pkg_path string) string {
	// Builtin or current package.
	if pkg_path == "" {
		return ""
	}

	// Import and mark used.
	ret := scope.ImportPkg(pkg_path)
	scope.pkgNames[ret] = true

	return ret
}

// List (pkg_path, pkg_name) in the (file) scope. pkg_name will be "_"
// if the package is imported but not used.
func (scope *Scope) ListPkg() [][]string {

	ret := make([][]string, 0, len(scope.pkgPaths))
	for pkg_path, pkg_name := range scope.pkgPaths {
		// If pkg is not used, change it to "_"
		if !scope.pkgNames[pkg_name] {
			pkg_name = "_"
		}
		ret = append(ret, []string{
			pkg_path,
			pkg_name,
		})
	}
	return ret

}

// Set of Scope.
type Scopes struct {
	scopes    map[string]*Scope
	currScope *Scope
}

func NewScopes() *Scopes {
	ret := &Scopes{
		scopes:    make(map[string]*Scope),
		currScope: nil,
	}
	ret.SwitchScope("")
	return ret
}

// Current (file) scope.
func (scopes *Scopes) CurrScope() *Scope {
	return scopes.currScope
}

// Switch to a (file) scope.
func (scopes *Scopes) SwitchScope(scope_name string) *Scope {
	if scope, ok := scopes.scopes[scope_name]; ok {
		scopes.currScope = scope
		return scope
	}
	curr := NewScope(scope_name)
	scopes.scopes[scope_name] = curr
	scopes.currScope = curr
	return curr
}

func (scopes *Scopes) CreatePkgName(pkg_path string) *PkgName {
	return &PkgName{
		scopes:  scopes,
		PkgPath: pkg_path,
	}
}

func (scopes *Scopes) CreateTypeName(pkg_path, type_name string) *TypeName {
	return &TypeName{
		PkgName:  scopes.CreatePkgName(pkg_path),
		TypeName: type_name,
	}
}

// Create TypeName from dot-seperated spec:
//   [full_pkg_path.]type
// Example:
//   "[]byte"
//   "sql.NullString"
//   "github.com/go-sql-driver/mysql.NullTime"
func (scopes *Scopes) CreateTypeNameFromSpec(s string) *TypeName {
	var pkg_path, type_name string
	i := strings.LastIndex(s, ".")
	if i < 0 {
		pkg_path = ""
		type_name = s
	} else {
		pkg_path = s[:i]
		type_name = s[i+1:]
	}

	return scopes.CreateTypeName(pkg_path, type_name)
}

// PkgName represents a package used in source code.
type PkgName struct {
	// In which set of scopes the pkg is declared.
	scopes *Scopes

	// Full import path, empty if it's builtin or in current package.
	PkgPath string
}

// PkgName is dynamicly determined by current scope. For example,
// "github.com/go-sql-driver/mysql" maybe render as "mysql" in one scope or
// "mysql_1" in another scope due to name conflict.
func (pn *PkgName) String() string {
	if pn.PkgPath == "" {
		return ""
	}
	return pn.scopes.CurrScope().UsePkg(pn.PkgPath)
}

// TypeName represents a type used in source code.
type TypeName struct {
	// The pkg containing the type.
	*PkgName

	// Name of the type.
	TypeName string
}

// Return "PkgName.TypeName". Note that PkgName is dynamicly determined by
// current scope. See PkgName's doc.
func (tn *TypeName) String() string {
	pkg_name := tn.PkgName.String()
	if pkg_name == "" {
		return tn.TypeName
	}
	return fmt.Sprintf("%s.%s", pkg_name, tn.TypeName)
}
