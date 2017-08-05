package dft

import (
	"github.com/huangjunwen/JustSQL/render"
)

func init() {
	render.RegistBuiltinTemplate("table", render.DefaultTemplateName, `
{{/* =========================== */}}
{{/*          imports            */}}
{{/* =========================== */}}
{{- $ctx := imp "context" -}}
{{- $fmt := imp "fmt" -}}
{{- $sqlx := imp "database/sqlx" -}}
{{- $driver := imp "database/sql/driver" -}}
{{- $strings := imp "strings" -}}

{{/* =========================== */}}
{{/*          declares           */}}
{{/* =========================== */}}
{{- $tableName := .Table.Name -}}
{{- $structName := .Table.PascalName -}}
{{- $structFields := strings -}}
{{- $cols := .Table.Columns -}}
{{- $autoIncCol := .Table.AutoIncColumn -}}
{{- $primaryCols := .Table.PrimaryColumns -}}
{{- $nonPrimaryCols := .Table.NonPrimaryColumns -}}

{{ range $i, $col := $cols }}
	{{- if $col.IsEnum }}
	{{/* =========================== */}}
	{{/*          enum               */}}
	{{/* =========================== */}}
	{{- $enumName := printf "%s%s" $structName $col.PascalName -}}
	{{- $enumItems := strings -}}

// Enum {{ $enumName }}.
type {{ $enumName }} int

// Enum {{ $enumName }} items.
const (
	// NULL value.
	{{ $enumName }}NULL = {{ $enumName }}(0)
	{{- range $i, $elem := $col.Elems }}
		{{- if eq $elem "" }}
			{{- $enumItems.Add (printf "%sEmpty_" $enumName) }}
		{{- else }}
			{{- $enumItems.Add (printf "%s%s" $enumName (pascal $elem)) }}
		{{- end }}
	// {{ printf "%+q" $elem }}
	{{ $enumItems.Last }} = {{ $enumName }}({{ $i }} + 1)
	{{- end }}
)

func (e {{ $enumName }}) String() string {
	switch e {
	{{- range $i, $item := $enumItems }}
	case {{ $item }}:
		return {{ printf "%+q" (index $col.Elems $i) }}
	{{- end }}
	}
	return ""
}

func (e {{ $enumName }}) Valid() bool {
	return int(e) > 0 && int(e) <= {{ printf "%d" (len $col.Elems) }}
}

// Scan implements database/sql.Scanner interface.
func (e *{{ $enumName }}) Scan(value interface{}) error {
	if value == nil {
		*e = {{ $enumName }}NULL
		return nil
	}

	switch s := string(value.([]byte)); s {
	{{- range $i, $elem := $col.Elems }}
	case {{ printf "%+q" $elem }}:
		*e = {{ index $enumItems $i }}
	{{- end }}
	default:
		return {{ $fmt }}.Errorf("Unexpected value for {{ $enumName }}: %+q", s)
	}

	return nil
}

// Value implements database/sql/driver.Valuer interface.
func (e {{ $enumName }}) Value() (driver.Value, error) {
	if !e.Valid() {
		return nil, nil
	}
	return e.String(), nil
}

	{{- else if $col.IsSet }}
	{{/* =========================== */}}
	{{/*          set                */}}
	{{/* =========================== */}}
	{{- $setName := printf "%s%s" $structName $col.PascalName -}}
	{{- $setItems := strings -}}

// Set {{ $setName }}.
type {{ $setName }} struct {
	val uint64 // Up to 64 distinct members. See https://dev.mysql.com/doc/refman/5.7/en/set.html
	valid bool // NULL if valid is false.
}

// Set {{ $setName }} items.
const (
	{{- range $i, $elem := $col.Elems }}
		{{- if eq $elem "" }}
			{{- $setItems.Add (printf "%sEmpty_" $setName) }}
		{{- else }}
			{{- $setItems.Add (printf "%s%s" $setName (pascal $elem)) }}
		{{- end }}
	// {{ printf "%+q" $elem }}
	{{ $setItems.Last }} = uint64(1<<{{ $i }})
	{{- end }}
)

func New{{ $setName }}(items ...uint64) {{ $setName }} {
	var val uint64 = 0
	for _, item := range items {
		if item > 0 && (item & (item - 1)) == 0 && item <= (1 << ({{ len $col.Elems }} - 1)) {
			val |= item 
		}
	}
	return {{ $setName }}{
		val: val,
		valid: true,
	}
}

func (s {{ $setName }}) String() string {
	parts := make([]string, 0)
	{{- range $i, $item:= $setItems }}
	if s.val & {{ $item }} != 0 {
		parts = append(parts, {{ printf "%+q" (index $col.Elems $i) }})
	}
	{{- end }}
	return strings.Join(parts, ",")
}

func (s {{ $setName }}) Valid() bool {
	return s.valid
}

// Scan implements database/sql.Scanner interface.
func (s *{{ $setName }}) Scan(value interface{}) error {
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
			val |= {{ index $setItems $i }}
	{{- end }}
		default:
			return {{ $fmt }}.Errorf("Unexpected value for {{ $setName }}: %+q", part)
		}
	}

	s.val = val
	s.valid = true
	return nil
}

// Value implements database/sql/driver.Valuer interface.
func (s {{ $setName }}) Value() (driver.Value, error) {
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
// Table {{ $tableName }}
type {{ $structName }} struct {
{{ range $i, $col := $cols }}
	{{- $structFields.Add $col.PascalName }}
	{{- if or $col.IsEnum $col.IsSet }}
	{{ $structFields.Last }} {{ $structName }}{{ $col.PascalName }}
	{{- else }}
	{{ $structFields.Last }} {{ $col.AdaptType }}
	{{- end -}}
	{{- " " -}}`+"`db:\"{{ $col.Name }}\"`"+`
	{{- " " -}}// {{ $col.Name }}
{{- end }}
}

func (entry_ *{{ $structName }}) Insert(ctx_ {{ $ctx }}.Context, db_ DBer) error {

	sql_ := {{ $sqlx }}.Rebind(BindType,
		"INSERT INTO {{ $tableName }} ({{ columnNameList $cols }}) VALUES ({{ repeatJoin (len $cols) "?" ", " }})")

	{{ if notNil $autoIncCol }}res_{{ else }}_{{ end }}, err_ := db_.ExecContext(ctx_, sql_{{ range $i, $field := $structFields }}, entry_.{{ $field }}{{ end }})
	if err_ != nil {
		return err_
	}

	{{ if notNil $autoIncCol -}}
	lastInsertId_, err_ := res_.LastInsertId()
	if err_ != nil {
		return err_
	}

	entry_.{{ $autoIncCol.PascalName }} = {{ $autoIncCol.AdaptType }}(lastInsertId_)
	{{ end -}}

	return nil
}

{{ if ne (len $primaryCols) 0 -}}

func (entry_ *{{ $structName }}) Select(ctx_ {{ $ctx }}.Context, db_ DBer) error {

	sql_ := {{ $sqlx }}.Rebind(BindType, "SELECT {{ columnNameList $cols }} FROM {{ $tableName }} " +
		"WHERE {{ range $i, $col := $primaryCols }}{{ if ne $i 0 }}AND {{ end }}{{ $col.Name }}=? {{ end }}")

	row_ := db_.QueryRowContext(ctx_, sql_{{ range $i, $col := $primaryCols }}, entry_.{{ $col.PascalName }}{{ end }})
	if err_ := row_.Scan({{ range $i, $field := $structFields }}{{ if ne $i 0 }}, {{ end }}&entry_.{{ $field }}{{ end }}); err_ != nil {
		return err_
	}

	return nil
}

{{ if ne (len $nonPrimaryCols) 0 -}}
func (entry_ *{{ $structName }}) Update(ctx_ {{ $ctx }}.Context, db_ DBer) (int64, error) {

	sql_ := {{ $sqlx }}.Rebind(BindType,
		"UPDATE {{ $tableName }} SET {{ range $i, $col := $nonPrimaryCols }}{{ if ne $i 0 }}, {{ end }}{{ $col.Name }}=?{{ end }}" +
		" WHERE {{ range $i, $col := $primaryCols }}{{ if ne $i 0 }}AND {{ end }}{{ $col.Name }}=? {{ end }}")

	r_, err_ := db_.ExecContext(ctx_, sql_{{ range $i, $col := $nonPrimaryCols }}, entry_.{{ $col.PascalName }}{{ end }}{{ range $i, $col := $primaryCols }}, entry_.{{ $col.PascalName }}{{ end }})
	if err_ != nil {
		return 0, err_
	}

	return r_.RowsAffected()

}
{{ end }}

func (entry_ *{{ $structName }}) Delete(ctx_ {{ $ctx }}.Context, db_ DBer) (int64, error) {

	sql_ := {{ $sqlx }}.Rebind(BindType,
		"DELETE FROM {{ $tableName }} " +
		"WHERE {{ range $i, $col := $primaryCols }}{{ if ne $i 0 }}AND {{ end }}{{ $col.Name }}=? {{ end }}")

	r_, err_ := db_.ExecContext(ctx_, sql_{{ range $i, $col := $primaryCols }}, entry_.{{ $col.PascalName }}{{ end }})
	if err_ != nil {
		return 0, err_
	}

	return r_.RowsAffected()
}

{{ end }}

`)

}
