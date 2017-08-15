package render

import (
	"fmt"
	"github.com/pingcap/tidb/mysql"
	ts "github.com/pingcap/tidb/util/types"
)

// For adapting database type and go type.
type TypeAdapter struct {
	*Scopes
	customAdatpers []func(*TypeAdapter, *ts.FieldType) *TypeName
}

func NewTypeAdapter(scopes *Scopes) *TypeAdapter {
	return &TypeAdapter{
		Scopes:         scopes,
		customAdatpers: make([]func(*TypeAdapter, *ts.FieldType) *TypeName, 0),
	}
}

// Add a custom type adapter.
func (ta *TypeAdapter) AddCustomAdapter(f func(*TypeAdapter, *ts.FieldType) *TypeName) {
	ta.customAdatpers = append(ta.customAdatpers, f)
}

// Main method of TypeAdapter. Find a type suitable to store a db field data.
func (ta *TypeAdapter) AdaptType(ft *ts.FieldType) *TypeName {
	// Iterate custom adapters in reverse order
	for i := len(ta.customAdatpers) - 1; i >= 0; i -= 1 {
		tn := ta.customAdatpers[i](ta, ft)
		if tn != nil {
			return tn
		}
	}

	// see: github.com/pingcap/tidb/mysql/type.go and github.com/pingcap/tidb/util/types/field_type.go
	cls := ft.ToClass()
	tp := ft.Tp
	flen := ft.Flen
	flag := ft.Flag
	nullable := !mysql.HasNotNullFlag(flag)
	unsigned := mysql.HasUnsignedFlag(flag)
	binary := mysql.HasBinaryFlag(flag)

	switch cls {
	case ts.ClassInt:
		switch tp {
		case mysql.TypeBit: // bit
			if flen == 1 {
				if nullable {
					return ta.Scopes.CreateTypeName("database/sql", "NullBool")
				} else {
					return ta.Scopes.CreateTypeName("", "bool")
				}
			}

			if nullable {
				return ta.Scopes.CreateTypeName("database/sql", "NullInt64")
			}

			if flen <= 8 {
				return ta.Scopes.CreateTypeName("", "uint8")
			} else if flen <= 16 {
				return ta.Scopes.CreateTypeName("", "uint16")
			} else if flen <= 32 {
				return ta.Scopes.CreateTypeName("", "uint32")
			} else {
				return ta.Scopes.CreateTypeName("", "uint64")
			}

		case mysql.TypeTiny: // tinyint
			// tinyint(1) also means bool
			if flen == 1 {
				if nullable {
					return ta.Scopes.CreateTypeName("database/sql", "NullBool")
				} else {
					return ta.Scopes.CreateTypeName("", "bool")
				}
			}

			if nullable {
				return ta.Scopes.CreateTypeName("database/sql", "NullInt64")
			}

			if unsigned {
				return ta.Scopes.CreateTypeName("", "uint8")
			} else {
				return ta.Scopes.CreateTypeName("", "int8")
			}

		case mysql.TypeShort: // smallint
			if nullable {
				return ta.Scopes.CreateTypeName("database/sql", "NullInt64")
			}

			if unsigned {
				return ta.Scopes.CreateTypeName("", "uint16")
			} else {
				return ta.Scopes.CreateTypeName("", "int16")
			}

		case mysql.TypeInt24: // mediumint
			fallthrough

		case mysql.TypeLong: // int
			if nullable {
				return ta.Scopes.CreateTypeName("database/sql", "NullInt64")
			}

			if unsigned {
				return ta.Scopes.CreateTypeName("", "uint32")
			} else {
				return ta.Scopes.CreateTypeName("", "int32")
			}

		case mysql.TypeLonglong: // bigint
			if nullable {
				return ta.Scopes.CreateTypeName("database/sql", "NullInt64")
			}

			if unsigned {
				return ta.Scopes.CreateTypeName("", "uint64")
			} else {
				return ta.Scopes.CreateTypeName("", "int64")
			}

		case mysql.TypeYear:
			// 16-bit is enough for yyyy
			return ta.Scopes.CreateTypeName("", "uint16")
		}

	case ts.ClassReal:
		if nullable {
			return ta.Scopes.CreateTypeName("database/sql", "NullFloat64")
		}

		switch tp {
		case mysql.TypeFloat: // float
			return ta.Scopes.CreateTypeName("", "float32")
		case mysql.TypeDouble: // double
			return ta.Scopes.CreateTypeName("", "float64")
		}

	// NOTE: it is STRONGLY recommended to use precise type to store decimal.
	case ts.ClassDecimal:
		if nullable {
			return ta.Scopes.CreateTypeName("database/sql", "NullFloat64")
		}
		return ta.Scopes.CreateTypeName("", "float64")

	case ts.ClassString:
		switch tp {
		case mysql.TypeDatetime, mysql.TypeDate, mysql.TypeTimestamp:
			// Since we are using mysql
			if nullable {
				return ta.Scopes.CreateTypeName("github.com/go-sql-driver/mysql", "NullTime")
			}
			return ta.Scopes.CreateTypeName("time", "Time")

		default:
			if binary {
				return ta.Scopes.CreateTypeName("", "[]byte")
			}
			if nullable {
				return ta.Scopes.CreateTypeName("database/sql", "NullString")
			}
			return ta.Scopes.CreateTypeName("", "string")
		}

	}

	// Should never be here
	panic(fmt.Errorf("AdaptType failed"))
}

