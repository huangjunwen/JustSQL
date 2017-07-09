package context

import (
	"fmt"
	"testing"
)

func testCreateTypeNameFromSpec(t *testing.T, tctx *TypeContext, type_string string, expect string, expect_err bool) {
	tn, err := tctx.CreateTypeNameFromSpec(type_string)
	fmt.Printf("%q:\n\texpect=%q\n\tresult=%q %#v\n\texpect_err=%v\n\terr=%v\n",
		type_string, expect, tn, tn, expect_err, err)

	if (err != nil && !expect_err) || (err == nil && expect_err) {
		t.Errorf("%q: err != expect_err\n", type_string)
		return
	}
	if err == nil {
		if expect != tn.String() {
			t.Errorf("%q: result != expect\n", type_string)
			return
		}
	}
}

func TestTypeContext(t *testing.T) {
	fmt.Println("TestParsing")
	tctx := NewTypeContext()
	testCreateTypeNameFromSpec(t, tctx, "[]int", "[]int", false)
	testCreateTypeNameFromSpec(t, tctx, "sql:NullString", "sql.NullString", false)
	testCreateTypeNameFromSpec(t, tctx, "github.com/go-sql-driver/mysql:NullTime", "mysql.NullTime", false)
	testCreateTypeNameFromSpec(t, tctx, "github.com/pingcap/tidb/mysql:SQLError", "mysql_1.SQLError", false)
	testCreateTypeNameFromSpec(t, tctx, "github.com/pingcap/tidb/mysql.invalid:SQLError", "", true)

}
