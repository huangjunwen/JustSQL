package main

import (
	"bytes"
	"fmt"
	"github.com/huangjunwen/JustSQL/context"
	"github.com/huangjunwen/JustSQL/render"
	// Remember to import builtin templates. Otherwise files will be
	// all empty.
	_ "github.com/huangjunwen/JustSQL/templates/dft"
	"github.com/ngaut/log"
	"github.com/pingcap/tidb/ast"
	"go/format"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"text/template"
)

// Call Initialize to fill these globals.
var (
	options     *Options
	packageName string
	ctx         *context.Context
	renderer    *render.Renderer
)

func Initialize() {

	var err error
	options = ParseOptions()

	// Set log options.
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	log.SetLevelByString(options.LogLevel)

	// Get package name.
	// Already checked in option parsing.
	packageName = filepath.Base(options.OutputDir)

	// Init context.
	ctx, err = context.NewContext("", "")
	if err != nil {
		log.Fatalf("NewContext(): %s", err)
	}

	// Init renderer.
	renderer, err = render.NewRenderer(ctx)
	renderer.TypeAdapter.AllNullTypes = options.AllNullTypes
	if err != nil {
		log.Fatalf("NewRenderer(): %s", err)
	}

}

func ReadFilesFromGlobs(globs []string) func() (string, []byte, bool) {

	fileNames := []string{}
	for _, glob := range globs {

		fns, err := filepath.Glob(glob)
		if err != nil {
			log.Fatalf("filepath.Glob(%+q): %s", glob, err)
		}

		if len(fns) == 0 {
			continue
		}

		// Sort file names.
		sort.Strings(fns)

		for _, fn := range fns {
			absFn, err := filepath.Abs(fn)
			if err != nil {
				log.Fatalf("filepath.Abs(%+q): %s", fn, err)
			}
			fileNames = append(fileNames, absFn)
		}

	}

	i := 0
	return func() (fileName string, fileContent []byte, ok bool) {

		for {
			if i >= len(fileNames) {
				ok = false
				return
			}

			var err error

			fileName = fileNames[i]
			fi, err := os.Stat(fileName)
			if err != nil {
				log.Fatalf("os.Stat(%+q): %s", fileName, err)
			}
			if fi.IsDir() {
				// Skip directories.
				continue
			}

			log.Infof("ioutil.ReadFile(%+q)", fileName)
			fileContent, err = ioutil.ReadFile(fileName)
			if err != nil {
				log.Fatalf("ioutil.ReadFile(%+q): %s", fileName, err)
			}

			ok = true
			i += 1
			return
		}
		return

	}

}

func LoadTemplate() {

	log.Infof("LoadTemplate(): starts...")

	globs := []string{}
	for _, templateDir := range options.CustomTemplateDir {
		globs = append(globs, filepath.Join(templateDir, "*.tmpl"))
	}

	lastTemplateSetName := ""

	iter := ReadFilesFromGlobs(globs)
	for fileName, fileContent, ok := iter(); ok; fileName, fileContent, ok = iter() {

		// Directory name as template set name.
		templateSetName := filepath.Base(filepath.Dir(fileName))

		// File name as type name.
		typeName := filepath.Base(fileName)
		typeName = typeName[:len(typeName)-5] // strip ".tmpl"

		// Load template.
		log.Infof("LoadTemplate(): file %+q", fileName)
		if err := renderer.AddTemplate(typeName, templateSetName, string(fileContent)); err != nil {
			log.Fatalf("Renderer.AddTemplate(%+q): %s", fileName, err)
		}
		lastTemplateSetName = templateSetName

	}

	if options.TemplateSetName != "" {
		renderer.Use(options.TemplateSetName)
	} else if lastTemplateSetName != "" {
		renderer.Use(lastTemplateSetName)
	}

	log.Infof("LoadTemplate(): ended.")

}

func LoadDDL() {

	db := ctx.DB
	iter := ReadFilesFromGlobs(options.DDL)
	log.Infof("LoadDDL(): starts...")

	for fileName, fileContent, ok := iter(); ok; fileName, fileContent, ok = iter() {

		// Parse content.
		fileContentString := string(fileContent)
		stmts, err := db.Parse(fileContentString)
		if err != nil {
			log.Fatalf("LoadDDL(): file %+q, parsing got error: %s", fileName, err)
		}

		for _, stmt := range stmts {

			stmtText := stmt.Text()
			log.Infof("LoadDDL(): file %+q, statement: %+q", fileName, stmtText)

			switch stmt.(type) {
			// Allow create/drop/alter table/index.
			case *ast.CreateTableStmt, *ast.AlterTableStmt, *ast.DropTableStmt, *ast.RenameTableStmt,
				*ast.CreateIndexStmt, *ast.DropIndexStmt:
			// Also allow set statement.
			case *ast.SetStmt:
			default:
				log.Fatalf("LoadDDL(): file %+q, %T is not an allowed DDL. ", fileName, stmt)
			}

			// Run it.
			if _, err := db.Execute(stmtText); err != nil {
				log.Fatalf("LoadDDL(): file: %+q, execution got error: %s", fileName, err)
			}

		}

	}

	if _, err := ctx.GetDBMeta(ctx.DBName); err != nil {
		log.Fatalf("LoadDDL(): GetDBMeta(%+q) got error: %s", ctx.DBName, err)
	}

	log.Infof("LoadDDL(): ended.")

}

