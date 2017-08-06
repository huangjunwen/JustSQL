package render

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

var identRe *regexp.Regexp = regexp.MustCompile(`^([A-Za-z][A-Za-z0-9_]*)`)

// Import a package into the (file) scope and return a unique pkg name.
func (scope *Scope) ImportPkg(pkgPath string) string {
	// Do nothing for builtin or current package.
	if pkgPath == "" {
		return ""
	}

	// Already imported.
	if pkgName, ok := scope.pkgPaths[pkgPath]; ok {
		return pkgName
	}

	// Determin pkg name.
	base := strings.ToLower(identRe.FindString(path.Base(pkgPath)))
	if base == "" {
		base = "pkg"
	}

	// Resolve name conflict.
	pkgName := base
	i := 0
	for {
		if _, ok := scope.pkgNames[pkgName]; !ok {
			break
		}
		// Name conflict. Add a number suffix.
		i += 1
		pkgName = fmt.Sprintf("%s_%d", base, i)
	}

	// Store and return.
	scope.pkgPaths[pkgPath] = pkgName
	scope.pkgNames[pkgName] = false

	return pkgName
}

// Import (if not yet) package then use it.
func (scope *Scope) UsePkg(pkgPath string) string {
	// Builtin or current package.
	if pkgPath == "" {
		return ""
	}

	// Import and mark used.
	ret := scope.ImportPkg(pkgPath)
	scope.pkgNames[ret] = true

	return ret
}

// List (pkgPath, pkgName) in the (file) scope. pkgName will be "_"
// if the package is imported but not used.
func (scope *Scope) ListPkg() [][]string {

	ret := make([][]string, 0, len(scope.pkgPaths))
	for pkgPath, pkgName := range scope.pkgPaths {
		// If pkg is not used, change it to "_"
		if !scope.pkgNames[pkgName] {
			pkgName = "_"
		}
		ret = append(ret, []string{
			pkgPath,
			pkgName,
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
func (scopes *Scopes) SwitchScope(scopeName string) *Scope {
	if scope, ok := scopes.scopes[scopeName]; ok {
		scopes.currScope = scope
		return scope
	}
	curr := NewScope(scopeName)
	scopes.scopes[scopeName] = curr
	scopes.currScope = curr
	return curr
}

func (scopes *Scopes) CreatePkgName(pkgPath string) *PkgName {
	return &PkgName{
		scopes:  scopes,
		PkgPath: pkgPath,
	}
}

func (scopes *Scopes) CreateTypeName(pkgPath, typeName string) *TypeName {
	return &TypeName{
		PkgName:  scopes.CreatePkgName(pkgPath),
		TypeName: typeName,
	}
}

// Create TypeName from dot-seperated spec:
//   [pkgPath.]type
// Example:
//   "[]byte"
//   "sql.NullString"
//   "github.com/go-sql-driver/mysql.NullTime"
func (scopes *Scopes) CreateTypeNameFromSpec(s string) *TypeName {
	var pkgPath, typeName string
	i := strings.LastIndex(s, ".")
	if i < 0 {
		pkgPath = ""
		typeName = s
	} else {
		pkgPath = s[:i]
		typeName = s[i+1:]
	}

	return scopes.CreateTypeName(pkgPath, typeName)
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
	pkgName := tn.PkgName.String()
	if pkgName == "" {
		return tn.TypeName
	}
	return fmt.Sprintf("%s.%s", pkgName, tn.TypeName)
}
