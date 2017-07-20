package annot

import (
	"fmt"
	"reflect"
	"testing"
)

func testAnnot(t *testing.T, src string, expect Annot, expect_err bool) {
	annot, err := ParseAnnot(src)
	fmt.Printf("%q:\n\texpect=%#v\n\tresult=%#v\n\texpect_err=%v\n\terr=%v\n",
		src, expect, annot, expect_err, err)

	if (err != nil && !expect_err) || (err == nil && expect_err) {
		t.Errorf("%q: err != expect_err\n", src)
		return
	}
	if !reflect.DeepEqual(annot, expect) {
		t.Errorf("%q: result != expect\n", src)
		return
	}

}

func TestParsing(t *testing.T) {
	fmt.Println("TestParsing")
	testAnnot(t, "func", nil, true)
	testAnnot(t, "func:", nil, true)
	testAnnot(t, "func:*", nil, true)
	testAnnot(t, "func:\"a b\"", nil, true)
	testAnnot(t, "func:\"a\\cb\"", &FuncAnnot{
		Name: "acb",
	}, false)
}

/*
func TestBind(t *testing.T) {
	fmt.Println("TestBind")
	testAnnot(t, "bind:_a", &BindAnnot{
		Name: "_a",
	}, false)
	testAnnot(t, "bind:abc type:sql.NullString", &BindAnnot{
		Name: "abc",
		Type: "sql.NullString",
	}, false)
	testAnnot(t, "bind:abc slice type:sql.NullString", &BindAnnot{
		Name:  "abc",
		Type:  "sql.NullString",
		Slice: true,
	}, false)
}
*/