var sourceHeader = template.Must(template.New("sourceHeader").Parse(`
package {{ .PackageName }}

import (
{{ range $i, $pkg := .Imports -}}
{{ $pkgPath := index $pkg 0 -}}
{{ $pkgName := index $pkg 1 -}}
	{{ $pkgName }} {{ printf "%q" $pkgPath }}
{{ end -}}
)

// This file is generated by JustSQL (https://github.com/huangjunwen/JustSQL).
// Don't modify this file. Modify the source instead.

`))

func OutputFile(fileName string, content io.Reader) {

	var buf bytes.Buffer

	// Write header.
	if err := sourceHeader.Execute(&buf, map[string]interface{}{
		"PackageName": packageName,
		"Imports":     renderer.Scopes.CurrScope().ListPkg(),
	}); err != nil {
		log.Fatalf("ExportFile(%+q): output source header error: %s", fileName, err)
	}

	// Write content.
	io.Copy(&buf, content)

	// Open file.
	f, err := os.OpenFile(filepath.Join(options.OutputDir, fileName),
		os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		log.Fatalf("os.OpenFile(%+q): %s", fileName, err)
	}
	defer f.Close()

	// Output.
	if !options.NoFormat {
		if formatted, err := format.Source(buf.Bytes()); err != nil {
			log.Fatalf("format.Source(%q): %s", fileName, err)
		} else {
			if _, err := f.Write(formatted); err != nil {
				log.Fatalf("File.Write(%q): %s", fileName, err)
			}
		}
	} else {
		if _, err := f.Write(buf.Bytes()); err != nil {
			log.Fatalf("File.Write(%q): %s", fileName, err)
		}
	}

}

func OutputTables() {

	log.Infof("OutputTables(): starts...")

	dbMeta, err := ctx.GetDBMeta(ctx.DBName)
	if err != nil {
		log.Fatalf("ctx.GetDBMeta(%+q): %s", ctx.DBName, err)
	}

	for _, tableMeta := range dbMeta.Tables {

		log.Infof("OutputTables(): table %+q", tableMeta.Name)

		scope := fmt.Sprintf("%s.tb.go", tableMeta.Name)
		renderer.Scopes.SwitchScope(scope)

		var buf bytes.Buffer
		if err := renderer.Render(tableMeta, &buf); err != nil {
			log.Fatalf("Renderer.Render(%q): %s", scope, err)
		}

		OutputFile(scope, &buf)
	}

	log.Infof("OutputTables(): ended.")

}

func LoadAndOutputDML() {

	db := ctx.DB
	iter := ReadFilesFromGlobs(options.DML)
	log.Infof("LoadAndOutputDML(): starts...")

	for fileName, fileContent, ok := iter(); ok; fileName, fileContent, ok = iter() {

		scope := fmt.Sprintf("%s.go", filepath.Base(fileName))
		renderer.Scopes.SwitchScope(scope)

		var buf bytes.Buffer

		// Parse.
		stmts, err := db.Parse(string(fileContent))
		if err != nil {
			log.Fatalf("LoadAndOutputDML(): parsing %+q error %s", fileName, err)
		}

		// Check and render stmts.
		for _, stmt := range stmts {

			stmtText := stmt.Text()
			log.Infof("LoadAndOutputDML(): file %+q, statement: %+q", fileName, stmtText)

			switch stmt.(type) {
			case *ast.SelectStmt, *ast.InsertStmt, *ast.DeleteStmt, *ast.UpdateStmt:
			default:
				log.Fatalf("LoadAndOutputDML(): file %+q, %T is not an allowed DML. ", fileName, stmt)
			}

			if err := renderer.Render(stmt, &buf); err != nil {
				log.Fatalf("Renderer.Render(%q): %s", scope, err)
			}

		}

		OutputFile(scope, &buf)

	}

	log.Infof("LoadAndOutputDML(): ended.")

}

func OutputStandalone() {

	scope := "justsql.go"
	renderer.Scopes.SwitchScope(scope)

	var buf bytes.Buffer
	if err := renderer.Render(nil, &buf); err != nil {
		log.Fatalf("Renderer.Render(%q): %s", scope, err)
	}

	OutputFile(scope, &buf)

}

func main() {
	Initialize()
	LoadTemplate()
	LoadDDL()
	OutputTables()
	LoadAndOutputDML()
	OutputStandalone()
}
