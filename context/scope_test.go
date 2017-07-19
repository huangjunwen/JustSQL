package context

import (
	"fmt"
	"testing"
)

func testCreateTypeNameFromSpec(t *testing.T, scopes *Scopes, type_string string, expect string) {
	tn := scopes.CreateTypeNameFromSpec(type_string)
	fmt.Printf("%q:\n\texpect=%q\n\tresult=%q %#v\n",
		type_string, expect, tn, tn)

	if expect != tn.String() {
		t.Errorf("%q: result != expect\n", type_string)
		return
	}
}

func TestScopes(t *testing.T) {
	fmt.Println("TestScopes")
	scopes := NewScopes()
	testCreateTypeNameFromSpec(t, scopes, "[]int", "[]int")
	testCreateTypeNameFromSpec(t, scopes, "sql:NullString", "sql.NullString")
	testCreateTypeNameFromSpec(t, scopes, "github.com/go-sql-driver/mysql:NullTime", "mysql.NullTime")
	testCreateTypeNameFromSpec(t, scopes, "github.com/pingcap/tidb/mysql:SQLError", "mysql_1.SQLError")
	testCreateTypeNameFromSpec(t, scopes, "github.com/pingcap/tidb/mysql.dot:SQLError", "mysql_2.SQLError")

}
