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
{{- $return_type := printf "%sResult" .Func.Name -}}
{{- $return_one := eq .Func.Return "one" -}}

{{/* =========================== */}}
{{/*          main function      */}}
{{/* =========================== */}}

type {{ $return_type }} struct {
{{- range $rf := $rfs }}
	{{- if and (or $rf.IsEnum $rf.IsSet) (not_nil $rf.Table) }}
	{{ $rf.PascalName }} {{ $rf.Table.PascalName}}{{ $rf.Column.PascalName }}
	{{- else }}
	{{ $rf.PascalName }} {{ $rf.AdaptType }}
	{{- end }}
{{- end }}
}

const _{{ $func_name }}QueryTmpl = template.Must(template.New({{ printf "%q" .Func.Name }}).Parse(
` + "`" + `{{ printf "%s" .Func.Query }}` + "`" + `))

// Generated from: {{ printf "%+q" .Src }}
func {{ $func_name }}(ctx_ {{ $ctx }}.Context, tx_ *{{ $sqlx }}.Tx{{ range $arg := .Func.Args }}, {{ $arg.Name}} {{ $arg.AdaptType }} {{ end }}) ({{ if $return_one }}*{{ $return_type }}{{ else }}[]*{{ $return_type }}{{ end }}, error) {

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
	ret_ := new({{ $return_type }})
	if err_ := row_.Scan({{ range $i, $rf := $rfs }}{{ if ne $i 0 }}, {{ end }}&ret_.{{ $rf.PascalName }}{{ end }}); err != nil {
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
	ret_ := make([]*{{ $return_type }}, 0)
	for rows_.Rows.Next() {
		r_ := new({{ $return_type }})
		if err_ := rows_.Rows.Scan({{ range $i, $rf := $rfs }}{{ if ne $i 0 }}, {{ end }}&r_.{{ $rf.PascalName }}{{ end }}); err != nil {
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
