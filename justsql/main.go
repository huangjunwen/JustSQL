package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/huangjunwen/JustSQL/context"
	"github.com/huangjunwen/JustSQL/render"
	"github.com/huangjunwen/JustSQL/utils"
	"github.com/ngaut/log"
	"github.com/pingcap/tidb/ast"
	"go/format"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
)

type StringArr []string

func (a *StringArr) String() string {
	if a == nil {
		return ""
	}
	return strings.Join([]string(*a), ";")
}

func (a *StringArr) Set(val string) error {
	*a = append(*a, val)
	return nil
}

var (
	// Global options.
	log_level  string
	ddl_globs  StringArr
	dml_globs  StringArr
	output_dir string
	// Global variables.
	package_name string
	ctx          *context.Context
	ri           *render.RenderInfo
)

func parseOptionsAndInit() {

	flag.StringVar(&log_level, "l", "error", "Set level of logging: fatal/error/warn/info/debug.")
	flag.Var(&ddl_globs, "d", "Glob of DDL files (file containing DDL SQL). Multiple \"-d\" is allowed.")
	flag.Var(&dml_globs, "m", "Glob of DML files (file containing DML SQL). Multiple \"-m\" is allowed.")
	flag.StringVar(&output_dir, "o", "", "Output directory.")
	flag.Parse()

	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	switch log_level {
	case "fatal", "error", "warn", "warning", "info", "debug":
		log.SetLevelByString(log_level)
	default:
		log.Fatalf("-l: illegal log level %q", log_level)
	}

	if output_dir == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	abs_output_dir, err := filepath.Abs(output_dir)
	if err != nil {
		log.Fatalf("filepath.Abs(%q): %s", output_dir, err)
	}

	fi, err := os.Stat(abs_output_dir)
	if err != nil {
		log.Fatalf("os.Stat(%q): %s", abs_output_dir, err)
	}

	if !fi.IsDir() {
		log.Fatalf("FileInfo.IsDir(%q): false", abs_output_dir)
	}

	base := filepath.Base(abs_output_dir)
	if !utils.IsIdent(base) {
		log.Fatalf("filepath.Base(%q): is not a valid go package name", abs_output_dir)
	}

	output_dir = abs_output_dir
	log.Infof("Output directory: %q", output_dir)

	package_name = base

	// Init context.
	ctx, err = context.NewContext("", "")
	if err != nil {
		log.Fatalf("NewContext: %s", err)
	}
	ctx.TypeContext.UseMySQLNullTime()

	// Init render info.
	ri = render.NewRenderInfo(ctx)

}

var pkg_file_head_tmpl = template.Must(template.New("pkg_file_head").Parse(`
package {{ .PackageName }}

import (
{{ range $i, $pkg := .Imports -}}
{{ $pkg_path := index $pkg 0 -}}
{{ $pkg_name := index $pkg 1 -}}
	{{ printf "%s" $pkg_name }} {{ printf "%q" $pkg_path }}
{{ end -}}
)

// This file is generated by JustSQL (https://github.com/huangjunwen/JustSQL).
// Don't modify this file. Modify the source instead.

`))

func genPackageFileHead(ctx *context.Context, w io.Writer) error {
	return pkg_file_head_tmpl.Execute(w, map[string]interface{}{
		"PackageName": package_name,
		"Imports":     ctx.TypeContext.CurrScope().ListPkg(),
	})
}

func readGlobs(globs []string) func() (string, []byte, bool) {

	files := []string{}
	for _, glob := range globs {
		f, err := filepath.Glob(glob)
		if err != nil {
			log.Fatalf("filepath.Glob(%q): %s", glob, err)
		}

		if len(f) == 0 {
			continue
		}

		// Sort file names
		sort.Strings(f)

		files = append(files, f...)
	}

	i := 0
	return func() (string, []byte, bool) {
		if i >= len(files) {
			return "", nil, false
		}

		file := files[i]
		log.Infof("ioutil.ReadFile(%q) ...", file)
		content, err := ioutil.ReadFile(file)
		if err != nil {
			log.Fatalf("ioutil.ReadFile(%q): %s", file, err)
		}

		i += 1
		return file, content, true
	}

}

func loadDDL() {

	db := ctx.DB
	iter := readGlobs(ddl_globs)
	log.Infof("DDL loading ...")

	for file, content, ok := iter(); ok; file, content, ok = iter() {

		// Parse content.
		content_str := string(content)
		stmts, err := db.Parse(content_str)
		if err != nil {
			log.Fatalf("db.Parse(%q): %s", file, err)
		}

		// Check and run DDL.
		for _, stmt := range stmts {

			stmt_text := stmt.Text()
			switch stmt.(type) {
			default:
				log.Fatalf("%q: %q (%T) is not an allowed statement type", file, stmt_text, stmt)

			// Allow only create/drop/alter table/index.
			case *ast.CreateTableStmt, *ast.AlterTableStmt, *ast.DropTableStmt, *ast.RenameTableStmt,
				*ast.CreateIndexStmt, *ast.DropIndexStmt:

			// Also allow set statement.
			case *ast.SetStmt:
			}

			// Run it.
			log.Infof("db.Execute(%q) ...", stmt_text)
			if _, err := db.Execute(stmt_text); err != nil {
				log.Fatalf("db.Execute(%q): %s", stmt_text, err)
			}
		}

	}

	log.Infof("DDL loaded")

}

func exportTables() {

	db_meta, err := ctx.DBMeta()
	if err != nil {
		log.Fatalf("ctx.DBMeta: %s", err)
	}

	for _, table_meta := range db_meta.Tables {

		log.Infof("exportTable(%q) ...", table_meta.Name.O)

		scope := fmt.Sprintf("%s.ddl.go", table_meta.Name.O)
		ctx.TypeContext.SwitchScope(scope)

		var body bytes.Buffer
		if err := ri.Run(table_meta, &body); err != nil {
			log.Fatalf("render.Render(%q): %s", scope, err)
		}

		var out bytes.Buffer
		if err := genPackageFileHead(ctx, &out); err != nil {
			log.Fatalf("genPackageFileHead(%q): %s", scope, err)
		}

		io.Copy(&out, &body)

		f, err := os.OpenFile(filepath.Join(output_dir, scope), os.O_RDWR|os.O_CREATE, 0755)
		if err != nil {
			log.Fatalf("os.OpenFile(%q): %s", scope, err)
		}
		defer f.Close()

		if formatted, err := format.Source(out.Bytes()); err != nil {
			log.Fatalf("format.Source(%q): %s", scope, err)
		} else {
			if _, err := f.Write(formatted); err != nil {
				log.Fatalf("File.Write(%q): %s", scope, err)
			}
		}

	}
}

func main() {

	parseOptionsAndInit()
	loadDDL()
	exportTables()

}
