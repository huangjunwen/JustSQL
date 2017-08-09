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
{{/*          variables          */}}
{{/* =========================== */}}
{{- $funcName := .Annot.FuncName -}}
{{- $rfs := .Stmt.ResultFields -}}
{{- $retName := printf "%sResult" .Annot.FuncName -}}
{{- $hasInBinding := gt (len .Annot.InBindings) 0 -}}
{{- $returnStyle := .Annot.ReturnStyle -}}

{{- $retFieldNameList := uniqueStringList (fn "pascal") "NoNameField" -}}
{{- $retFieldTypeList := stringList -}}
{{- $retStructFieldNameList := stringList -}}
{{- $retStructFieldTypeList := stringList -}}
{{- $retFieldNameFlattenList := stringList -}}
{{- range $i, $rf := $rfs -}}
	{{/* $wildcardTableRefName != "" if the result field is inside a wildcard */}}
	{{- $wildcardTableRefName := $.OriginStmt.FieldList.WildcardTableRefName $i -}}
	{{- $wildcardTableIsNormal := and ($.OriginStmt.TableRefs.IsNormalTable $wildcardTableRefName) (not ($.OriginStmt.TableRefs.IsDerivedTable $wildcardTableRefName)) -}}
	{{/* BUG: the table should be in default database */}}
	{{- $wildcardOffset := $.OriginStmt.FieldList.WildcardOffset $i -}}

	{{- if and $wildcardTableIsNormal (eq $rf.DBName (dbname)) -}}
		{{- if eq $wildcardOffset 0 -}}
			{{- append $retFieldNameList $wildcardTableRefName -}}
			{{- append $retFieldTypeList (printf "*%s" $rf.Table.PascalName) -}}
			{{- append $retStructFieldNameList (last $retFieldNameList) -}}
			{{- append $retStructFieldTypeList $rf.Table.PascalName -}}
		{{- end -}}
		{{- append $retFieldNameFlattenList (printf "%s.%s" (last $retFieldNameList) $rf.Column.PascalName) }}
	{{- else -}}
		{{- append $retFieldNameList $rf.Name -}}
		{{- if and (or $rf.IsEnum $rf.IsSet) (notNil $rf.Table) }}
			{{- append $retFieldTypeList (printf "%s%s" $rf.Table.PascalName $rf.Column.PascalName) -}}
		{{- else }}
			{{- append $retFieldTypeList (typeName $rf.Type) -}}
		{{- end }}
		{{- append $retFieldNameFlattenList (last $retFieldNameList) }}
	{{- end -}}
{{- end -}}
{{- $retFieldNames := $retFieldNameList.Strings -}}
{{- $retFieldTypes := $retFieldTypeList.Strings -}}
{{- $retStructFieldNames := $retStructFieldNameList.Strings -}}
{{- $retStructFieldTypes := $retStructFieldTypeList.Strings -}}
{{- $retFieldNamesFlatten := $retFieldNameFlattenList.Strings -}}

{{/* =========================== */}}
{{/*          return type        */}}
{{/* =========================== */}}

// {{ $retName }} is the return type of {{ $funcName }}.
type {{ $retName }} struct {
{{- range $i, $name := $retFieldNames }}
	{{ $name }} {{ index $retFieldTypes $i }}
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
{{- range $line := split .Annot.Text "\n" }}
	"{{ printf "%s" $line }} " +
{{- end }}""))

{{/* =========================== */}}
{{/*        main function        */}}
{{/* =========================== */}}
// {{ $funcName }} is generated from:
//
{{- range $line := split .OriginStmt.SelectStmt.Text "\n" }}
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
	if err_ := row_.Scan({{ range $i, $name := $retFieldNamesFlatten }}{{ if ne $i 0 }}, {{ end }}&ret_.{{ $name }}{{ end }}); err != nil {
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
		if err_ := rows_.Rows.Scan({{ range $i, $name := $retFieldNamesFlatten }}{{ if ne $i 0 }}, {{ end }}&r_.{{ $name }}{{ end }}); err != nil {
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
