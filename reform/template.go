package main

import (
	"github.com/xaionaro/reform"
	"text/template"
)

// StructData represents struct info for XXX_reform.go file generation.
type StructData struct {
	reform.StructInfo
	LogType             string
	TableType           string
	LogTableType        string
	ScopeType           string
	FilterType          string
	FilterPublicType    string
	FilterShorthandType string
	TableVar            string
	LogTableVar         string
	IsPrivateStruct     bool
	QuerierVar          string
	ImitateGorm         bool
	SkipMethodOrder     bool
}

var (
	prologTemplate = template.Must(template.New("prolog").Parse(`
// Generated with gopkg.in/reform.v1. Do not edit by hand.

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/xaionaro/reform"
)
`))

	structTemplate = template.Must(template.New("struct").Parse(`
type {{ .ScopeType }} struct {
	item *{{ .Type }}

	db           reform.ReformDBTX
	where        [][]interface{}
	order        []string
	groupBy      []string
	limit        int
	tableQuery   *string
	fieldsFilter []string
	appendTail   string

	loggingEnabled  bool
	loggingAuthor  *string
	loggingComment  string
}

{{- if .IsPrivateStruct }}
type {{ .FilterPublicType }} {{ .Type }}
{{- end }}

{{- if .IsPrivateStruct }}
type {{ .FilterShorthandType }} {{ .Type }}
{{- end }}
type {{ .FilterType }} {{ .Type }}

type {{ .LogType }} struct {
	{{ .Type }}
	LogAuthor	*string
	LogAction	 string
	LogDate		 time.Time
	LogComment	 string
}

// Schema returns a schema name in SQL database ("{{ .SQLSchema }}").
type {{ .TableType }} struct {
	s reform.StructInfo
	z []interface{}
}

func (v {{ .TableType }}) Schema() string {
	return v.s.SQLSchema
}

// Name returns a view or table name in SQL database ("{{ .SQLName }}").
func (v {{ .TableType }}) Name() string {
	return v.s.SQLName
}

// Columns returns a new slice of column names for that view or table in SQL database.
func (v {{ .TableType }}) Columns() []string {
	return {{ printf "%#v" .Columns }}
}

// NewStruct makes a new struct for that view or table.
func (v {{ .TableType }}) NewStruct() reform.Struct {
	return new({{ .Type }})
}

{{- if .IsTable }}

// NewRecord makes a new record for that table.
func (v *{{ .TableType }}) NewRecord() reform.Record {
	return new({{ .Type }})
}

func (v *{{ .TableType }}) NewScope() *{{ .ScopeType }} {
	return &{{ .ScopeType }}{ item: &{{ .Type }}{} }
}

// PKColumnIndex returns an index of primary key column for that table in SQL database.
func (v *{{ .TableType }}) PKColumnIndex() uint {
	return uint(v.s.PKFieldIndex)
}

func (v {{ .TableType }}) CreateTableIfNotExists(db *reform.DB) (bool, error) {
	if db == nil {
		db = defaultDB_{{ .Type }}
	}
	return db.CreateTableIfNotExists(v.s)
}

{{- end }}

func (v {{ .TableType }}) StructInfo() reform.StructInfo {
	return v.s
}

// {{ .TableVar }} represents {{ .SQLName }} view or table in SQL database.
var {{ .TableVar }} = &{{ .TableType }} {
	s: {{ printf "%#v" .StructInfo }},
	z: new({{ .Type }}).Values(),
}

type {{ .LogTableType }} struct {
	s reform.StructInfo
	z []interface{}
}

func (v *{{ .LogTableType }}) Schema() string {
	return v.s.SQLSchema
}

func (v *{{ .LogTableType }}) Name() string {
	return v.s.SQLName
}

func (v *{{ .LogTableType }}) Columns() []string {
	return {{ printf "%#v" .ToLog.Columns }}
}

func (v *{{ .LogTableType }}) NewStruct() reform.Struct {
	return new({{ .Type }})
}

{{- if .IsTable }}

func (v *{{ .LogTableType }}) NewRecord() reform.Record {
	return new({{ .Type }})
}

func (v *{{ .LogTableType }}) NewScope() *{{ .ScopeType }} {
	return &{{ .ScopeType }}{ item: &{{ .Type }}{} }
}

func (v *{{ .LogTableType }}) PKColumnIndex() uint {
	return uint(v.s.PKFieldIndex)
}

{{- end }}

var {{ .LogTableVar }} = &{{ .LogTableType }} {
	s: {{ printf "%#v" .StructInfo.ToLog.UnPointer }},
	z: new({{ .LogType }}).Values(),
}

func (s {{ .TableType }}) ColumnNameByFieldName(fieldName string) string {
	switch (fieldName) {
	{{- range $i, $f := .Fields }}
	case "{{ $f.Name }}": return "{{ $f.Column }}"
	{{- end }}
	}
	return ""
}

func (s {{ .LogTableType }}) ColumnNameByFieldName(fieldName string) string {
	switch (fieldName) {
	{{- range $i, $f := .Fields }}
	case "{{ $f.Name }}": return "{{ $f.Column }}"
	{{- end }}
	case "LogAuthor": return "log_author"
	case "LogAction": return "log_action"
	case "LogDate": return "log_date"
	case "LogComment": return "log_comment"
	}
	return ""
}

func (s *{{ .Type }}) FieldPointersByNames(fieldNames []string) (fieldPointers []interface{}) {
	if len(fieldNames) == 0 {
		return s.Pointers()
	}

	for _, fieldName := range fieldNames {
		fieldPointer := s.FieldPointerByName(fieldName)
		if fieldPointer == nil {
			panic("Invalid field name:"+ fieldName)
		}
		fieldPointers = append(fieldPointers, fieldPointer)
	}

	return
}

func (s *{{ .LogType }}) FieldPointersByNames(fieldNames []string) (fieldPointers []interface{}) {
	if len(fieldNames) == 0 {
		return s.Pointers()
	}

	for _, fieldName := range fieldNames {
		fieldPointer := s.FieldPointerByName(fieldName)
		if fieldPointer == nil {
			panic("Invalid field name:"+ fieldName)
		}
		fieldPointers = append(fieldPointers, fieldPointer)
	}

	return
}

func (s *{{ .Type }}) FieldPointerByName(fieldName string) interface{} {
	switch (fieldName) {
	{{- range $i, $f := .Fields }}
	case "{{ $f.Name }}": return &s.{{ $f.FullName }}
	{{- end }}
	}

	return nil
}

func (s *{{ .LogType }}) FieldPointerByName(fieldName string) interface{} {
	switch (fieldName) {
	{{- range $i, $f := .Fields }}
	case "{{ $f.Name }}": return &s.{{ $f.FullName }}
	{{- end }}
	case "LogAuthor": return &s.LogAuthor
	case "LogAction": return &s.LogAction
	case "LogDate": return &s.LogDate
	case "LogComment": return &s.LogComment
	}

	return nil
}

// String returns a string representation of this struct or record.
func (s {{ .Type }}) String() string {
	res := make([]string, {{ len .Fields }})
	{{- range $i, $f := .Fields }}
	res[{{ $i }}] = "{{ $f.Name }}: " + reform.Inspect(s.{{ $f.FullName }}, true)
	{{- end }}
	return strings.Join(res, ", ")
}
func (s {{ .LogType }}) String() string {
	res := make([]string, {{ len .ToLog.Fields }})
	{{- range $i, $f := .ToLog.Fields }}
	res[{{ $i }}] = "{{ $f.Name }}: " + reform.Inspect(s.{{ $f.FullName }}, true)
	{{- end }}
	return strings.Join(res, ", ")
}

// Values returns a slice of struct or record field values.
// Returned interface{} values are never untyped nils.
func (s *{{ .Type }}) Values() []interface{} {
	return []interface{}{ {{- range .Fields }}
		s.{{ .FullName }}, {{- end }}
	}
}
func (s *{{ .LogType }}) Values() []interface{} {
	return append(s.{{ .Type }}.Values(), []interface{}{
		s.LogAuthor,
		s.LogAction,
		s.LogDate,
		s.LogComment,
	}...)
}

// Pointers returns a slice of pointers to struct or record fields.
// Returned interface{} values are never untyped nils.
func (s *{{ .Type }}) Pointers() []interface{} {
	return []interface{}{ {{- range .Fields }}
		&s.{{ .FullName }}, {{- end }}
	}
}
func (s *{{ .LogType }}) Pointers() []interface{} {
	return append(s.{{.Type}}.Pointers(), []interface{}{
		&s.LogAuthor,
		&s.LogAction,
		&s.LogDate,
		&s.LogComment,
	}...)
}

// View returns View object for that struct.
func (s {{ .Type }}) View() reform.View {
	return {{ .TableVar }}
}
func (s {{ .Type }}Scope) View() reform.View {
	return s.item.View()
}
func (s {{ .LogType }}) View() reform.View {
	return {{ .LogTableVar }}
}

// Generate a scope for object
func (s {{ .Type }}) Scope() *{{ .ScopeType }} {
	return &{{ .ScopeType }}{ item: &s, db: defaultDB_{{ .Type }} }
}
func (s *{{ .Type }}) PtrScope() *{{ .ScopeType }} {
	return &{{ .ScopeType }}{ item: s, db: defaultDB_{{ .Type }} }
}

// Sets DB to do queries
func (s {{ .Type }}) DB(db reform.ReformDBTX) (scope *{{ .ScopeType }}) { return s.Scope().DB(db) }
func (s *{{ .ScopeType }}) DB(db reform.ReformDBTX) *{{ .ScopeType }} {
	if db != nil {
		s.db = db
	}
	afterDBer, ok := interface{}(s).(reform.AfterDBer)
	if ok {
		afterDBer.AfterDB()
	}
	return s
}

// Gets DB
func (s {{ .Type }}) Get{{ if eq .ImitateGorm true }}Reform{{ end }}DB() (db *reform.DB) { return s.Scope().Get{{ if eq .ImitateGorm true }}Reform{{ end }}DB() }
func (s {{ .ScopeType }}) Get{{ if eq .ImitateGorm true }}Reform{{ end }}DB() *reform.DB {
	return s.db.(*reform.DB)
}

func (s {{ .Type }}) StartTransaction() (*reform.TX, error) { return s.Scope().StartTransaction() }
func (s {{ .ScopeType }}) StartTransaction() (*reform.TX, error) {
	return s.db.(*reform.DB).Begin()
}

// Sets default DB (to do not call the scope.DB() method every time)
func (s *{{ .Type }}) SetDefaultDB(db *reform.DB) (err error) {
	defaultDB_{{ .Type }} = db
	return nil
}


// Compiles SQL tail for defined limit scope
// TODO: should be compiled via dialects
func (s *{{ .ScopeType }}) getLimitTail() (tail string, args []interface{}, err error) {
	if s.limit <= 0 {
		return
	}

	tail = fmt.Sprintf("%v", s.limit)
	return
}

// Compiles SQL tail for defined group scope
// TODO: should be compiled via dialects
func (s *{{ .ScopeType }}) getGroupTail() (tail string, args []interface{}, err error) {
	tail = strings.Join(s.groupBy, ", ")

	return
}

// Compiles SQL tail for defined order scope
// TODO: should be compiled via dialects
func (s *{{ .ScopeType }}) getOrderTail() (tail string, args []interface{}, err error) {
	var fieldName string
	var orderStringParts []string

	for idx,orderStr := range s.order {
		switch (idx%2) {
			case 0:
				fieldName       = orderStr
			case 1:
				orderDirection := orderStr

				orderStringParts = append(orderStringParts, s.db.EscapeTableName(fieldName)+" "+orderDirection)
		}
	}

	tail = strings.Join(orderStringParts, ", ")

	return
}

// Compiles SQL tail for defined filter
// TODO: should be compiled via dialects
func (s *{{ .ScopeType }}) getWhereTailForFilter(filter {{ .FilterType }}) (tail string, whereTailArgs []interface{}, err error) {
	return s.db.GetWhereTailForFilter({{ .Type }}(filter), {{ if .ImitateGorm }}{{ .TableVar }}.ColumnNameByFieldName{{else}}nil{{end}}, "", {{ .ImitateGorm }})
}

// parseQuerierArgs considers different ways of defning the tail (using scope properties or/and in_args)
func (s {{ .ScopeType }}) parseWhereTailComponent(in_args []interface{}, placeholderCounter *int) (tail string, args []interface{}, err error) {
	if len(in_args) > 0 {
		switch arg := in_args[0].(type) {
{{- if .IsTable }}
		case int:
			tail = "{{ .PKField.Column }} "+s.db.OperatorAndPlaceholderOfValueForSQL(in_args[0], *placeholderCounter)
			*placeholderCounter++
			args = s.db.ValueForSQL(in_args[0])
{{- end }}
		case string:
			tailWords := s.db.SplitConditionByPlaceholders(arg)

			if len(tailWords)-1 != len(in_args[1:]) {
				panic(fmt.Errorf("The pattern doesn't fit for passed arguments (wrong number of question marks?): len(tailWords)-1 != len(in_args[1:]): <%v> <%v>", arg, in_args[1:]))
			}

			for idx, rawNewArgs := range in_args[1:] {
				newArgs := s.db.ValueForSQL(rawNewArgs)
				newTailWords := []string{}
				for range newArgs {
					newTailWords = append(newTailWords, s.db.GetDialect().Placeholder(*placeholderCounter))
					*placeholderCounter++
				}
				tail += tailWords[idx] + strings.Join(newTailWords, ",")
				args = append(args, newArgs...)
			}
			tail += tailWords[len(in_args[1:])]

			return
		case *{{ .Type }}:
			in_args[0] = *arg
			return s.parseWhereTailComponent(in_args, placeholderCounter)
{{- if .IsPrivateStruct }}
		case *{{ .FilterShorthandType }}:
			in_args[0] = *arg
			return s.parseWhereTailComponent(in_args, placeholderCounter)
{{- end }}
		case *{{ .FilterType }}:
			in_args[0] = *arg
			return s.parseWhereTailComponent(in_args, placeholderCounter)
		case {{ .Type }}:
			if len(in_args) > 1 {
				s = *s.Where(in_args[1], in_args[2:]...)
			}
			tail, args, err = s.getWhereTailForFilter({{ .FilterType }}(arg))
{{- if .IsPrivateStruct }}
		case {{ .FilterShorthandType }}:
			if len(in_args) > 1 {
				s = *s.Where(in_args[1], in_args[2:]...)
			}
			tail, args, err = s.getWhereTailForFilter({{ .FilterType }}(arg))
{{- end }}
		case {{ .FilterType }}:
			if len(in_args) > 1 {
				s = *s.Where(in_args[1], in_args[2:]...)
			}
			tail, args, err = s.getWhereTailForFilter(arg)
		default:
			err = fmt.Errorf("Invalid first element of \"in_args\" (%T). It should be a string or {{ .FilterType }}.", arg)
			return
		}
	}

	return
}

// Compiles SQL tail for defined filter
// TODO: should be compiled via dialects
func (s *{{ .ScopeType }}) getWhereTail() (tail string, whereTailArgs []interface{}, err error) {
	var whereTailStringParts []string

	if len(s.where) == 0 {
		return
	}

	placeholderCounter := 0

	for _,whereComponent := range s.where {
		var whereTailStringPart string
		var whereTailArgsPart []interface{}

		whereTailStringPart, whereTailArgsPart, err = s.parseWhereTailComponent(whereComponent, &placeholderCounter)
		if err != nil {
			return
		}

		if len(whereTailStringPart) > 0 {
			whereTailStringParts = append(whereTailStringParts, whereTailStringPart)
		}
		whereTailArgs = append(whereTailArgs, whereTailArgsPart...)
	}

	if len(whereTailStringParts) == 0 {
		return
	}

	tail = "(" + strings.Join(whereTailStringParts, ") AND (") + ")"

	return
}

func (s {{ .Type }}) Where(requiredArg interface{}, args ...interface{}) (scope *{{ .ScopeType }}) { return s.Scope().Where(requiredArg, args...) }
func (s {{ .ScopeType }}) Where(requiredArg interface{}, in_args ...interface{}) *{{ .ScopeType }} {
	s.where = append(s.where, append([]interface{}{ requiredArg }, in_args...))
	return &s
}
func (s {{ .ScopeType }}) SetWhere(where [][]interface{}) *{{ .ScopeType }} {
	s.where = where
	return &s
}
func (s {{ .ScopeType }}) GetWhere() ([][]interface{}) {
	return s.where
}

// Sets all scope-related parameters to be equal as in passed scope (as an argument)
func (s {{ .ScopeType }}) Set{{ if eq .ImitateGorm true }}Reform{{ end }}Scope(anotherScope reform.{{ if eq .ImitateGorm true }}GormImitate{{ end }}Scope) *{{ .ScopeType }} {
	s.where   = anotherScope.GetWhere()
	s.order   = anotherScope.GetOrder()
	s.groupBy = anotherScope.GetGroup()
	s.limit   = anotherScope.GetLimit()
	s.db      = anotherScope.Get{{ if eq .ImitateGorm true }}Reform{{ end }}DB()

	return &s
}
func (s {{ .ScopeType }}) ISet{{ if eq .ImitateGorm true }}Reform{{ end }}Scope(anotherScope reform.{{ if eq .ImitateGorm true }}GormImitate{{ end }}Scope) reform.{{ if eq .ImitateGorm true }}GormImitate{{ end }}Scope {
	return s.ISet{{ if eq .ImitateGorm true }}Reform{{ end }}Scope(anotherScope)
}

// Compiles SQL tail for defined db/where/order/limit scope
// TODO: should be compiled via dialects
func (s *{{ .ScopeType }}) getTail() (tail string, args []interface{}, err error) {
	whereTailString, whereTailArgs, err := s.getWhereTail()

	if err != nil {
		return
	}
	groupTailString, groupTailArgs, err := s.getGroupTail()
	if err != nil {
		return
	}
	orderTailString, orderTailArgs, err := s.getOrderTail()
	if err != nil {
		return
	}
	limitTailString, _            , err := s.getLimitTail()
	if err != nil {
		return
	}

	args = append(whereTailArgs, append(groupTailArgs, orderTailArgs...)...)

	if len(whereTailString) > 0 {
		whereTailString = " WHERE "+whereTailString+" "
	}

	if len(groupTailString) > 0 {
		groupTailString = " GROUP BY "+groupTailString+" "
	}

	if len(orderTailString) > 0 {
		orderTailString = " ORDER BY "+orderTailString+" "
	}

	if len(limitTailString) > 0 {
		limitTailString = " LIMIT "+limitTailString+" "
	}

	tail = whereTailString+groupTailString+orderTailString+limitTailString

	if len(s.appendTail) > 0 {
		tail += " " + s.appendTail
	}

	return

}

// SelectRows is a simple wrapper to get raw "sql.Rows"
func (s {{ .Type }}) SelectRows(query string, args ...interface{}) (rows *sql.Rows, err error) { return s.Scope().SelectRows(query, args...) }
func (s *{{ .ScopeType }}) SelectRows(query string, queryArgs ...interface{}) (rows *sql.Rows, err error) {
	tail, args, err := s.getTail()
	if err != nil {
		return
	}

	return s.db.Query("SELECT "+query+" FROM ` + "`" + `"+{{ .TableVar }}.Name()+"` + "`" + ` "+tail, append(queryArgs, args...)...)
}

func (s *{{ .ScopeType }}) callStructMethod(str *{{ .Type }}, methodName string) error {
	if method := reflect.ValueOf(str).MethodByName(methodName); method.IsValid() {
		switch f := method.Interface().(type) {
		case func():
			f()

		case func(reform.ReformDBTX):
			f(s.db)

		case func(*{{ .ScopeType }}):
			f(s)

		case func(interface{}): // For compatibility with other ORMs
			f(s.db)

		case func() error:
			return f()

		case func(reform.ReformDBTX) error:
			return f(s.db)

		case func(*{{ .ScopeType }}) error:
			return f(s)

		case func(interface{}) error: // For compatibility with other ORMS
			return f(s.db)

		default:
			panic("Unknown type of method: \""+methodName+"\"")
		}
	}
	return nil
}

func (s {{ .ScopeType }}) checkDb() {
	if s.db == nil {
		panic("s.db == nil")
	}
}

// Select is a handy wrapper for SelectRows() and NextRow(): it makes a query and collects the result into a slice
func (s {{ .Type }}) Select(args ...interface{}) (result []{{.Type}}, err error) { return s.Scope().Select(args...) }
func (s {{ .ScopeType }}) Select(args ...interface{}) (result []{{.Type}}, err error) {
	s.checkDb()

	if len(args) > 0 {
		s = *s.Where(args[0], args[1:]...)
	}
	tail, args, err := s.getTail()
	if err != nil {
		return
	}

	rows, err := s.db.FlexSelectRows({{ .TableVar }}, s.tableQuery, s.fieldsFilter, tail, args...)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		item := {{ .Type }}{}
		err = rows.Scan(item.FieldPointersByNames(s.fieldsFilter)...)
		if err != nil {
			return
		}

		s.callStructMethod(&item, "AfterFind")

		result = append(result, item)
	}

	err = rows.Err()
	if err != nil {
		return
	}

	return
}
func (s {{ .Type }}) SelectI(args ...interface{}) (result interface{}, err error) { return s.Scope().Select(args...) }
func (s {{ .ScopeType }}) SelectI(args ...interface{}) (result interface{}, err error) { return s.Select(args...) }

// "First" a method to select and return only one record.
func (s {{ .Type }}) First(args ...interface{}) (result {{.Type}}, err error) { return s.Scope().First(args...) }
func (s {{ .ScopeType }}) First(args ...interface{}) (result {{.Type}}, err error) {
	s.checkDb()

	if len(args) > 0 {
		s = *s.Where(args[0], args[1:]...)
	}
	tail, args, err := s.Limit(1).getTail()
	if err != nil {
		return
	}

	err = s.db.FlexSelectOneTo(&result, s.tableQuery, s.fieldsFilter, tail, args...)

	return
}
func (s {{ .Type }}) FirstI(args ...interface{}) (result interface{}, err error) { return s.Scope().First(args...) }
func (s {{ .ScopeType }}) FirstI(args ...interface{}) (result interface{}, err error) { return s.First(args...) }

// Sets "GROUP BY".
func (s {{ .Type }}) Group(args ...interface{}) (scope *{{ .ScopeType }}) { return s.Scope().Group(args...) }
func (s {{ .ScopeType }}) Group(argsI ...interface{}) (*{{ .ScopeType }}) {
	for _,argI := range argsI {
		s.groupBy = append(s.groupBy, argI.(string))
	}

	return &s
}
func (s {{ .ScopeType }}) SetGroup(groupBy []string) (*{{ .ScopeType }}) {
	s.groupBy = groupBy
	return &s
}
func (s {{ .ScopeType }}) GetGroup() ([]string) {
	return s.groupBy
}

// Sets a table query. For example SetTableQuery("table1 JOIN table2 USING(key)")
func (s {{ .Type }}) SetTableQuery(query string) (scope *{{ .ScopeType }}) { return s.Scope().SetTableQuery(query) }
func (s {{ .ScopeType }}) SetTableQuery(query string) (*{{ .ScopeType }}) {
	if query == "" {
		s.tableQuery = nil
	} else {
		s.tableQuery = &query
	}
	return &s
}
func (s {{ .ScopeType }}) GetTableQuery() string {
	if s.tableQuery != nil {
		return *s.tableQuery
	}
	return s.db.QualifiedView(s.View())
}

// Sets which structure fields should be queried while Select()/First(). For example SetFields("StructField1", "StructIdField", "StructCommentsField"). Could be used just to speed up a query.
// It's not recommended to use this function!
func (s {{ .Type }}) SetQueryFieldsByNames(fields ...string) (scope *{{ .ScopeType }}) { return s.Scope().SetQueryFieldsByNames(fields...) }
func (s {{ .ScopeType }}) SetQueryFieldsByNames(fields ...string) (*{{ .ScopeType }}) {
	s.fieldsFilter = fields
	return &s
}
func (s {{ .ScopeType }}) GetQueryFields() []string {
	return s.fieldsFilter
}

{{- if not .SkipMethodOrder }}
// Sets order. Arguments should be passed by pairs column-{ASC,DESC}. For example Order("id", "ASC", "value" "DESC")
func (s {{ .Type }}) Order(args ...interface{}) (scope *{{ .ScopeType }}) { return s.Scope().Order(args...) }
func (s {{ .ScopeType }}) Order(argsI ...interface{}) (*{{ .ScopeType }}) {
	switch len(argsI) {
	case 0:
	case 1:
		arg   := argsI[0].(string)
		args0 := strings.Split(arg, ",")
		var args []string
		for _,arg0 := range args0 {
			args = append(args, strings.Split(arg0, ":")...)
		}
		s.order = args
	default:
		var args []string
		for _,argI := range argsI {
			args = append(args, argI.(string))
		}
		s.order = args
	}

	return &s
}
{{- end }}
func (s {{ .ScopeType }}) SetOrder(order []string) (*{{ .ScopeType }}) {
	s.order = order
	return &s
}
func (s {{ .ScopeType }}) GetOrder() ([]string) {
	return s.order
}

func (s {{ .Type }}) SetSQLAppend(appendTail string) (scope *{{ .ScopeType }}) { return s.Scope().SetSQLAppend(appendTail) }
func (s {{ .ScopeType }}) SetSQLAppend(appendTail string) (*{{ .ScopeType }}) {
	s.appendTail = appendTail
	return &s
}

// Sets limit.
func (s {{ .Type }}) {{ if eq .ImitateGorm true }}Reform{{ end }}Limit(limit int) (scope *{{ .ScopeType }}) { return s.Scope().Limit(limit) }
func (s *{{ .ScopeType }}) Limit(limit int) (*{{ .ScopeType }}) {
	s.limit = limit
	return s
}

// Gets limit
func (s {{ .ScopeType }}) GetLimit() int {
	return s.limit
}

{{- if .IsTable }}

// "Reload" reloads record using Primary Key
func (s *{{ .FilterType }}) {{ if eq .ImitateGorm true }}Reform{{ end }}Reload(db *reform.DB) error { return (*{{ .Type }})(s).{{ if eq .ImitateGorm true }}Reform{{ end }}Reload(db) }
func (s *{{ .Type }}) {{ if eq .ImitateGorm true }}Reform{{ end }}Reload(db *reform.DB) (err error) {
	return db.FindByPrimaryKeyTo(s, s.PKValue())
}

// Create and Insert inserts new record to DB
func (s *{{ .Type }}) {{ if eq .ImitateGorm true }}Reform{{ end }}Create() (err error) { return s.PtrScope().{{ if eq .ImitateGorm true }}Reform{{ end }}Create() }
func (s *{{ .ScopeType }}) {{ if eq .ImitateGorm true }}Reform{{ end }}Create() (err error) {
	return s.{{ if eq .ImitateGorm true }}Reform{{ end }}Insert()
}
func (s *{{ .Type }}) {{ if eq .ImitateGorm true }}Reform{{ end }}Insert() (err error) { return s.PtrScope().{{ if eq .ImitateGorm true }}Reform{{ end }}Insert() }
func (s *{{ .ScopeType }}) {{ if eq .ImitateGorm true }}Reform{{ end }}Insert() (err error) {
	s.checkDb()
	err = s.db.Insert(s.item)
	if err == nil {
		s.doLog("INSERT")
	}
	return err
}

// Replace "REPLACE INTO" new record to DB
func (s *{{ .Type }}) {{ if eq .ImitateGorm true }}Reform{{ end }}Replace() (err error) { return s.PtrScope().{{ if eq .ImitateGorm true }}Reform{{ end }}Replace() }
func (s *{{ .ScopeType }}) {{ if eq .ImitateGorm true }}Reform{{ end }}Replace() (err error) {
	s.checkDb()
	err = s.db.Replace(s.item)
	if err == nil {
		s.doLog("REPLACE")
	}
	return err
}

// Save inserts new record to DB is PK is zero and updates existing record if PK is not zero
func (s *{{ .Type }}) {{ if eq .ImitateGorm true }}Reform{{ end }}Save() (err error) { return s.PtrScope().{{ if eq .ImitateGorm true }}Reform{{ end }}Save() }
func (s *{{ .ScopeType }}) {{ if eq .ImitateGorm true }}Reform{{ end }}Save() (err error) {
	s.checkDb()
	err = s.db.Save(s.item)
	if err == nil {
		s.doLog("INSERT")
	}
	return err
}

// Update updates existing record in DB
func (s {{ .Type }}) {{ if eq .ImitateGorm true }}Reform{{ end }}Update() (err error) { return s.Scope().{{ if eq .ImitateGorm true }}Reform{{ end }}Update() }
func (s *{{ .ScopeType }}) {{ if eq .ImitateGorm true }}Reform{{ end }}Update() (err error) {
	s.checkDb()
	err = s.db.Update(s.item)
	if err == nil {
		s.doLog("UPDATE")
	}
	return err
}

// Delete deletes existing record in DB
func (s {{ .Type }}) {{ if eq .ImitateGorm true }}Reform{{ end }}Delete() (err error) { return s.Scope().{{ if eq .ImitateGorm true }}Reform{{ end }}Delete() }
func (s *{{ .ScopeType }}) {{ if eq .ImitateGorm true }}Reform{{ end }}Delete() (err error) {
	s.checkDb()
	err = s.db.Delete(s.item)
	if err == nil {
		s.doLog("DELETE")
	}
	return err
}

func (s *{{ .ScopeType }}) doLog(requestType string) {
	if !s.loggingEnabled {
		return
	}

	var logRow {{ .LogType }}
	logRow.{{.Type}}  = *s.item
	logRow.LogAuthor  = s.loggingAuthor
	logRow.LogAction  = requestType
	logRow.LogDate    = time.Now()
	logRow.LogComment = s.loggingComment

	s.db.Insert(&logRow)
}

// Enables logging to table "{{ .SQLName }}_log". This table should has the same schema, except:
// - Unique/Primary keys should be removed
// - Should be added next fields: "log_author" (nullable string), "log_date" (timestamp), "log_action" (enum("INSERT", "UPDATE", "DELETE")), "log_comment" (string)
func (s *{{ .Type }}) Log(enableLogging bool, author *string, commentFormat string, commentArgs ...interface{}) (scope *{{ .ScopeType }}) { return s.Scope().Log(enableLogging, author, commentFormat, commentArgs...) }
func (s *{{ .ScopeType }}) Log(enableLogging bool, author *string, commentFormat string, commentArgs ...interface{}) (scope *{{ .ScopeType }}) {
	s.loggingEnabled = enableLogging
	s.loggingAuthor  = author
	s.loggingComment = fmt.Sprintf(commentFormat, commentArgs...)

	return s
}

// Table returns Table object for that record.
func (s {{ .Type }}) Table() reform.Table {
	return {{ .TableVar }}
}

// PKValue returns a value of primary key for that record.
// Returned interface{} value is never untyped nil.
func (s {{ .Type }}) PKValue() interface{} {
	return s.{{ .PKField.Name }}
}

// PKPointer returns a pointer to primary key field for that record.
// Returned interface{} value is never untyped nil.
func (s {{ .Type }}) PKPointer() interface{} {
	return &s.{{ .PKField.Name }}
}

// HasPK returns true if record has non-zero primary key set, false otherwise.
func (s {{ .Type }}) HasPK() bool {
	return s.{{ .PKField.Name }} != {{ .TableVar }}.z[{{ .TableVar }}.s.PKFieldIndex]
}

// SetPK sets record primary key.
func (s *{{ .FilterType }}) SetPK(pk interface{}) { (*{{ .Type }})(s).SetPK(pk) }
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
	_ reform.Struct = (*{{ .Type }})(nil)
{{- if .IsTable }}
	_ reform.Table  = {{ .TableVar }}
	_ reform.Record = (*{{ .Type }})(nil)
{{- end }}
	_ fmt.Stringer  = (*{{ .Type }})(nil)

	// querier
	{{ .QuerierVar }} = {{ .Type }}{} // Should be read only
	defaultDB_{{ .Type }} *reform.DB
)

`))

	initTemplate = template.Must(template.New("init").Parse(`
func init() {
	{{- range $i, $sd := . }}
	//parse.AssertUpToDate(&{{ $sd.TableVar }}.s, new({{ $sd.Type }})) // Temporary disabled (doesn't work with arbitary types like "type sliceString []string")
	{{- end }}
}
`))
)
