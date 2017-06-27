package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/huangjunwen/JustSQL/embed_db"
	"github.com/ngaut/log"
	"github.com/pingcap/tidb/ast"
	"io/ioutil"
	"os"
	"path/filepath"
)

var importFilesPattern = flag.String("import", "", "")

func printResultFields(rfs []*ast.ResultField) {
	for _, rf := range rfs {
		c := rf.Column
		cn := rf.ColumnAsName
		//t := rf.Table
		//tn := rf.TableAsName
		fmt.Printf("col: id=%v name=%v offset=%v exp=%v type=%v col_as_name=%v;\n",
			c.ID, c.Name.O, c.Offset, c.GeneratedExprString, c.FieldType.String(), cn.O)
	}
}
func main() {
	log.SetLevelByString("error")

	flag.Parse()

	db, err := embed_db.NewEmbedDB("")
	if err != nil {
		panic(err)
	}

	if _, err := db.Execute("USE test"); err != nil {
		panic(err)
	}

	if *importFilesPattern != "" {
		importFiles, err := filepath.Glob(*importFilesPattern)
		if err != nil {
			panic(err)
		}

		for _, filename := range importFiles {
			fmt.Printf("Loading %q ...\n", filename)
			content, err := ioutil.ReadFile(filename)
			if err != nil {
				panic(err)
			}
			if _, err := db.Execute(string(content)); err != nil {
				panic(err)
			}
			fmt.Println("Done.")
		}
	}

	reader := bufio.NewReader(os.Stdin)
	sql, _ := reader.ReadString('\n')

	stmt, err := db.CompileOne(sql)
	if err != nil {
		panic(err)
	}

	select_stmt, ok := stmt.(*ast.SelectStmt)
	if !ok {
		panic(fmt.Errorf("Not a select"))
	}

	printResultFields(select_stmt.GetResultFields())
	return

}
