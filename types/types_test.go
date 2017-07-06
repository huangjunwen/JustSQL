package types

import (
	"fmt"
	"testing"
)

func testParseTypeName(t *testing.T, env *TypeEnv, type_string string, expect string, expect_err bool) {
	tn, err := env.ParseTypeName(type_string)
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

func TestTypeEnv(t *testing.T) {
	fmt.Println("TestParsing")
	env := NewTypeEnv()
	testParseTypeName(t, env, "[]int", "[]int", false)
	testParseTypeName(t, env, "sql:NullString", "sql.NullString", false)
	testParseTypeName(t, env, "github.com/go-sql-driver/mysql:NullTime", "mysql.NullTime", false)
	testParseTypeName(t, env, "github.com/pingcap/tidb/mysql:SQLError", "mysql_1.SQLError", false)
	testParseTypeName(t, env, "github.com/pingcap/tidb/mysql.invalid:SQLError", "", true)

}
