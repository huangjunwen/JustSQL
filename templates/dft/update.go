package dft

import (
	"github.com/huangjunwen/JustSQL/render"
)

func init() {
	render.RegistBuiltinTemplate("update", render.DefaultTemplateName, `
{{/* =========================== */}}
{{/*          imports            */}}
{{/* =========================== */}}
{{- $ctx := imp "context" -}}
{{- $sqlx := imp "github.com/jmoiron/sqlx" -}}
{{- $template := imp "text/template" -}}
{{- $bytes := imp "bytes" -}}

{{/* =========================== */}}
{{/*          variables          */}}
{{/* =========================== */}}
{{- $funcName := .Annot.FuncName -}}
{{- $hasInBinding := ne (.Annot.Env "hasInBinding") "" -}}

{{/* =========================== */}}
{{/*        main function        */}}
{{/* =========================== */}}
var _{{ $funcName }}SQLTmpl = {{ $template }}.Must({{ $template }}.New({{ printf "%q" $funcName }}).Parse("" +
{{- range $line := split .Annot.Text "\n" }}
	"{{ printf "%s" $line }} " +
{{- end }}""))

// {{ $funcName }} is generated from:
//
{{- range $line := split .Stmt.UpdateStmt.Text "\n" }}
	{{- if ne (len $line) 0 }}
//    {{ printf "%s" $line }}
	{{- end }}
{{- end }}
//
func {{ $funcName }}(ctx_ {{ $ctx }}.Context, db_ DBer{{ range $arg := .Annot.Args }}, {{ $arg.Name }} {{ typeName $arg.Type }} {{ end }}) (int64, error) {

	// - Dot object for template and query parameter.
	dot_ := map[string]interface{}{
{{- range $arg := .Annot.Args }}
		{{ printf "%q" $arg.Name }}: {{ $arg.Name }},
{{- end }}
	}

	// - Render from template.
	buf_ := new({{ $bytes }}.Buffer)
	if err_ := _{{ $funcName }}SQLTmpl.Execute(buf_, dot_); err_ != nil {
		return nil, err_
	}

	// - Handle named query.
	query_, args_, err_ := {{ $sqlx }}.Named(buf_.String(), dot_)
	if err_ != nil {
		return nil, err_
	}

{{ if $hasInBinding -}}
	// - Handle "IN (?)".
	query_, args_, err_ = {{ $sqlx }}.In(query_, args_...)
	if err_ != nil {
		return nil, err_
	}
{{- end }}

	// - Rebind.
	query_ = {{ $sqlx }}.Rebind(BindType, query_)

	// - Execute.
	return db_.ExecContext(ctx_, sql_, args_...).RowsAffected()
	
}


`)

}
