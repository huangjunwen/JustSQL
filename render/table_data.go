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
	s, ok := value.(string) 
	if !ok {
		return {{ $fmt }}.Errorf("Expect string to scan Enum {{ printf "%s" $enum_name }}")
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

{{/* main struct */}}
{{- $table_name := .Table.Name.O -}}
{{- $struct_name := .Table.Name -}}
// Table {{ $table_name }}
type {{ $struct_name }} struct {
{{ range $i, $column := .Table.Columns }}
	{{- if $column.IsEnum }}
	{{ $column.Name }} {{ $.Table.Name }}{{ $column.Name }}
	{{- else if $column.IsSet }}
	{{ $column.Name }} {{ $.Table.Name }}{{ $column.Name }}
	{{- else }}
	{{ $column.Name }} {{ $column.Type }}
	{{- end -}}
	{{- " " -}}// {{ $column.Name.O }}: {{ if $column.IsNotNULL }}NOT NULL;{{ else }}NULL;{{ end }}{{ if $column.IsAutoIncrement }} AUTO INCREMENT;{{ end }} DEFAULT {{ printf "%#v" $column.DefaultValue }};{{ if $column.IsOnUpdateNow }} ON UPDATE "CURRENT_TIMESTAMP";{{ end }}
{{- end }}
}

func (entry *{{ $struct_name }}) Insert(ctx {{ $ctx }}.Context, db *{{ $sql }}.DB) error {
	{{ $cols := columns_for_insert .Table -}}
	const sql = "INSERT INTO {{ .Table.Name.O }} ({{ printf "%s" (column_name_list $cols) }}) VALUES ({{ printf "%s" (placeholder_list (len $cols)) }})"

	res, err := db.ExecContext(ctx, sql{{ range $i, $col := $cols }}, entry.{{ $col.Name }}{{ end }})
	if err != nil {
		return err
	}

	{{ if not_nil .Table.AutoIncrementColumn -}}
	last_insert_id, err := res.LastInsertId()
	if err != nil {
		return err
	}

	entry.{{ .Table.AutoIncrementColumn.Name }} = {{ .Table.AutoIncrementColumn.Type }}(last_insert_id)
	{{ end -}}

	return nil
}

{{ if not_nil .Table.Primary -}}
func {{ $struct_name }}ByPrimaryKey(ctx {{ $ctx }}.Context, db *{{ $sql }}.DB{{ range $i, $col := .Table.PrimaryColumns }}, {{ $col.Name.CamelCase }} {{ $col.Type }}{{ end }}) (*{{ $struct_name }}, error) {
	{{ $cols := .Table.Columns -}}
	const sql = "SELECT {{ printf "%s" (column_name_list $cols) }} FROM {{ .Table.Name.O }} " +
		"WHERE {{ range $i, $col := .Table.PrimaryColumns }}{{ if ne $i 0 }}AND {{ end }}{{ $col.Name.O }}={{ placeholder }} {{ end }} LIMIT 2"

	rows, err := db.QueryContext(ctx, sql{{ range $i, $col := .Table.PrimaryColumns }}, {{ $col.Name.CamelCase }}{{ end }})
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
