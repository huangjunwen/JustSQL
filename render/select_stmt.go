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

{{/* =========================== */}}
{{/*          main function      */}}
{{/* =========================== */}}

const _{{ .Func.Name }}QueryTmpl = template.Must(template.New({{ printf "%q" .Func.Name }}).Parse(` + "`" + `{{ printf "%s" .Func.Query }}` + "`" + `))

func {{ .Func.Name }}(ctx_ {{ $ctx }}.Context, tx_ *{{ $sqlx }}.Tx{{ range $arg := .Func.Args }}, {{ $arg.Name}} {{ $arg.AdaptType }} {{ end }}) error {

	// Dot object for template and query parameter.
	dot_ := map[string]interface{}{
{{- range $arg := .Func.Args }}
		{{ printf "%q" $arg.Name }}: {{ $arg.Name }},
{{- end }}
	}

	// - Render from template.
	buf_ := new(bytes.Buffer)
	if err_ := _{{ .Func.Name }}QueryTmpl.Execute(buf_, dot_); err_ != nil {
		return err_
	}

	// - Handle named query.
	query_, args_, err_ := {{ $sqlx }}.Named(buf_.String(), dot_)
	if err_ != nil {
		return err_
	}

{{ if $has_multi_arg -}}
	// - Handle "IN (?)".
	query_, args_, err_ := {{ $sqlx }}.In(query_, args_...)
	if err_ != nil {
		return err_
	}
{{- end }}

	// - Rebind.
	query_ = tx_.Rebind(query_)

	// - Query.
	rows_, err_ := tx_.QueryxContext(ctx_, query_, args_...)
	if err_ != nil {
		return err_
	}

}

`
	RegistDefaultTypeTemplate((*ast.SelectStmt)(nil), t)
}
