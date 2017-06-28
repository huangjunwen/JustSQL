package main

import (
	"flag"
	"github.com/huangjunwen/JustSQL/embed_db"
	"github.com/pingcap/tidb/ast"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

var (
	ddl_dir = flag.String("ddl", "", "Directory containing DDL SQL files (.sql) to pre-load")
	dml_dir = flag.String("dml", "", "Directory containing DML SQL files (.sql)")
	out_dir = flag.String("o", "", "Output directory")
)

func checkDir(path string) bool {
	fi, err := os.Stat(path)
	if err == nil {
		return fi.IsDir()
	}
	return false
}

func main() {
	log.SetFlags(0)

	// Parse cmd args.
	flag.Parse()

	if *ddl_dir != "" {
		if !checkDir(*ddl_dir) {
			log.Fatalf("-ddl: %q do not exist or it's not a directory", *ddl_dir)
		}
	}

	if *dml_dir != "" {
		if !checkDir(*dml_dir) {
			log.Fatalf("-dml: %q do not exist or it's not a directory", *dml_dir)
		}
	}

	if *out_dir == "" {
		log.Fatalf("-o: No output directory specified")
	} else {
		if err := os.MkdirAll(*out_dir, os.ModePerm); err != nil {
			log.Fatalf("-o: %s", err)
		}
	}

	// Create embed db.
	db, err := embed_db.NewEmbedDB("")
	if err != nil {
		log.Fatalf("NewEmbedDB(): %s", err)
	}

	db.MustExecute("CREATE DATABASE IF NOT EXISTS justsql;")
	db.MustExecute("USE justsql;")

	// Load DDL.
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
