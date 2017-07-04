package lex

import (
	"fmt"
	"reflect"
	"testing"
)

func testScanComment(t *testing.T, src string, expect []Comment, expect_err bool) {
	res, err := ScanComment(src)
	if (err != nil && !expect_err) || (err == nil && expect_err) {
		t.Errorf("%q: expect_err=%v, err=%v\n", src, expect_err, err)
		return
	}
	fmt.Printf("%q: expect=%v result=%v\n", src, expect, res)
	if !reflect.DeepEqual(res, expect) {
		t.Errorf("%q: scan failed\n", src)
		return
	}

}

func TestSharp(t *testing.T) {
	fmt.Println("TestSharp")
	testScanComment(t, "# abc ", []Comment{
		Comment{
			Offset:  0,
			Length:  6,
			Content: "abc",
		},
	}, false)
	testScanComment(t, "# abc \nx", []Comment{
		Comment{
			Offset:  0,
			Length:  7,
			Content: "abc",
		},
	}, false)
}

func TestDash(t *testing.T) {
	fmt.Println("TestDash")
	testScanComment(t, "-- abc ", []Comment{
		Comment{
			Offset:  0,
			Length:  7,
			Content: "abc",
		},
	}, false)
	testScanComment(t, "-- abc \nx", []Comment{
		Comment{
			Offset:  0,
			Length:  8,
			Content: "abc",
		},
	}, false)
	testScanComment(t, "-", []Comment{}, false)
	testScanComment(t, "1-1", []Comment{}, false)
}

func TestBlock(t *testing.T) {
	fmt.Println("TestBlock")
	testScanComment(t, "/* abc */", []Comment{
		Comment{
			Offset:  0,
			Length:  9,
			Content: "abc",
		},
	}, false)
	testScanComment(t, "/*", nil, true)
	testScanComment(t, "/*/", nil, true)
	testScanComment(t, "/* abc", nil, true)
	testScanComment(t, "/* abc *", nil, true)
	testScanComment(t, "/* abc * */", []Comment{
		Comment{
			Offset:  0,
			Length:  11,
			Content: "abc *",
		},
	}, false)
	testScanComment(t, "/* \n\n\n */", []Comment{
		Comment{
			Offset:  0,
			Length:  9,
			Content: "",
		},
	}, false)
}

func TestMix(t *testing.T) {
	fmt.Println("TestMix")
	testScanComment(t, "/* abc *//* def */", []Comment{
		Comment{
			Offset:  0,
			Length:  9,
			Content: "abc",
		},
		Comment{
			Offset:  9,
			Length:  9,
			Content: "def",
		},
	}, false)
	testScanComment(t, "# /* abc */", []Comment{
		Comment{
			Offset:  0,
			Length:  11,
			Content: "/* abc */",
		},
	}, false)
	testScanComment(t, "# \n/* abc */", []Comment{
		Comment{
			Offset:  0,
			Length:  3,
			Content: "",
		},
		Comment{
			Offset:  3,
			Length:  9,
			Content: "abc",
		},
	}, false)
	testScanComment(t, "/* abc # def \n */", []Comment{
		Comment{
			Offset:  0,
			Length:  17,
			Content: "abc # def",
		},
	}, false)
}

func TestStringLiteral(t *testing.T) {
	fmt.Println("TestStringLiteral")
	testScanComment(t, "hello '/* alsjfdjas */'/*  */", []Comment{
		Comment{
			Offset:  23,
			Length:  6,
			Content: "",
		},
	}, false)
}
