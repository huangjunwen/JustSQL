package main

import (
	"bufio"
	"fmt"
	"github.com/huangjunwen/JustSQL/embed_db"
	"github.com/ngaut/log"
	"github.com/pingcap/tidb/ast"
	"os"
)

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
	var (
		db    *embed_db.EmbedDB
		stmts []ast.StmtNode
		err   error
	)

	log.SetLevelByString("error")

	reader := bufio.NewReader(os.Stdin)
	sql, _ := reader.ReadString('\n')

	if db, err = embed_db.NewEmbedDB(""); err != nil {
		goto ERR
	}

	if stmts, err = db.Compile(sql); err != nil {
		goto ERR
	}

	for _, stmt := range stmts {
		select_stmt, ok := stmt.(*ast.SelectStmt)
		if !ok {
			continue
		}
		printResultFields(select_stmt.GetResultFields())
	}
	return

ERR:
	fmt.Println(err)
	return
}
