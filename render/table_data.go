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
		return {{ $fmt }}.Errorf("Unknown enum value %+q for {{ printf "%s" $enum_name }}", s)
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
	{{ $column.Name }} {{ $.Table.Name }}{{ $column.Name }} // {{ $column.Name.O }}
	{{- else if $column.IsSet }}
	{{ $column.Name }} {{ $.Table.Name }}{{ $column.Name }} // {{ $column.Name.O }}
	{{- else }}
	{{ $column.Name }} {{ $column.Type }} // {{ $column.Name.O }}
	{{- end -}}
{{ end }}
}

func (entry *{{ $struct_name }}) Insert(ctx {{ $ctx }}.Context, db *{{ $sql }}.DB) error {
}

`
	RegistDefaultTypeTemplate((*context.TableData)(nil), t)
}
