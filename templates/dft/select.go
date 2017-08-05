package dft

import (
	"github.com/huangjunwen/JustSQL/render"
)

func init() {
	render.RegistBuiltinTemplate("select", render.DefaultTemplateName, `
{{/* =========================== */}}
{{/*          imports            */}}
{{/* =========================== */}}
{{- $ctx := imp "context" -}}
{{- $sqlx := imp "github.com/jmoiron/sqlx" -}}
{{- $sql := imp "database/sql" -}}
{{- $template := imp "text/template" -}}
{{- $bytes := imp "bytes" -}}

{{/* =========================== */}}
{{/*          declares           */}}
{{/* =========================== */}}
{{- $funcName := .Func.Name -}}
{{- $rfs := .Stmt.ResultFields -}}
{{- $returnType := printf "%sResult" .Func.Name -}}
{{- $returnTypeFields := uniqueStrings (fn "pascal") "NoNameField" -}}
{{- $returnTypeFieldsFlatten := strings -}}
{{- $hasInBinding := .Func.HasInBinding -}}
{{- $returnStyle := .Func.ReturnStyle -}}

{{/* =========================== */}}
{{/*          return type        */}}
{{/* =========================== */}}

// {{ $returnType }} is the result type of {{ $funcName }}.
type {{ $returnType }} struct {
{{- range $i, $rf := $rfs -}}
	{{/* whether this result field is in a normal table wildcard expansion */}}
	{{- $wildcardTableRefName := $.OriginStmt.FieldList.WildcardTableRefName $i -}}
	{{- $wildcardTableIsNormal := and ($.OriginStmt.TableRefs.IsNormalTable $wildcardTableRefName) (not ($.OriginStmt.TableRefs.IsDerivedTable $wildcardTableRefName)) -}}
	{{- $wildcardOffset := $.OriginStmt.FieldList.WildcardOffset $i -}}

	{{- if $wildcardTableIsNormal }}
		{{- if eq $wildcardOffset 0 }}
			{{- $returnTypeFields.Add $wildcardTableRefName }}
			{{ $returnTypeFields.Last }} {{ $rf.Table.PascalName }} // {{ $wildcardTableRefName }}.*
		{{- end }}
		{{- $returnTypeFieldsFlatten.Add (printf "%s.%s" $returnTypeFields.Last $rf.Column.PascalName) }}
	{{- else }}
		{{- $returnTypeFields.Add $rf.Name -}}
		{{- if and (or $rf.IsEnum $rf.IsSet) (notNil $rf.Table) }}
			{{ $returnTypeFields.Last }} {{ $rf.Table.PascalName }}{{ $rf.Column.PascalName }}
		{{- else }}
			{{ $returnTypeFields.Last }} {{ $rf.AdaptType }}
		{{- end }}
		{{- $returnTypeFieldsFlatten.Add $returnTypeFields.Last }}
	{{- end }}

{{- end }}
}

{{/* =========================== */}}
{{/*    sql template type        */}}
{{/* =========================== */}}

var _{{ $funcName }}SQLTmpl = template.Must(template.New({{ printf "%q" $funcName }}).Parse("" +
{{- range $line := splitLines .Func.Query }}
	"{{ printf "%s" $line }} " +
{{- end }}""))

{{/* =========================== */}}
{{/*        main function        */}}
{{/* =========================== */}}
// {{ $funcName }} is generated from:
//
{{- range $line := splitLines .Func.SrcQuery }}
{{- if ne (len $line) 0 }}
//    {{ printf "%s" $line }}
{{- end }}
{{- end }}
func {{ $funcName }}(ctx_ {{ $ctx }}.Context, db_ DBer{{ range $arg := .Func.Args }}, {{ $arg.Name }} {{ $arg.AdaptType }} {{ end }}) ({{ if eq $returnStyle "one" }}*{{ $returnType }}{{ else if eq $returnStyle "many" }}[]*{{ $returnType }}{{ end }}, error) {

	// - Dot object for template and query parameter.
	dot_ := map[string]interface{}{
{{- range $arg := .Func.Args }}
		{{ printf "%q" $arg.Name }}: {{ $arg.Name }},
{{- end }}
	}

	// - Render from template.
	buf_ := new(bytes.Buffer)
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
	query_, args_, err_ := {{ $sqlx }}.In(query_, args_...)
	if err_ != nil {
		return nil, err_
	}
{{- end }}

	// - Rebind.
	query_ = {{ $sqlx }}.Rebind(BindType, query_)

{{ if eq $returnStyle "one" -}}
	// - Query.
	row_, err_ := db_.QueryRowContext(ctx_, query_, args_...)
	if err_ != nil {
		return nil, err_
	}

	// - Scan.
	ret_ := new({{ $returnType }})
	if err_ := row_.Scan({{ range $i, $field := $returnTypeFieldsFlatten }}{{ if ne $i 0 }}, {{ end }}&ret_.{{ $field }}{{ end }}); err != nil {
		return nil, err
	}

	return ret_, nil
{{ else if eq $returnStyle "many" -}}
	// - Query.
	rows_, err_ := db_.QueryContext(ctx_, query_, args_...)
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
`)

}
