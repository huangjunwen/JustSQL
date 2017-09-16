package dft

import (
	"github.com/huangjunwen/JustSQL/render"
)

func init() {
	render.RegistBuiltinTemplate("standalone", render.DefaultTemplateSetName, `
{{/* =========================== */}}
{{/*          imports            */}}
{{/* =========================== */}}
{{- $ctx := imp "context" -}}
{{- $fmt := imp "fmt" -}}
{{- $reflect := imp "reflect" -}}
{{- $errors := imp "errors" -}}
{{- $sql := imp "database/sql" -}}
{{- $driver := imp "database/sql/driver" -}}
{{- $sqlx := imp "github.com/jmoiron/sqlx" -}}

// Global variables.
var (
	BindType int
)

// SetBindType set the bind type for SQL.
func SetBindType(driverName string) {
	BindType = {{ $sqlx }}.BindType(driverName)
}

// DBer for *sql.Tx and *sql.DB
type DBer interface {
	ExecContext({{ $ctx }}.Context, string, ...interface{}) ({{ $sql }}.Result, error)
	QueryContext({{ $ctx }}.Context, string, ...interface{}) (*{{ $sql }}.Rows, error)
	QueryRowContext({{ $ctx }}.Context, string, ...interface{}) *{{ $sql }}.Row
}

// IsValueValid return true if value is not 'NULL'
func IsValueValid(value interface{}) bool {
	switch val := value.(type) {
	case {{ $driver }}.Valuer:
		v, err := val.Value()
		if v == nil || err != nil {
			return false
		}
	}
	return true
}

// CoerceFromInt64 convert int64 to target type: *intX, *uintX, *sql.NullInt64.
// Data maybe truncated.
func CoerceFromInt64(src int64, target interface{}) {
	switch v := target.(type) {
	case *int8, *int16, *int32, *int64, *int:
		{{ $reflect }}.ValueOf(v).Elem().SetInt(src)
	case *uint8, *uint16, *uint32, *uint64, *uint:
		{{ $reflect }}.ValueOf(v).Elem().SetUint(uint64(src))
	case *{{ $sql }}.NullInt64:
		v.Int64 = src
		v.Valid = true
	default:
		panic({{ $fmt }}.Errorf("CoerceFromInt64 not support target type %T", target))
	}
}

// CoerceFromInt64 convert target type: intX, *intX, uintX, *uintX, sql.NullInt64, *sql.NullInt64
// to int64. Data maybe truncated.
func CoerceToInt64(src interface{}) int64 {
	switch v := src.(type) {
	case int8, int16, int32, int64, int:
		return {{ $reflect }}.ValueOf(v).Int()
	case *int8, *int16, *int32, *int64, *int:
		return {{ $reflect }}.ValueOf(v).Elem().Int()
	case uint8, uint16, uint32, uint64, uint:
		return int64({{ $reflect }}.ValueOf(v).Uint())
	case *uint8, *uint16, *uint32, *uint64, *uint:
		return int64({{ $reflect }}.ValueOf(v).Elem().Uint())
	case {{ $sql }}.NullInt64:
		return v.Int64
	case *{{ $sql }}.NullInt64:
		return v.Int64
	default:
		panic({{ $fmt }}.Errorf("CoerceToInt64 not support src type %T", src))
	}
}

var (
	CoerceErr error = {{ $errors }}.New("Coerce Error")
)

// SaveCoerceFromInt64 is similar to CoerceFromInt64. And will return error 
// if data is truncated.
func SaveCoerceFromInt64(src int64, target interface{}) error {
	CoerceFromInt64(src, target)
	src2 := CoerceToInt64(target)
	if src != src2 {
		return CoerceErr
	}
	return nil
}

`)

}
