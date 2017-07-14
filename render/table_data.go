package render

import (
	"github.com/huangjunwen/JustSQL/context"
)

func init() {
	t := `
{{/* imports */}}
{{- $ctx := imp "context" -}}
{{- $fmt := imp "fmt" -}}
{{- $sql := imp "database/sql" -}}
{{- $driver := imp "database/sql/driver" -}}

{{/* enum and set types */}}
{{ range $i, $column := .Table.Columns }}
	{{- if $column.IsEnum }}

	{{- $enum_name := printf "%s%s" $.Table.Name $column.Name -}}
// Enum{{ printf "%v" $column.Elems}}
type {{ $enum_name }} struct {
	val   string
	valid bool   // Valid is true if Val is not NULL
}

func check{{ $enum_name }}Value(s string) bool {
	switch s {
{{- range $i, $elem := $column.Elems }}
	case {{ printf "%+q" $elem }}:
		return true
{{- end }}
	default:
		return false
	}
}

// Create {{ $enum_name }}.
func New{{ $enum_name }}(s string) {{ $enum_name }} {
	if !check{{ $enum_name }}Value(s) {
		return {{ $enum_name }}{}
	}
	return {{ $enum_name }}{
		val: s,
		valid: true,
	}
}

// As string.
func (e {{ $enum_name }}) String() string {
	return e.val
}

// NULL if not valid.
func (e {{ $enum_name }}) Valid() bool {
	return e.valid
}

// Scan implements the Scanner interface.
func (e *{{ $enum_name }}) Scan(value interface{}) error {
	if value == nil {
		e.val  = ""
		e.valid = false
		return nil
	}

	var s string
	switch value.(type) {
	case []byte:
		s = string(value.([]byte))
	case string:
		s = value.(string)
	default:
		return {{ $fmt }}.Errorf("Expect string/[]byte to scan Enum {{ printf "%s" $enum_name }} but got %T", value)
	}

	if !check{{ $enum_name }}Value(s) {
		return {{ $fmt }}.Errorf("Invalid enum value %+q for {{ printf "%s" $enum_name }}", s)
	}
	e.val = s
	e.valid = true
	return nil
}

// Value implements the driver Valuer interface.
func (e {{ $enum_name }}) Value() (driver.Value, error) {
	if !e.valid {
		return nil, nil
	}
	return e.val, nil
}

	{{- else if $column.IsSet }}

	{{/* TODO */}}

	{{- end -}}
{{ end }}

{{/* declare some common vars */}}
{{- $table_name := .Table.Name.O -}}
{{- $struct_name := .Table.Name -}}
{{- $cols := .Table.Columns -}}
{{- $auto_inc_col := .Table.AutoIncColumn -}}
{{- $primary_cols := .Table.PrimaryColumns -}}
{{- $insert_cols := .Table.InsertColumns -}}

// Table {{ $table_name }}
type {{ $struct_name }} struct {
{{ range $i, $col := $cols }}
	{{- if or $col.IsEnum $col.IsSet }}
	{{ $col.Name }} {{ $struct_name }}{{ $col.Name }}
	{{- else }}
	{{ $col.Name }} {{ $col.Type }}
	{{- end -}}
	{{- " " -}}// {{ $col.Name.O }}: {{ if $col.IsNotNULL }}NOT NULL;{{ else }}NULL;{{ end }}{{ if $col.IsAutoIncrement }} AUTO INCREMENT;{{ end }} DEFAULT {{ printf "%#v" $col.DefaultValue }};{{ if $col.IsOnUpdateNow }} ON UPDATE "CURRENT_TIMESTAMP";{{ end }}
{{- end }}
}

func (entry *{{ $struct_name }}) Insert(ctx {{ $ctx }}.Context, db *{{ $sql }}.DB) error {
	const sql = "INSERT INTO {{ $table_name }} ({{ printf "%s" (column_name_list $insert_cols) }}) VALUES ({{ printf "%s" (placeholder_list (len $insert_cols)) }})"

	{{ if not_nil $auto_inc_col }}res{{ else }}_{{ end }}, err := db.ExecContext(ctx, sql{{ range $i, $col := $insert_cols }}, entry.{{ $col.Name }}{{ end }})
	if err != nil {
		return err
	}

	{{ if not_nil $auto_inc_col -}}
	last_insert_id, err := res.LastInsertId()
	if err != nil {
		return err
	}

	entry.{{ $auto_inc_col.Name }} = {{ $auto_inc_col.Type }}(last_insert_id)
	{{ end -}}

	return nil
}

{{ if ne (len $primary_cols) 0 -}}
func {{ $struct_name }}ByPrimaryKey(ctx {{ $ctx }}.Context, db *{{ $sql }}.DB{{ range $i, $col := $primary_cols }}, {{ $col.Name.CamelCase }} {{ $col.Type }}{{ end }}) (*{{ $struct_name }}, error) {
	const sql = "SELECT {{ printf "%s" (column_name_list $cols) }} FROM {{ $table_name }} " +
		"WHERE {{ range $i, $col := $primary_cols }}{{ if ne $i 0 }}AND {{ end }}{{ $col.Name.O }}={{ placeholder }} {{ end }} LIMIT 2"

	rows, err := db.QueryContext(ctx, sql{{ range $i, $col := $primary_cols }}, {{ $col.Name.CamelCase }}{{ end }})
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ret := {{ $struct_name }}{}
	cnt := 0
	for rows.Next() {
		cnt += 1
		if cnt >= 2 {
			return nil, {{ $fmt }}.Errorf("{{ $struct_name }}ByPrimaryKey returns more than one entry.")
		}
		if err := rows.Scan({{ range $i, $col := $cols }}{{ if ne $i 0 }}, {{ end }}&ret.{{ $col.Name }}{{ end }}); err != nil {
			return nil, err
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if cnt == 0 {
		return nil, nil
	}

	return &ret, nil
}
{{ end -}}

`
	RegistDefaultTypeTemplate((*context.TableData)(nil), t)
}
