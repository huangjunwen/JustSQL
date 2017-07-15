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
{{- $non_primary_cols := .Table.NonPrimaryColumns -}}

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

func (entry_ *{{ $struct_name }}) Insert(ctx_ {{ $ctx }}.Context, tx_ *{{ $sql }}.Tx) error {
	const sql_ = "INSERT INTO {{ $table_name }} ({{ printf "%s" (column_name_list $cols) }}) VALUES ({{ printf "%s" (placeholder_list (len $cols)) }})"

	{{ if not_nil $auto_inc_col }}res_{{ else }}_{{ end }}, err_ := tx_.ExecContext(ctx_, sql_{{ range $i, $col := $cols }}, entry_.{{ $col.Name }}{{ end }})
	if err_ != nil {
		return err_
	}

	{{ if not_nil $auto_inc_col -}}
	last_insert_id_, err_ := res_.LastInsertId()
	if err_ != nil {
		return err_
	}

	entry_.{{ $auto_inc_col.Name }} = {{ $auto_inc_col.Type }}(last_insert_id_)
	{{ end -}}

	return nil
}

{{ if ne (len $primary_cols) 0 -}}

{{ if ne (len $non_primary_cols) 0 -}}
func (entry_ *{{ $struct_name }}) Update(ctx_ {{ $ctx }}.Context, tx_ *{{ $sql }}.Tx) error {
	const sql_ = "UPDATE {{ $table_name }} SET {{ range $i, $col := $non_primary_cols }}{{ if ne $i 0 }}, {{ end }}{{ $col.Name.O }}={{ placeholder }}{{ end }}" +
		"WHERE {{ range $i, $col := $primary_cols }}{{ if ne $i 0 }}AND {{ end }}{{ $col.Name.O }}={{ placeholder }} {{ end }}"

	res_, err_ := tx_.ExecContext(ctx_, sql_{{ range $i, $col := $non_primary_cols }}, entry_.{{ $col.Name }}{{ end }}{{ range $i, $col := $primary_cols }}, entry_.{{ $col.Name }}{{ end }})
	if err_ != nil {
		return err_
	}

	return nil

}
{{ end }}

func {{ $struct_name }}ByPrimaryKey(ctx_ {{ $ctx }}.Context, tx_ *{{ $sql }}.Tx{{ range $i, $col := $primary_cols }}, {{ $col.Name.CamelCase }} {{ $col.Type }}{{ end }}) (*{{ $struct_name }}, error) {
	const sql_ = "SELECT {{ printf "%s" (column_name_list $cols) }} FROM {{ $table_name }} " +
		"WHERE {{ range $i, $col := $primary_cols }}{{ if ne $i 0 }}AND {{ end }}{{ $col.Name.O }}={{ placeholder }} {{ end }} LIMIT 2"

	rows_, err_ := tx_.QueryContext(ctx_, sql_{{ range $i, $col := $primary_cols }}, {{ $col.Name.CamelCase }}{{ end }})
	if err_ != nil {
		return nil, err_
	}
	defer rows_.Close()

	ret_ := {{ $struct_name }}{}
	cnt_ := 0
	for rows_.Next() {
		cnt_ += 1
		if cnt_ >= 2 {
			return nil, {{ $fmt }}.Errorf("{{ $struct_name }}ByPrimaryKey returns more than one entry.")
		}
		if err_ := rows_.Scan({{ range $i, $col := $cols }}{{ if ne $i 0 }}, {{ end }}&ret_.{{ $col.Name }}{{ end }}); err_ != nil {
			return nil, err_
		}
	}

	if err_ := rows_.Err(); err_ != nil {
		return nil, err_
	}

	if cnt_ == 0 {
		return nil, nil
	}

	return &ret_, nil
}
{{ end -}}

`
	RegistDefaultTypeTemplate((*context.TableData)(nil), t)
}
