package render

import (
	"github.com/pingcap/tidb/ast"
)

func init() {
	t := `
{{/* =========================== */}}
{{/*          imports            */}}
{{/* =========================== */}}
{{- $ctx := imp "context" -}}
{{- $sqlx := imp "github.com/jmoiron/sqlx" -}}
{{- $template := imp "text/template" -}}
{{- $bytes := imp "bytes" -}}

{{/* =========================== */}}
{{/*          declares           */}}
{{/* =========================== */}}
{{- $has_multi_arg := .Func.HasMultiArg -}}
{{- $func_name := .Func.Name -}}
{{- $rfs := .Stmt.ResultFields -}}
{{- $result_name := printf "%sResult" .Func.Name -}}
{{- $result_field_names := unique_names -}}
{{- $result_fields_flatten := string_arr -}}
{{- $return_one := eq .Func.Return "one" -}}

{{/* =========================== */}}
{{/*          main function      */}}
{{/* =========================== */}}

type {{ $result_name }} struct {
{{- range $i, $rf := $rfs -}}
	{{/* whether this result field is in a normal table wildcard expansion */}}
	{{- $wildcard_tbl_src_name := $.Wildcard.TableSourceName $i -}}
	{{- $wildcard_tbl_is_normal := and ($.Stmt.Sources.Has $wildcard_tbl_src_name) (not ($.Stmt.Sources.HasDerived $wildcard_tbl_src_name)) -}}
	{{- $wildcard_offset := $.Wildcard.Offset $i -}}

	{{- if $wildcard_tbl_is_normal }}
		{{- if eq $wildcard_offset 0 }}
			{{- $result_field_names.Add $wildcard_tbl_src_name }}
			{{ $result_field_names.Last }} {{ $rf.Table.PascalName }} // {{ $wildcard_tbl_src_name }}.*
		{{- end }}
		{{- $result_fields_flatten.Push (printf "%s.%s" $result_field_names.Last $rf.Column.PascalName) }}
	{{- else }}
		{{- $result_field_names.Add $rf.Name -}}
		{{- if and (or $rf.IsEnum $rf.IsSet) (not_nil $rf.Table) }}
			{{ $result_field_names.Last }} {{ $rf.Table.PascalName }}{{ $rf.Column.PascalName }}
		{{- else }}
			{{ $result_field_names.Last }} {{ $rf.AdaptType }}
		{{- end }}
		{{- $result_fields_flatten.Push $result_field_names.Last }}
	{{- end }}
{{- end }}
}

const _{{ $func_name }}QueryTmpl = template.Must(template.New({{ printf "%q" .Func.Name }}).Parse(
` + "`" + `{{ printf "%s" .Func.Query }}` + "`" + `))

// Generated from: {{ printf "%+q" .Src }}
func {{ $func_name }}(ctx_ {{ $ctx }}.Context, tx_ *{{ $sqlx }}.Tx{{ range $arg := .Func.Args }}, {{ $arg.Name}} {{ $arg.AdaptType }} {{ end }}) ({{ if $return_one }}*{{ $result_name }}{{ else }}[]*{{ $result_name }}{{ end }}, error) {

	// - Dot object for template and query parameter.
	dot_ := map[string]interface{}{
{{- range $arg := .Func.Args }}
		{{ printf "%q" $arg.Name }}: {{ $arg.Name }},
{{- end }}
	}

	// - Render from template.
	buf_ := new(bytes.Buffer)
	if err_ := _{{ .Func.Name }}QueryTmpl.Execute(buf_, dot_); err_ != nil {
		return nil, err_
	}

	// - Handle named query.
	query_, args_, err_ := {{ $sqlx }}.Named(buf_.String(), dot_)
	if err_ != nil {
		return nil, err_
	}

{{ if $has_multi_arg -}}
	// - Handle "IN (?)".
	query_, args_, err_ := {{ $sqlx }}.In(query_, args_...)
	if err_ != nil {
		return nil, err_
	}
{{- end }}

	// - Rebind.
	query_ = tx_.Rebind(query_)

{{ if $return_one -}}
	// - Query.
	row_, err_ := tx_.QueryRowxContext(ctx_, query_, args_...)
	if err_ != nil {
		return nil, err_
	}

	// - Scan.
	ret_ := new({{ $result_name }})
	if err_ := row_.Scan({{ range $i, $field := $result_fields_flatten }}{{ if ne $i 0 }}, {{ end }}&ret_.{{ $field }}{{ end }}); err != nil {
		return nil, err
	}

	return ret_, nil
{{ else -}}
	// - Query.
	rows_, err_ := tx_.QueryxContext(ctx_, query_, args_...)
	if err_ != nil {
		return nil, err_
	}
	defer rows_.Rows.Close()

	// - Scan.
	ret_ := make([]*{{ $result_name }}, 0)
	for rows_.Rows.Next() {
		r_ := new({{ $result_name }})
		if err_ := rows_.Rows.Scan({{ range $i, $field := $result_fields_flatten }}{{ if ne $i 0 }}, {{ end }}&r_.{{ $field }}{{ end }}); err != nil {
			return nil, err_
		}
		ret_ = append(ret_, r_)
	}
	
	if err_ := rows_.Rows.Err(); err_ != nil {
		return nil, err_
	}

	return ret_, nil

{{- end }}

}

`
	RegistDefaultTypeTemplate((*ast.SelectStmt)(nil), t)
}
