package dft

import (
	"github.com/huangjunwen/JustSQL/render"
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
{{- $strings := imp "strings" -}}

{{/* =========================== */}}
{{/*          declares           */}}
{{/* =========================== */}}
{{- $table_name := .Table.PascalName -}}
{{- $struct_name := .Table.PascalName -}}
{{- $cols := .Table.Columns -}}
{{- $auto_inc_col := .Table.AutoIncColumn -}}
{{- $primary_cols := .Table.PrimaryColumns -}}
{{- $non_primary_cols := .Table.NonPrimaryColumns -}}

{{ range $i, $col := $cols }}
	{{- if $col.IsEnum }}
{{/* =========================== */}}
{{/*          enum               */}}
{{/* =========================== */}}
{{- $enum_name := printf "%s%s" $struct_name $col.PascalName -}}

// Enum {{ $enum_name }}.
type {{ $enum_name }} int

const (
	// NULL value.
	{{ $enum_name }}NULL = {{ $enum_name }}(0)
{{- range $i, $elem := $col.Elems }}
	// {{ printf "%+q" $elem }}
	{{ $enum_name }}{{ pascal $elem }} = {{ $enum_name }}({{ $i }} + 1)
{{- end }}
)

func (e {{ $enum_name }}) String() string {
	switch e {
{{- range $i, $elem := $col.Elems }}
	case {{ $enum_name }}{{ pascal $elem }}:
		return {{ printf "%+q" $elem }}
{{- end }}
	}
	return ""
}

func (e {{ $enum_name }}) Valid() bool {
	return int(e) > 0 && int(e) <= {{ printf "%d" (len $col.Elems) }}
}

// Scan implements database/sql.Scanner interface.
func (e *{{ $enum_name }}) Scan(value interface{}) error {
	if value == nil {
		*e = {{ $enum_name }}NULL
		return nil
	}

	switch s := string(value.([]byte)); s {
{{- range $i, $elem := $col.Elems }}
	case {{ printf "%+q" $elem }}:
		*e = {{ $enum_name }}{{ pascal $elem }}
{{- end }}
	default:
		return {{ $fmt }}.Errorf("Unexpected value for {{ $enum_name }}: %+q", s)
	}

	return nil
}

// Value implements database/sql/driver.Valuer interface.
func (e {{ $enum_name }}) Value() (driver.Value, error) {
	if !e.Valid() {
		return nil, nil
	}
	return e.String(), nil
}

	{{- else if $col.IsSet }}
{{/* =========================== */}}
{{/*          set                */}}
{{/* =========================== */}}
{{- $set_name := printf "%s%s" $struct_name $col.PascalName -}}

// Set {{ $set_name }}.
type {{ $set_name }} struct {
	val uint64 // Up to 64 distinct members. See https://dev.mysql.com/doc/refman/5.7/en/set.html
	valid bool // NULL if valid is false.
}

const (
{{- range $i, $elem := $col.Elems }}
	// {{ printf "%+q" $elem }}
	{{ $set_name }}{{ pascal $elem }} = uint64(1<<{{ $i }})
{{- end }}
)

func New{{ $set_name }}(items ...uint64) {{ $set_name }} {
	var val uint64 = 0
	for _, item := range items {
		if item > 0 && (item & (item - 1)) == 0 && item <= (1 << ({{ len $col.Elems }} - 1)) {
			val |= item 
		}
	}
	return {{ $set_name }}{
		val: val,
		valid: true,
	}
}

func (s {{ $set_name }}) String() string {
	parts := make([]string, 0)
{{- range $i, $elem := $col.Elems }}
	if s.val & {{ $set_name }}{{ pascal $elem }} != 0 {
		parts = append(parts, {{ printf "%+q" $elem }})
	}
{{- end }}
	return strings.Join(parts, ",")
}

func (s {{ $set_name }}) Valid() bool {
	return s.valid
}

// Scan implements database/sql.Scanner interface.
func (s *{{ $set_name }}) Scan(value interface{}) error {
	if value == nil {
		s.val = 0
		s.valid = false
		return nil
	}

	var val uint64 = 0
	for _, part := range {{ $strings }}.Split(string(value.([]byte)), ",") {
		switch part {
{{- range $i, $elem := $col.Elems }}
		case {{ printf "%+q" $elem }}:
			val |= {{ $set_name }}{{ pascal $elem }}
{{- end }}
		default:
			return {{ $fmt }}.Errorf("Unexpected value for {{ $set_name }}: %+q", part)
		}
	}

	s.val = val
	s.valid = true
	return nil
}

// Value implements database/sql/driver.Valuer interface.
func (s {{ $set_name }}) Value() (driver.Value, error) {
	if !s.Valid() {
		return nil, nil
	}
	return s.String(), nil
}

	{{- end -}}
{{ end }}


{{/* =========================== */}}
{{/*          main struct        */}}
{{/* =========================== */}}
// Table {{ $table_name }}
type {{ $struct_name }} struct {
{{ range $i, $col := $cols }}
	{{- if or $col.IsEnum $col.IsSet }}
	{{ $col.PascalName }} {{ $struct_name }}{{ $col.PascalName }}
	{{- else }}
	{{ $col.PascalName }} {{ $col.AdaptType }}
	{{- end -}}
	{{- " " -}}// {{ $col.Name }}: {{ if $col.IsNotNULL }}NOT NULL;{{ else }}NULL;{{ end }}{{ if $col.IsAutoInc }} AUTO INCREMENT;{{ end }} DEFAULT {{ printf "%#v" $col.DefaultValue }};{{ if $col.IsOnUpdateNow }} ON UPDATE "CURRENT_TIMESTAMP";{{ end }}
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

	entry_.{{ $auto_inc_col.PascalName }} = {{ $auto_inc_col.AdaptType }}(last_insert_id_)
	{{ end -}}

	return nil
}

{{ if ne (len $primary_cols) 0 -}}

func (entry_ *{{ $struct_name }}) Select(ctx_ {{ $ctx }}.Context, tx_ *{{ $sql }}.Tx) error {
	const sql_ = "SELECT {{ printf "%s" (column_name_list $cols) }} FROM {{ $table_name }} " +
		"WHERE {{ range $i, $col := $primary_cols }}{{ if ne $i 0 }}AND {{ end }}{{ $col.Name }}={{ placeholder }} {{ end }}"

	row_ := tx_.QueryRowContext(ctx_, sql_{{ range $i, $col := $primary_cols }}, entry_.{{ $col.PascalName }}{{ end }})
	if err_ := row_.Scan({{ range $i, $col := $cols }}{{ if ne $i 0 }}, {{ end }}&entry_.{{ $col.PascalName }}{{ end }}); err_ != nil {
		return err_
	}

	return nil
}

{{ if ne (len $non_primary_cols) 0 -}}
func (entry_ *{{ $struct_name }}) Update(ctx_ {{ $ctx }}.Context, tx_ *{{ $sql }}.Tx) error {
	const sql_ = "UPDATE {{ $table_name }} SET {{ range $i, $col := $non_primary_cols }}{{ if ne $i 0 }}, {{ end }}{{ $col.Name }}={{ placeholder }}{{ end }}" +
		" WHERE {{ range $i, $col := $primary_cols }}{{ if ne $i 0 }}AND {{ end }}{{ $col.Name }}={{ placeholder }} {{ end }}"

	_, err_ := tx_.ExecContext(ctx_, sql_{{ range $i, $col := $non_primary_cols }}, entry_.{{ $col.PascalName }}{{ end }}{{ range $i, $col := $primary_cols }}, entry_.{{ $col.PascalName }}{{ end }})
	if err_ != nil {
		return err_
	}

	return nil
}
{{ end }}

func (entry_ *{{ $struct_name }}) Delete(ctx_ {{ $ctx }}.Context, tx_ *{{ $sql }}.Tx) error {
	const sql_ = "DELETE FROM {{ $table_name }} " +
		"WHERE {{ range $i, $col := $primary_cols }}{{ if ne $i 0 }}AND {{ end }}{{ $col.Name }}={{ placeholder }} {{ end }}"

	_, err_ := tx_.ExecContext(ctx_, sql_{{ range $i, $col := $primary_cols }}, entry_.{{ $col.PascalName }}{{ end }})
	if err_ != nil {
		return err_
	}

	return nil
}

{{ end }}

`
	render.RegistBuiltinTemplate("table", render.DefaultTemplateName, t)
}
