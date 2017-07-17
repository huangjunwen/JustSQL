package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/huangjunwen/JustSQL/context"
	"github.com/huangjunwen/JustSQL/render"
	"github.com/pingcap/tidb/ast"
	"go/format"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

var (
	ddl_dir = flag.String("ddl", "", "Directory containing DDL SQL files (.sql) to pre-load")
)

func checkDir(path string) bool {
	fi, err := os.Stat(path)
	if err == nil {
		return fi.IsDir()
	}
	return false
}

func loadDDL(ctx *context.Context) {
	db := ctx.DB
	// No ddl dir.
	if *ddl_dir == "" {
		return
	}

	ddl_files, err := filepath.Glob(*ddl_dir + "/*.sql")
	if err != nil {
		log.Fatalf("filepath.Glob(%q): %s", *ddl_dir+"/*.sql", err)
	}

	// For each SQL file.
	for _, ddl_file := range ddl_files {
		content, err := ioutil.ReadFile(ddl_file)
		if err != nil {
			log.Fatalf("ioutil.ReadFile(%q): %s", ddl_file, err)
		}
		ddl := string(content)

		// Parse file content.
		stmts, err := db.Parse(ddl)
		if err != nil {
			log.Fatalf("db.Parse(<file: %q>): %s", ddl_file, err)
		}

		// Check DDL statments.
		for _, stmt := range stmts {
			if _, ok := stmt.(ast.DDLNode); ok {
				continue
			}
			if _, ok := stmt.(*ast.SetStmt); ok {
				continue
			}
			log.Fatalf("<file: %q>: %q is not a DDL statement (%T)", ddl_file, stmt.Text(), stmt)
		}

		// Execute it.
		if _, err := db.Execute(ddl); err != nil {
			log.Fatalf("db.Execute(<file: %q>): %s", ddl_file, err)
		}
	}
}

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	// Parse cmd args.
	flag.Parse()

	if *ddl_dir != "" {
		if !checkDir(*ddl_dir) {
			log.Fatalf("-ddl: %q do not exist or it's not a directory", *ddl_dir)
		}
	}

	ctx, err := context.NewContext("", "")
	if err != nil {
		log.Fatalf("NewContext: %s", err)
	}
	ctx.TypeContext.UseMySQLNullTime()

	// Load DDL.
	log.Printf("Start to load DDL\n")
	loadDDL(ctx)
	log.Printf("DDL loaded\n")

	db_meta, err := ctx.DBMeta()
	if err != nil {
		log.Fatalf("ctx.DBMeta: %s", err)
	}

	// XXX: Test name conflict
	//ctx.CurrScope().UsePkg("hello/fmt")

	var all, body bytes.Buffer
	ri := render.NewRenderInfo(ctx)

	for _, table_meta := range db_meta.Tables {
		err := ri.Run(table_meta, &body)
		if err != nil {
			log.Fatalf("render.Render: %s", err)
		}
	}

	all.Write([]byte("package main\n\n"))
	all.Write([]byte("import (\n"))
	for _, pkg := range ctx.TypeContext.CurrScope().ListPkg() {
		pkg_path, pkg_name := pkg[0], pkg[1]
		all.Write([]byte(fmt.Sprintf("\t%s %q\n", pkg_name, pkg_path)))
	}
	all.Write([]byte(")\n"))

	io.Copy(&all, &body)

	if true {
		formatted, err := format.Source(all.Bytes())
		if err != nil {
			log.Fatalf("format.Source: %s", err)
		}

		io.Copy(os.Stdout, bytes.NewBuffer(formatted))
	} else {
		io.Copy(os.Stdout, &all)
	}

}
