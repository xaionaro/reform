package main

import (
	"text/template"
	"gopkg.in/reform.v1/parse"
)

type StructData struct {
	parse.StructInfo
	TableType       string
	ScopeType	string
	FilterType	string
	TableVar        string
	IsPrivateStruct bool
	QuerierVar      string
}

var (
	prologTemplate = template.Must(template.New("prolog").Parse(`
// generated with gopkg.in/reform.v1

import (
	"fmt"
	"reflect"
	"strings"

	"gopkg.in/reform.v1"
	"gopkg.in/reform.v1/parse"
)
`))

	structTemplate = template.Must(template.New("struct").Parse(`
type {{ .TableType }} struct {
	s parse.StructInfo
	z []interface{}
}

type {{ .ScopeType }} struct {
	{{ .Type }}
	order []string
}

type {{ .FilterType }} {{ .Type }}

// Schema returns a schema name in SQL database ("{{ .SQLSchema }}").
func (v *{{ .TableType }}) Schema() string {
	return v.s.SQLSchema
}

// Name returns a view or table name in SQL database ("{{ .SQLName }}").
func (v *{{ .TableType }}) Name() string {
	return v.s.SQLName
}

// Columns returns a new slice of column names for that view or table in SQL database.
func (v *{{ .TableType }}) Columns() []string {
	return {{ printf "%#v" .Columns }}
}

// NewStruct makes a new struct for that view or table.
func (v *{{ .TableType }}) NewStruct() reform.Struct {
	return new({{ .Type }})
}

{{- if .IsTable }}

// NewRecord makes a new record for that table.
func (v *{{ .TableType }}) NewRecord() reform.Record {
	return new({{ .Type }})
}

func (v *{{ .TableType }}) NewScope() *{{ .ScopeType }} {
	return &{{ .ScopeType }}{}
}

// PKColumnIndex returns an index of primary key column for that table in SQL database.
func (v *{{ .TableType }}) PKColumnIndex() uint {
	return uint(v.s.PKFieldIndex)
}

{{- end }}

// {{ .TableVar }} represents {{ .SQLName }} view or table in SQL database.
var {{ .TableVar }} = &{{ .TableType }} {
	s: {{ printf "%#v" .StructInfo }},
	z: new({{ .Type }}).Values(),
}

// String returns a string representation of this struct or record.
func (s {{ .Type }}) String() string {
	res := make([]string, {{ len .Fields }})
	{{- range $i, $f := .Fields }}
	res[{{ $i }}] = "{{ $f.Name }}: " + reform.Inspect(s.{{ $f.Name }}, true)
	{{- end }}
	return strings.Join(res, ", ")
}

// Values returns a slice of struct or record field values.
// Returned interface{} values are never untyped nils.
func (s *{{ .Type }}) Values() []interface{} {
	return []interface{}{ {{- range .Fields }}
		s.{{ .Name }}, {{- end }}
	}
}

// Pointers returns a slice of pointers to struct or record fields.
// Returned interface{} values are never untyped nils.
func (s *{{ .Type }}) Pointers() []interface{} {
	return []interface{}{ {{- range .Fields }}
		&s.{{ .Name }}, {{- end }}
	}
}

// View returns View object for that struct.
func (s *{{ .Type }}) View() reform.View {
	return {{ .TableVar }}
}

// Generate a scope for object
func (s *{{ .Type }}) Scope() *{{ .ScopeType }} {
	return &{{ .ScopeType }}{ {{ .Type }}: *s }
}

// Compiles SQL tail for defined order scope
// TODO: should be compiled via dialects
func (s *{{ .ScopeType }}) getOrderTail(db *reform.DB) (tail string, args []interface{}, err error) {
	var fieldName string
	var orderStringParts []string

	for idx,orderStr := range s.order {
		switch (idx%2) {
			case 0:
				fieldName       = orderStr
			case 1:
				orderDirection := orderStr

				orderStringParts = append(orderStringParts, fieldName+" "+orderDirection) // TODO: escape field name
		}
	}

	tail = strings.Join(orderStringParts, ", ")

	return
}

// Compiles SQL tail for defined filter
// TODO: should be compiled via dialects
func (s *{{ .ScopeType }}) getWhereTail(db *reform.DB, filter {{ .FilterType }}) (tail string, whereTailArgs []interface{}, err error) {
	var whereTailStringParts []string

	sample := {{ .Type }}(filter)

	v  := reflect.ValueOf(sample)
	vT := v.Type()

	numField := v.NumField()

	counter := 0
	for i := 0; i < numField; i++ {
		f  := v.Field(i)
		fT := f.Type()

		if f.Interface() == reflect.Zero(fT).Interface() {
			continue
		}

		s  := vT.Field(i)
		rN := s.Tag.Get("reform")

		counter++
		whereTailStringParts = append(whereTailStringParts, rN+" = "+db.Dialect.Placeholder(counter)) // TODO: escape field name
		whereTailArgs        = append(whereTailArgs, f.Interface())
	}

	tail = strings.Join(whereTailStringParts, " AND ")

	return
}

// Compiles SQL tail for defined order scope and filter
// TODO: should be compiled via dialects
func (s *{{ .ScopeType }}) compileTailUsingFilter(db *reform.DB, filter {{ .FilterType }} ) (tail string, args []interface{}, err error) {
	whereTailString, whereTailArgs, err := s.getWhereTail(db, filter)
	if err != nil {
		return
	}
	orderTailString, orderTailArgs, err := s.getOrderTail(db)
	if err != nil {
		return
	}

	args = append(whereTailArgs, orderTailArgs...)

	if len(whereTailString) > 0 {
		whereTailString = " WHERE "+whereTailString+" "
	}

	if len(orderTailString) > 0 {
		orderTailString = " ORDER BY "+orderTailString+" "
	}

	tail = whereTailString+orderTailString
	return

}

// parseQuerierArgs considers different ways of defning the tail (using scope properties or/and in_args)
func (s *{{ .ScopeType }}) parseQuerierArgs(db *reform.DB, in_args []interface{}) (tail string, args []interface{}, err error) {
	if len(in_args) > 0 {
		switch arg := in_args[0].(type) {
		case string:
			if len(s.order) > 0 {
				err = fmt.Errorf("This case is not implemented yet. You cannot use Order() and string tail argument in one request.")
				return
			}
			tail = arg
			args = in_args[1:]
		case {{ .Type }}:
			if len(args) > 1 {
				err = fmt.Errorf("Too many arguments.")
				return
			}
			tail, args, err = s.compileTailUsingFilter(db, {{ .FilterType }}(arg))
		case {{ .FilterType }}:
			if len(args) > 1 {
				err = fmt.Errorf("Too many arguments.")
				return
			}
			tail, args, err = s.compileTailUsingFilter(db, arg)
		default:
			err = fmt.Errorf("Invalid first element of \"args\". It should be a string or {{ .FilterType }}.")
		}
	}

	return
}

// Select is a wrapper for SelectRows() and NextRow(): it makes a query and collects the result into a slice
func (s *{{ .Type }}) Select(db *reform.DB, args ...interface{}) (result []{{.Type}}, err error) { return s.Scope().Select(db, args...) }
func (s *{{ .ScopeType }}) Select(db *reform.DB, args ...interface{}) (result []{{.Type}}, err error) {
	tail, args, err := s.parseQuerierArgs(db, args)
	if err != nil {
		return
	}

	rows, err := db.SelectRows({{ .TableVar }}, tail, args...)
	if err != nil {
		return
	}
	defer rows.Close()

	for {
		err := db.NextRow(s, rows)
		if err != nil {
			break
		}
		result = append(result, (*s).{{ .Type }})
	}

	return
}

// "First" a method to select and return only one record.
func (s *{{ .Type }}) First(db *reform.DB, args ...interface{}) (result {{.Type}}, err error) { return s.Scope().First(db, args...) }
func (s *{{ .ScopeType }}) First(db *reform.DB, args ...interface{}) (result {{.Type}}, err error) {
	tail, args, err := s.parseQuerierArgs(db, args)
	if err != nil {
		return
	}

	err = db.SelectOneTo(&result, tail, args...)

	return
}

// Create and Insert inserts new record to DB
func (s *{{ .Type }}) Create(db *reform.DB) (err error) { return s.Scope().Create(db) }
func (s *{{ .ScopeType }}) Create(db *reform.DB) (err error) {
	return db.Insert(s)
}
func (s *{{ .Type }}) Insert(db *reform.DB) (err error) { return s.Scope().Insert(db) }
func (s *{{ .ScopeType }}) Insert(db *reform.DB) (err error) {
	return db.Insert(s)
}

// Save inserts new record to DB is PK is zero and updates existing record if PK is not zero
func (s *{{ .Type }}) Save(db *reform.DB) (err error) { return s.Scope().Save(db) }
func (s *{{ .ScopeType }}) Save(db *reform.DB) (err error) {
	return db.Save(s)
}

// Update updates existing record in DB
func (s *{{ .Type }}) Update(db *reform.DB) (err error) { return s.Scope().Update(db) }
func (s *{{ .ScopeType }}) Update(db *reform.DB) (err error) {
	return db.Update(s)
}

// Delete deletes existing record in DB
func (s *{{ .Type }}) Delete(db *reform.DB) (err error) { return s.Scope().Delete(db) }
func (s *{{ .ScopeType }}) Delete(db *reform.DB) (err error) {
	return db.Delete(s)
}


// Sets order. Arguments should be passed by pairs column-{ASC,DESC}. For example Order("id", "ASC", "value" "DESC")
func (s *{{ .Type }}) Order(args ...string) (scope *{{ .ScopeType }}) { return s.Scope().Order(args...) }
func (s *{{ .ScopeType }}) Order(args ...string) (*{{ .ScopeType }}) {
	s.order = args
	return s
}

{{- if .IsTable }}

// Table returns Table object for that record.
func (s *{{ .Type }}) Table() reform.Table {
	return {{ .TableVar }}
}

// PKValue returns a value of primary key for that record.
// Returned interface{} value is never untyped nil.
func (s *{{ .Type }}) PKValue() interface{} {
	return s.{{ .PKField.Name }}
}

// PKPointer returns a pointer to primary key field for that record.
// Returned interface{} value is never untyped nil.
func (s *{{ .Type }}) PKPointer() interface{} {
	return &s.{{ .PKField.Name }}
}

// HasPK returns true if record has non-zero primary key set, false otherwise.
func (s *{{ .Type }}) HasPK() bool {
	return s.{{ .PKField.Name }} != {{ .TableVar }}.z[{{ .TableVar }}.s.PKFieldIndex]
}

// SetPK sets record primary key.
func (s *{{ .Type }}) SetPK(pk interface{}) {
	if i64, ok := pk.(int64); ok {
		s.{{ .PKField.Name }} = {{ .PKField.Type }}(i64)
	} else {
		s.{{ .PKField.Name }} = pk.({{ .PKField.Type }})
	}
}

{{- end }}

var (
	// check interfaces
	_ reform.View   = {{ .TableVar }}
	_ reform.Struct = new({{ .Type }})
{{- if .IsTable }}
	_ reform.Table  = {{ .TableVar }}
	_ reform.Record = new({{ .Type }})
{{- end }}
	_ fmt.Stringer   = new({{ .Type }})
{{- if .IsPrivateStruct }}

	// querier
	{{ .QuerierVar }} = {{ .Type }}{} // Should be read only
{{- end }}
)

`))

	initTemplate = template.Must(template.New("init").Parse(`
func init() {
	{{- range $i, $sd := . }}
	parse.AssertUpToDate(&{{ $sd.TableVar }}.s, new({{ $sd.Type }}))
	{{- end }}
}
`))
)