func (ta *TypeAdapter) CastType(srcExpr string, srcTypeName, dstTypeName *TypeName) (string, error) {

	srcSpec := srcTypeName.Spec()
	dstSpec := dstTypeName.Spec()

	if srcSpec == dstSpec {
		return srcExpr, nil
	}

	switch srcSpec {
	case "int", "uint", "int8", "uint8", "int16", "uint16", "int32", "uint32", "int64", "uint64":
		switch dstSpec {
		case "database/sql.NullInt64":
			return fmt.Sprintf("%s.NullInt64{Int64: int64(%s), Valid: true}", dstTypeName.PkgName.String(), srcExpr), nil
		}
	case "database/sql.NullInt64":
		switch dstSpec {
		case "int", "uint", "int8", "uint8", "int16", "uint16", "int32", "uint32", "int64", "uint64":
			return fmt.Sprintf("%s(%s.Int64)", dstSpec, srcExpr), nil
		}
	case "string":
		switch dstSpec {
		case "database/sql.NullString":
			return fmt.Sprintf("%s.NullString{String: %s, Valid: true}", dstTypeName.PkgName.String(), srcExpr), nil
		}
	case "database/sql.NullString":
		switch dstSpec {
		case "string":
			return fmt.Sprintf("%s.String", srcExpr), nil
		}
	case "float32", "float64":
		switch dstSpec {
		case "database/sql.NullFloat64":
			return fmt.Sprintf("%s.NullFloat64{Float64 : float64(%s), Valid: true}", dstTypeName.PkgName.String(), srcExpr), nil
		}
	case "database/sql.NullFloat64":
		switch dstSpec {
		case "float32", "float64":
			return fmt.Sprintf("%s(%s.Float64)", dstSpec, srcExpr), nil
		}
	case "bool":
		switch dstSpec {
		case "database/sql.NullBool":
			return fmt.Sprintf("%s.NullBool{Bool: %s, Valid: true}", dstTypeName.PkgName.String(), srcExpr), nil
		}
	case "database/sql.NullBool":
		switch dstSpec {
		case "bool":
			return fmt.Sprintf("%s.Bool)", srcExpr), nil
		}
	case "time.Time":
		switch dstSpec {
		case "github.com/go-sql-driver/mysql.NullTime":
			return fmt.Sprintf("%s.NullTime{Time: %s, Valid: true}", dstTypeName.PkgName.String(), srcExpr), nil
		}
	case "github.com/go-sql-driver/mysql.NullTime":
		switch dstSpec {
		case "time.Time":
			return fmt.Sprintf("%s.Time)", srcExpr), nil
		}
	case "[]byte":

	}

	return "", fmt.Errorf("Don't know how to cast %q to %q", srcSpec, dstSpec)

}
