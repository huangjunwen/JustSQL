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
{{- $hasMultiArg := .Func.HasMultiArg -}}
{{- $funcName := .Func.Name -}}
{{- $rfs := .Stmt.ResultFields -}}
{{- $returnType := printf "%sResult" .Func.Name -}}
{{- $returnTypeFields := unique_names -}}
{{- $returnTypeFieldsFlatten := string_arr -}}
{{- $returnOne := eq .Func.Return "one" -}}

{{/* =========================== */}}
{{/*          main function      */}}
{{/* =========================== */}}

type {{ $returnType }} struct {
{{- range $i, $rf := $rfs -}}
	{{/* whether this result field is in a normal table wildcard expansion */}}
	{{- $wildcardTableRefName := $.OriginStmt.FieldList.WildcardTableRefName $i -}}
	{{- $wildcardTableIsNormal := and ($.OriginStmt.TableRefs.Has $wildcardTableRefName) (not ($.OriginStmt.TableRefs.HasDerived $wildcardTableRefName)) -}}
	{{- $wildcardOffset := $.OriginStmt.FieldList.WildcardOffset $i -}}

	{{- if $wildcardTableIsNormal }}
		{{- if eq $wildcardOffset 0 }}
			{{- $returnTypeFields.Add $wildcardTableRefName }}
			{{ $returnTypeFields.Last }} {{ $rf.Table.PascalName }} // {{ $wildcardTableRefName }}.*
		{{- end }}
		{{- $returnTypeFieldsFlatten.Push (printf "%s.%s" $returnTypeFields.Last $rf.Column.PascalName) }}
	{{- else }}
		{{- $returnTypeFields.Add $rf.Name -}}
		{{- if and (or $rf.IsEnum $rf.IsSet) (not_nil $rf.Table) }}
			{{ $returnTypeFields.Last }} {{ $rf.Table.PascalName }}{{ $rf.Column.PascalName }}
		{{- else }}
			{{ $returnTypeFields.Last }} {{ $rf.AdaptType }}
		{{- end }}
		{{- $returnTypeFieldsFlatten.Push $returnTypeFields.Last }}
	{{- end }}

{{- end }}
}

const _{{ $funcName }}QueryTmpl = template.Must(template.New({{ printf "%q" $funcName }}).Parse(
` + "`" + `{{ printf "%s" .Func.Query }}` + "`" + `))

// Generated from: {{ printf "%+q" .OriginStmt.SelectStmt.Text }}
func {{ $funcName }}(ctx_ {{ $ctx }}.Context, tx_ *{{ $sqlx }}.Tx{{ range $arg := .Func.Args }}, {{ $arg.Name}} {{ $arg.AdaptType }} {{ end }}) ({{ if $returnOne }}*{{ $returnType }}{{ else }}[]*{{ $returnType }}{{ end }}, error) {

	// - Dot object for template and query parameter.
	dot_ := map[string]interface{}{
{{- range $arg := .Func.Args }}
		{{ printf "%q" $arg.Name }}: {{ $arg.Name }},
{{- end }}
	}

	// - Render from template.
	buf_ := new(bytes.Buffer)
	if err_ := _{{ $funcName }}QueryTmpl.Execute(buf_, dot_); err_ != nil {
		return nil, err_
	}

	// - Handle named query.
	query_, args_, err_ := {{ $sqlx }}.Named(buf_.String(), dot_)
	if err_ != nil {
		return nil, err_
	}

{{ if $hasMultiArg -}}
	// - Handle "IN (?)".
	query_, args_, err_ := {{ $sqlx }}.In(query_, args_...)
	if err_ != nil {
		return nil, err_
	}
{{- end }}

	// - Rebind.
	query_ = tx_.Rebind(query_)

{{ if $returnOne -}}
	// - Query.
	row_, err_ := tx_.QueryRowxContext(ctx_, query_, args_...)
	if err_ != nil {
		return nil, err_
	}

	// - Scan.
	ret_ := new({{ $returnType }})
	if err_ := row_.Scan({{ range $i, $field := $returnTypeFieldsFlatten }}{{ if ne $i 0 }}, {{ end }}&ret_.{{ $field }}{{ end }}); err != nil {
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
	ret_ := make([]*{{ $returnType }}, 0)
	for rows_.Rows.Next() {
		r_ := new({{ $returnType }})
		if err_ := rows_.Rows.Scan({{ range $i, $field := $returnTypeFieldsFlatten }}{{ if ne $i 0 }}, {{ end }}&r_.{{ $field }}{{ end }}); err != nil {
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
