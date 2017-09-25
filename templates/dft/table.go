package dft

import (
	"github.com/huangjunwen/JustSQL/render"
)

func init() {
	render.RegistBuiltinTemplate("table", render.DefaultTemplateSetName, `
{{/* =========================== */}}
{{/*          imports            */}}
{{/* =========================== */}}
{{- $ctx := imp "context" -}}
{{- $fmt := imp "fmt" -}}
{{- $sql := imp "database/sql" -}}
{{- $driver := imp "database/sql/driver" -}}
{{- $sqlx := imp "github.com/jmoiron/sqlx" -}}
{{- $strings := imp "strings" -}}

{{/* =========================== */}}
{{/*      global variables       */}}
{{/* =========================== */}}
{{- $tableName := .Table.Name -}}
{{- $structName := .Table.PascalName -}}
{{- $cols := .Table.Columns -}}
{{- $autoIncCol := .Table.AutoIncColumn -}}
{{- $primaryCols := .Table.PrimaryColumns -}}

{{- $structFieldNameList := stringList -}}
{{- $structFieldTypeList := stringList -}}
{{- $enumColList := columnList -}}
{{- $setColList := columnList -}}
{{- range $i, $col := $cols -}}
	{{- append $structFieldNameList $col.PascalName -}}
	{{- if $col.IsEnum -}}
		{{- append $structFieldTypeList (printf "%s%s" $structName $col.PascalName) -}}
		{{- append $enumColList $col -}}
	{{- else if $col.IsSet -}}
		{{- append $structFieldTypeList (printf "%s%s" $structName $col.PascalName) -}}
		{{- append $setColList $col -}}
	{{- else -}}
		{{- append $structFieldTypeList (typeName $col.Type) -}}
	{{- end -}}
{{- end -}}
{{- $structFieldNames := $structFieldNameList.Strings -}}
{{- $structFieldTypes := $structFieldTypeList.Strings -}}
{{- $enumCols := $enumColList.Cols -}}
{{- $setCols := $setColList.Cols -}}

{{/* =========================== */}}
{{/*          enums              */}}
{{/* =========================== */}}

{{ range $i, $col := $enumCols }}
	{{/* =========================== */}}
	{{/*        enum variables       */}}
	{{/* =========================== */}}
	{{- $enumName := printf "%s%s" $structName $col.PascalName -}}
	{{- $enumItemList := stringList -}}
	{{- range $i, $item := $col.Elems -}}
		{{- if eq $item "" -}}
			{{- append $enumItemList (printf "%sEmpty_" $enumName) -}}
		{{- else -}}
			{{- append $enumItemList (printf "%s%s" $enumName (pascal $item)) }}
		{{- end -}}
	{{- end -}}
	{{- $enumItems := $enumItemList.Strings -}}

	{{/* =========================== */}}
	{{/*        enum code            */}}
	{{/* =========================== */}}

// Enum {{ $enumName }}.
type {{ $enumName }} int

// Enum {{ $enumName }} items.
const (
	// NULL value.
	{{ $enumName }}NULL_ = {{ $enumName }}(0)
	{{- range $i, $item := $enumItems }}
	// {{ printf "%+q" (index $col.Elems $i) }}
	{{ $item }} = {{ $enumName }}({{ $i }} + 1)
	{{- end }}
)

func (e {{ $enumName }}) String() string {
	switch e {
	{{- range $i, $item := $enumItems }}
	case {{ $item }}:
		return {{ printf "%+q" (index $col.Elems $i) }}
	{{- end }}
	}
	return ""
}

func (e {{ $enumName }}) Valid() bool {
	return int(e) > 0 && int(e) <= {{ printf "%d" (len $col.Elems) }}
}

// Scan implements database/sql.Scanner interface.
func (e *{{ $enumName }}) Scan(value interface{}) error {
	if value == nil {
		*e = {{ $enumName }}NULL_
		return nil
	}

	switch s := string(value.([]byte)); s {
	{{- range $i, $item := $enumItems }}
	case {{ printf "%+q" (index $col.Elems $i) }}:
		*e = {{ $item }}
	{{- end }}
	default:
		return {{ $fmt }}.Errorf("Unexpected value for {{ $enumName }}: %+q", s)
	}

	return nil
}

// Value implements database/sql/driver.Valuer interface.
func (e {{ $enumName }}) Value() ({{ $driver }}.Value, error) {
	if !e.Valid() {
		return nil, nil
	}
	return e.String(), nil
}

{{ end }}

{{/* =========================== */}}
{{/*          sets               */}}
{{/* =========================== */}}

{{ range $i, $col := $setCols -}}

	{{/* =========================== */}}
	{{/*        set  variables       */}}
	{{/* =========================== */}}
	{{- $setName := printf "%s%s" $structName $col.PascalName -}}
	{{- $setItemList := stringList -}}
	{{- range $i, $item := $col.Elems -}}
		{{- if eq $item "" -}}
			{{- append $setItemList (printf "%sEmpty_" $setName) -}}
		{{- else -}}
			{{- append $setItemList (printf "%s%s" $setName (pascal $item)) }}
		{{- end -}}
	{{- end -}}
	{{- $setItems := $setItemList.Strings -}}

	{{/* =========================== */}}
	{{/*        set  code            */}}
	{{/* =========================== */}}

// Set {{ $setName }}.
type {{ $setName }} struct {
	val uint64 // Up to 64 distinct members. See https://dev.mysql.com/doc/refman/5.7/en/set.html
	valid bool // NULL if valid is false.
}

// Set {{ $setName }} items.
const (
	{{- range $i, $item := $setItems }}
	// {{ printf "%+q" (index $col.Elems $i) }}
	{{ $item }} = uint64(1<<{{ $i }})
	{{- end }}
)

func New{{ $setName }}(items ...uint64) {{ $setName }} {
	var val uint64 = 0
	for _, item := range items {
		if item > 0 && (item & (item - 1)) == 0 && item <= (1 << ({{ len $col.Elems }} - 1)) {
			val |= item 
		}
	}
	return {{ $setName }}{
		val: val,
		valid: true,
	}
}

func (s {{ $setName }}) String() string {
	parts := make([]string, 0)
	{{- range $i, $item:= $setItems }}
	if s.val & {{ $item }} != 0 {
		parts = append(parts, {{ printf "%+q" (index $col.Elems $i) }})
	}
	{{- end }}
	return strings.Join(parts, ",")
}

func (s {{ $setName }}) Valid() bool {
	return s.valid
}

// Scan implements database/sql.Scanner interface.
func (s *{{ $setName }}) Scan(value interface{}) error {
	if value == nil {
		s.val = 0
		s.valid = false
		return nil
	}

	var val uint64 = 0
	for _, part := range {{ $strings }}.Split(string(value.([]byte)), ",") {
		switch part {
	{{- range $i, $elem := $col.Elems }}
		case {{ printf "%+q" $elem }}:
			val |= {{ index $setItems $i }}
	{{- end }}
		default:
			return {{ $fmt }}.Errorf("Unexpected value for {{ $setName }}: %+q", part)
		}
	}

	s.val = val
	s.valid = true
	return nil
}

// Value implements database/sql/driver.Valuer interface.
func (s {{ $setName }}) Value() (driver.Value, error) {
	if !s.Valid() {
		return nil, nil
	}
	return s.String(), nil
}

{{- end }}

{{/* =========================== */}}
{{/*          main struct        */}}
{{/* =========================== */}}

// {{ $structName }} represents an entry of table "{{ $tableName }}".
type {{ $structName }} struct {
{{- range $i, $col := $cols }}
	{{ index $structFieldNames $i }} {{ index $structFieldTypes $i }} `+"`db:\"{{ $col.Name }}\"`"+` // {{ $col.Name }}
{{- end }}
}

{{/* =========================== */}}
{{/*     insert/update/delete    */}}
{{/* =========================== */}}

// Insert insert an entry of {{ $tableName }} into database.
func (entry_ *{{ $structName }}) Insert(ctx_ {{ $ctx }}.Context, db_ DBer) error {

	sql_ := {{ $sqlx }}.Rebind(BindType, "INSERT INTO {{ $tableName }} " +
		"({{ join (columnNames $cols) ", " }}) " +
		"VALUES ({{ join (dup "?" (len $cols))  ", " }})")

	{{ if notNil $autoIncCol }}res_{{ else }}_{{ end }}, err_ := db_.ExecContext(ctx_, sql_{{ range $i, $field := $structFieldNames }}, entry_.{{ $field }}{{ end }})
	if err_ != nil {
		return err_
	}

	{{ if notNil $autoIncCol -}}
	lastInsertId_, err_ := res_.LastInsertId()
	if err_ != nil {
		return err_
	}

	return SaveCoerceFromInt64(lastInsertId_, &entry_.{{ $autoIncCol.PascalName }})
	{{ else -}}
	return nil
	{{ end -}}
}

{{ if ne (len $primaryCols) 0 -}}

func (entry_ *{{ $structName }}) Update(ctx_ {{ $ctx }}.Context, db_ DBer) (int64, error) {

	sql_ := {{ $sqlx }}.Rebind(BindType, "UPDATE {{ $tableName }} " + 
		"SET {{ range $i, $col := $cols }}{{ if ne $i 0 }}, {{ end }}{{ $col.Name }}=?{{ end }} " +
		"WHERE {{ range $i, $col := $primaryCols }}{{ if ne $i 0 }}AND {{ end }}{{ $col.Name }}=? {{ end }}")

	r_, err_ := db_.ExecContext(ctx_, sql_{{ range $i, $col := $cols }}, entry_.{{ $col.PascalName }}{{ end }}{{ range $i, $col := $primaryCols }}, entry_.{{ $col.PascalName }}{{ end }})
	if err_ != nil {
		return 0, err_
	}

	return r_.RowsAffected()

}

func (entry_ *{{ $structName }}) Delete(ctx_ {{ $ctx }}.Context, db_ DBer) (int64, error) {

	sql_ := {{ $sqlx }}.Rebind(BindType, "DELETE FROM {{ $tableName }} " +
		"WHERE {{ range $i, $col := $primaryCols }}{{ if ne $i 0 }}AND {{ end }}{{ $col.Name }}=? {{ end }}")

	r_, err_ := db_.ExecContext(ctx_, sql_{{ range $i, $col := $primaryCols }}, entry_.{{ $col.PascalName }}{{ end }})
	if err_ != nil {
		return 0, err_
	}

	return r_.RowsAffected()
}

{{ end }}

{{/* =========================== */}}
{{/*         foreign key         */}}
{{/* =========================== */}}

{{- range $i, $fk := .Table.ForeignKeys }}
	{{- $refIndex := $fk.RefIndex }}
	{{- if $refIndex.Unique }}
	{{/* =========================== */}}
	{{/*    foreign key variables    */}}
	{{/* =========================== */}}
		{{- $fkColumns := $fk.Columns }}
		{{- $refTable := $fk.RefTable }}
		{{- $refColumns := $fk.RefColumns }}

	{{/* =========================== */}}
	{{/*       foreign key code      */}}
	{{/* =========================== */}}

// {{ $refTable.PascalName }} return {{ printf "%q" $refTable.Name }} entry by foreign key "{{ printf "%s.%s" $fk.Table.Name $fk.Name }}".
func (entry_ *{{ $structName }}) {{ $refTable.PascalName }}(ctx_ {{ $ctx }}.Context, db_ DBer) (*{{ $refTable.PascalName }}, error) {
	return {{ $refTable.PascalName }}By{{ $refIndex.PascalName }}(ctx_, db_
		{{- range $j, $fkCol := $fkColumns -}}
			{{- $expr := printf "entry_.%s" $fkCol.PascalName -}}
			{{- $fkColType := typeName $fkCol -}}
			{{- $refColType := typeName (index $refColumns $j) -}}
		, {{ cast $expr $fkColType $refColType }}
		{{- end -}})
}
	{{- end }}
{{- end }}


{{/* =========================== */}}
{{/*          unique indices     */}}
{{/* =========================== */}}

{{- range $i, $index := .Table.Indices }}
	{{- if $index.Unique }}
	{{/* =========================== */}}
	{{/*   unique index variables    */}}
	{{/* =========================== */}}
		{{- $indexCols := $index.Columns }}
		{{- $argNameList := stringList }}
		{{- $argTypeList := stringList }}
		{{- range $j, $col := $indexCols }}
			{{- append $argNameList (camel $col.Name) }}
			{{- append $argTypeList (typeName $col.Type) }}
		{{- end }}
		{{- $argNames := $argNameList.Strings }}
		{{- $argTypes := $argTypeList.Strings }}

	{{/* =========================== */}}
	{{/*      unique index code      */}}
	{{/* =========================== */}}

// {{ $structName }}By{{ $index.PascalName }} query {{ printf "%+q" $tableName }} table by {{ if $index.Primary }}primary key{{ else }}unique key {{ printf "%+q" $index.Name }}{{ end }}.
// Return nil if error occurred or there is not row found.
func {{ $structName }}By{{ $index.PascalName }}(ctx_ {{ $ctx }}.Context, db_ DBer{{ range $j, $argName := $argNames }}, {{ $argName }} {{ index $argTypes $j }}{{ end }}) (*{{ $structName }}, error) {

	sql_ := {{ $sqlx }}.Rebind(BindType, "SELECT {{ join (columnNames $cols) ", " }} " +
		"FROM {{ $tableName }} " +
		"WHERE {{ range $j, $col := $indexCols }}{{ if ne $j 0 }}AND {{ end }}{{ $col.Name }}=? {{ end }}")

	row_ := db_.QueryRowContext(ctx_, sql_{{ range $j, $argName := $argNames }}, {{ $argName }}{{ end }})

	entry_ := new({{ $structName }})
	if err_ := row_.Scan({{ range $j, $field := $structFieldNames }}{{ if ne $j 0 }}, {{ end }}&entry_.{{ $field }}{{ end }}); err_ != nil {
		if err_ == {{ $sql }}.ErrNoRows {
			return nil, nil
		}
		return nil, err_
	}

	return entry_, nil
}

	{{- end }}
{{- end }}

`)

}
