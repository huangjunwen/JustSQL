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
{{- $funcName := .Annot.FuncName -}}
{{- $rfs := .Stmt.ResultFields -}}
{{- $retName := printf "%sResult" .Annot.FuncName -}}
{{- $retFieldNames := uniqueStrings (fn "pascal") "NoNameField" -}}
{{- $retFieldNamesFlatten := strings -}}
{{- $retStructFieldNames := strings -}}
{{- $retStructFieldTypes := strings -}}
{{- $hasInBinding := gt (len .Annot.InBindings) 0 -}}
{{- $returnStyle := .Annot.ReturnStyle -}}

{{/* =========================== */}}
{{/*          return type        */}}
{{/* =========================== */}}

// {{ $retName }} is the result type of {{ $funcName }}.
type {{ $retName }} struct {
{{- range $i, $rf := $rfs -}}
	{{/* whether this result field is in a normal table wildcard expansion */}}
	{{- $wildcardTableRefName := $.OriginStmt.FieldList.WildcardTableRefName $i -}}
	{{- $wildcardTableIsNormal := and ($.OriginStmt.TableRefs.IsNormalTable $wildcardTableRefName) (not ($.OriginStmt.TableRefs.IsDerivedTable $wildcardTableRefName)) -}}
	{{- $wildcardOffset := $.OriginStmt.FieldList.WildcardOffset $i -}}

	{{- if $wildcardTableIsNormal }}
		{{- if eq $wildcardOffset 0 }}
			{{- $retFieldNames.Add $wildcardTableRefName }}
			{{- $retStructFieldNames.Add $retFieldNames.Last }}
			{{- $retStructFieldTypes.Add $rf.Table.PascalName }}
			{{ $retStructFieldNames.Last }} *{{ $retStructFieldTypes.Last }} // {{ $wildcardTableRefName }}.*
		{{- end }}
		{{- $retFieldNamesFlatten.Add (printf "%s.%s" $retFieldNames.Last $rf.Column.PascalName) }}
	{{- else }}
		{{- $retFieldNames.Add $rf.Name -}}
		{{- if and (or $rf.IsEnum $rf.IsSet) (notNil $rf.Table) }}
			{{ $retFieldNames.Last }} {{ $rf.Table.PascalName }}{{ $rf.Column.PascalName }}
		{{- else }}
			{{ $retFieldNames.Last }} {{ typeName $rf.Type }}
		{{- end }}
		{{- $retFieldNamesFlatten.Add $retFieldNames.Last }}
	{{- end }}

{{- end }}
}

func new{{ $retName }}() *{{ $retName }} {
	return &{{ $retName }}{
{{ range $i, $name := $retStructFieldNames -}}
		{{ $name }}: new({{ index $retStructFieldTypes $i }}),
{{ end -}}
	}
}

{{/* =========================== */}}
{{/*        sql template         */}}
{{/* =========================== */}}

var _{{ $funcName }}SQLTmpl = template.Must(template.New({{ printf "%q" $funcName }}).Parse("" +
{{- range $line := splitLines .Annot.Text }}
	"{{ printf "%s" $line }} " +
{{- end }}""))

{{/* =========================== */}}
{{/*        main function        */}}
{{/* =========================== */}}
// {{ $funcName }} is generated from:
//
{{- range $line := splitLines .OriginStmt.SelectStmt.Text }}
{{- if ne (len $line) 0 }}
//    {{ printf "%s" $line }}
{{- end }}
{{- end }}
//
func {{ $funcName }}(ctx_ {{ $ctx }}.Context, db_ DBer{{ range $arg := .Annot.Args }}, {{ $arg.Name }} {{ typeName $arg.Type }} {{ end }}) ({{ if eq $returnStyle "one" }}*{{ $retName }}{{ else if eq $returnStyle "many" }}[]*{{ $retName }}{{ end }}, error) {

	// - Dot object for template and query parameter.
	dot_ := map[string]interface{}{
{{- range $arg := .Annot.Args }}
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
	ret_ := new{{ $retName }}()
	if err_ := row_.Scan({{ range $i, $field := $retFieldNamesFlatten }}{{ if ne $i 0 }}, {{ end }}&ret_.{{ $field }}{{ end }}); err != nil {
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
	ret_ := make([]*{{ $retName }}, 0)
	for rows_.Rows.Next() {
		r_ := new{{ $retName }}()
		if err_ := rows_.Rows.Scan({{ range $i, $field := $retFieldNamesFlatten }}{{ if ne $i 0 }}, {{ end }}&r_.{{ $field }}{{ end }}); err != nil {
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
