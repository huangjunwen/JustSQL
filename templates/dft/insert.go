package dft

import (
	"github.com/huangjunwen/JustSQL/render"
)

func init() {
	render.RegistBuiltinTemplate("insert", render.DefaultTemplateSetName, `
{{/* =========================== */}}
{{/*          imports            */}}
{{/* =========================== */}}
{{- $ctx := imp "context" -}}
{{- $sqlx := imp "github.com/jmoiron/sqlx" -}}
{{- $template := imp "text/template" -}}

{{/* =========================== */}}
{{/*          variables          */}}
{{/* =========================== */}}
{{- $funcName := .Annot.FuncName -}}

{{/* =========================== */}}
{{/*        main function        */}}
{{/* =========================== */}}
// {{ $funcName }} is generated from:
//
{{- range $line := split .Stmt.InsertStmt.Text "\n" }}
	{{- if ne (len $line) 0 }}
//    {{ printf "%s" $line }}
	{{- end }}
{{- end }}
//
func {{ $funcName }}(ctx_ {{ $ctx }}.Context, db_ DBer{{ range $arg := .Annot.Args }}, {{ $arg.Name }} {{ typeName $arg.Type }} {{ end }}) (int64, error) {

	const sql_ = "" +
{{- range $line := split .Annot.Text "\n" }}
	{{- $lineSP := printf "%s%s" $line " " }}
		{{ printf "%+q" $lineSP }} +
{{- end }}""

	// - Dot object for query parameter.
	dot_ := map[string]interface{}{
{{- range $arg := .Annot.Args }}
		{{ printf "%q" $arg.Name }}: {{ $arg.Name }},
{{- end }}
	}

	// - Handle named query.
	query_, args_, err_ := {{ $sqlx }}.Named(sql_, dot_)
	if err_ != nil {
		return nil, err_
	}

	query_ = {{ $sqlx }}.Rebind(BindType, query_)

	// - Execute.
	return db_.ExecContext(ctx_, query_, args_...).RowsAffected()
	
}


`)

}
