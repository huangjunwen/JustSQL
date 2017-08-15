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

`)

}
