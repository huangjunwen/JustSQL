package render

import (
	"github.com/huangjunwen/JustSQL/context"
)

func init() {
	t := `
{{/* =========================== */}}
{{/*          imports            */}}
{{/* =========================== */}}
{{- $ctx := imp "context" -}}
{{- $fmt := imp "fmt" -}}
{{- $sql := imp "database/sql" -}}
{{- $driver := imp "database/sql/driver" -}}

{{/* =========================== */}}
{{/*          declares           */}}
{{/* =========================== */}}
{{- $table_name := .Table.Name.O -}}
{{- $struct_name := .Table.Name -}}
{{- $cols := .Table.Columns -}}
{{- $auto_inc_col := .Table.AutoIncColumn -}}
{{- $primary_cols := .Table.PrimaryColumns -}}
{{- $non_primary_cols := .Table.NonPrimaryColumns -}}

{{/* =========================== */}}
{{/*          enum and set       */}}
{{/* =========================== */}}
{{ range $i, $col := $cols }}
	{{- if $col.IsEnum }}
	{{- $enum_name := printf "%s%s" $struct_name $col.Name -}}

// Enum {{ $enum_name }}.
type {{ $enum_name }} int

const (
	// NULL value.
	{{ $enum_name }}NULL = {{ $enum_name }}(0)
{{- range $i, $elem := $col.Elems }}
	// {{ printf "%+q" $elem }}
	{{ $enum_name }}{{ if eq (len $elem) 0 }}Empty_{{ else }}{{ pascal $elem }}{{ end }} = {{ $enum_name }}({{ $i }} + 1)
{{- end }}
)

func (e {{ $enum_name }}) String() string {
	switch e {
{{- range $i, $elem := $col.Elems }}
{{- if ne (len $elem) 0 }}
	case {{ $enum_name }}{{ pascal $elem }}:
		return {{ printf "%+q" $elem }}
{{- end }}
{{- end }}
	}
	return ""
}

func (e {{ $enum_name }}) Valid() bool {
	return int(e) > 0 && int(e) <= {{ printf "%d" (len $col.Elems) }}
}

// Scan implements the Scanner interface.
func (e *{{ $enum_name }}) Scan(value interface{}) error {
	if value == nil {
		*e = {{ $enum_name }}NULL
		return nil
	}

	switch s := string(value.([]byte)); s {
{{- range $i, $elem := $col.Elems }}
	case {{ printf "%+q" $elem }}:
		*e = {{ $enum_name }}{{ if eq (len $elem) 0 }}Empty_{{ else }}{{ pascal $elem }}{{ end }}
{{- end }}
	default:
		return {{ $fmt }}.Errorf("Unexpected value for {{ $enum_name }}: %+q", s)
	}

	return nil
}

// Value implements the driver Valuer interface.
func (e {{ $enum_name }}) Value() (driver.Value, error) {
	if !e.Valid() {
		return nil, nil
	}
	return e.String(), nil
}

	{{- else if $col.IsSet }}

	{{- end -}}
{{ end }}


{{/* =========================== */}}
{{/*          main struct        */}}
{{/* =========================== */}}
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

func (entry_ *{{ $struct_name }}) Select(ctx_ {{ $ctx }}.Context, tx_ *{{ $sql }}.Tx) error {
	const sql_ = "SELECT {{ printf "%s" (column_name_list $cols) }} FROM {{ $table_name }} " +
		"WHERE {{ range $i, $col := $primary_cols }}{{ if ne $i 0 }}AND {{ end }}{{ $col.Name.O }}={{ placeholder }} {{ end }} LIMIT 2"

	row_, err_ := tx_.QueryRowContext(ctx_, sql_{{ range $i, $col := $primary_cols }}, entry_.{{ $col.Name }}{{ end }})
	if err_ != nil {
		return err_
	}
	
	if err_ := row_.Scan({{ range $i, $col := $cols }}{{ if ne $i 0 }}, {{ end }}&entry_.{{ $col.Name }}{{ end }}); err_ != nil {
		return err_
	}

	return nil
}

{{ if ne (len $non_primary_cols) 0 -}}
func (entry_ *{{ $struct_name }}) Update(ctx_ {{ $ctx }}.Context, tx_ *{{ $sql }}.Tx) error {
	const sql_ = "UPDATE {{ $table_name }} SET {{ range $i, $col := $non_primary_cols }}{{ if ne $i 0 }}, {{ end }}{{ $col.Name.O }}={{ placeholder }}{{ end }}" +
		" WHERE {{ range $i, $col := $primary_cols }}{{ if ne $i 0 }}AND {{ end }}{{ $col.Name.O }}={{ placeholder }} {{ end }}"

	res_, err_ := tx_.ExecContext(ctx_, sql_{{ range $i, $col := $non_primary_cols }}, entry_.{{ $col.Name }}{{ end }}{{ range $i, $col := $primary_cols }}, entry_.{{ $col.Name }}{{ end }})
	if err_ != nil {
		return err_
	}

	return nil
}
{{ end }}

func (entry_ *{{ $struct_name }}) Delete(ctx_ {{ $ctx }}.Context, tx_ *{{ $sql }}.Tx) error {
	const sql_ = "DELETE FROM {{ $table_name }} " +
		"WHERE {{ range $i, $col := $primary_cols }}{{ if ne $i 0 }}AND {{ end }}{{ $col.Name.O }}={{ placeholder }} {{ end }}"

	res_, err_ := tx_.ExecContext(ctx_, sql_{{ range $i, $col := $primary_cols }}, entry_.{{ $col.Name }}{{ end }})
	if err_ != nil {
		return err_
	}

	return nil
}

{{ end }}

`
	RegistDefaultTypeTemplate((*context.TableData)(nil), t)
}
