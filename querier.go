package reform

import (
	"database/sql"
	"fmt"
	mysqlDriver "github.com/go-sql-driver/mysql"
	"reflect"
	"strings"
	"time"
)

// Querier performs queries and commands.
type Querier struct {
	dbtx DBTX
	tag  string
	Dialect
	Logger         Logger
	dbForCallbacks *DB
}

func newQuerier(dbtx DBTX, dialect Dialect, logger Logger, dbForCallbacks *DB) *Querier {
	return &Querier{
		dbtx:           dbtx,
		Dialect:        dialect,
		Logger:         logger,
		dbForCallbacks: dbForCallbacks,
	}
}

func (q *Querier) logBefore(query string, args []interface{}) {
	if q.Logger != nil {
		q.Logger.Before(query, args)
	}
}

func (q *Querier) logAfter(query string, args []interface{}, d time.Duration, err error) {
	if q.Logger != nil {
		q.Logger.After(query, args, d, err)
	}
}

func (q *Querier) callStructMethod(str Struct, methodName string) error {
	if method := reflect.ValueOf(str).MethodByName(methodName); method.IsValid() {
		switch f := method.Interface().(type) {
		case func():
			f()

		case func(*DB):
			f(q.dbForCallbacks)

		case func(*Querier):
			f(q)

		case func(interface{}): // For compatibility with other ORMs
			f(q.dbForCallbacks)

		case func() error:
			return f()

		case func(*DB) error:
			return f(q.dbForCallbacks)

		case func(*Querier) error:
			return f(q)

		case func(interface{}) error: // For compatibility with other ORMS
			return f(q.dbForCallbacks)

		default:
			panic("Unknown type of method: \"" + methodName + "\"")
		}
	}
	return nil
}

func (q *Querier) startQuery(command string) string {
	if q.tag == "" {
		return command
	}
	return command + " /* " + q.tag + " */"
}

// WithTag returns a copy of Querier with set tag. Returned Querier is tied to the same DB or TX.
// See Tagging section in documentation for details.
func (q *Querier) WithTag(format string, args ...interface{}) *Querier {
	newQ := newQuerier(q.dbtx, q.Dialect, q.Logger, q.dbForCallbacks)
	if len(args) == 0 {
		newQ.tag = format
	} else {
		newQ.tag = fmt.Sprintf(format, args...)
	}
	return newQ
}

// QualifiedView returns quoted qualified view name.
func (q *Querier) QualifiedView(view View) string {
	v := q.QuoteIdentifier(view.Name())
	if view.Schema() != "" {
		v = q.QuoteIdentifier(view.Schema()) + "." + v
	}
	return v
}

// QualifiedColumns returns a slice of quoted qualified column names for given view.
func (q *Querier) QualifiedColumns(view View) []string {
	v := q.QualifiedView(view)
	res := view.Columns()
	for i := 0; i < len(res); i++ {
		res[i] = v + "." + q.QuoteIdentifier(res[i])
	}
	return res
}

// Exec executes a query without returning any rows.
// The args are for any placeholder parameters in the query.
func (q *Querier) Exec(query string, args ...interface{}) (sql.Result, error) {
	q.logBefore(query, args)
	start := time.Now()
	res, err := q.dbtx.Exec(query, args...)
	q.logAfter(query, args, time.Since(start), err)
	return res, err
}

// Query executes a query that returns rows, typically a SELECT.
// The args are for any placeholder parameters in the query.
func (q *Querier) Query(query string, args ...interface{}) (rows *sql.Rows, err error) {
	q.logBefore(query, args)
	start := time.Now()
	for {
		rows, err = q.dbtx.Query(query, args...)
		if err == mysqlDriver.ErrInvalidConn {
			continue
		}
		break
	}
	q.logAfter(query, args, time.Since(start), err)
	return rows, err
}

// QueryRow executes a query that is expected to return at most one row.
// QueryRow always returns a non-nil value. Errors are deferred until Row's Scan method is called.
func (q *Querier) QueryRow(query string, args ...interface{}) *sql.Row {
	q.logBefore(query, args)
	start := time.Now()
	row := q.dbtx.QueryRow(query, args...)
	q.logAfter(query, args, time.Since(start), nil)
	return row
}

// OperatorAndPlaceholderOfValueForSQL generates an operator and placeholder for a value intor a condition into SQL query (for exampel "= ?") for the first argument of sql.Exec()
func (querier Querier) OperatorAndPlaceholderOfValueForSQL(valueI interface{}, placeholderCounter int) string {
	switch valueI.(type) {
	case []int, []string, []float32, []float64, []int64:
		return " IN (" + querier.Dialect.Placeholder(placeholderCounter) + ")"
	case int, string, float32, float64, int64:
		return " = " + querier.Dialect.Placeholder(placeholderCounter)
	case nil:
		return " IS NULL"
	default:
		return " = " + querier.Dialect.Placeholder(placeholderCounter)
	}
}

/*
func sliceWrapperValue(value interface{}) string {
	return `"`+strings.Replace(strings.Trim(fmt.Sprintf("%v", value), "[]"), ` `, `", "`, -1)+`"`
}

type intSliceWrapper []int
func (a intSliceWrapper) Value() (driver.Value, error) {
	return sliceWrapperValue(a), nil
}
type int64SliceWrapper []int64
func (a int64SliceWrapper) Value() (driver.Value, error) {
	return sliceWrapperValue(a), nil
}
type stringSliceWrapper []string
func (a stringSliceWrapper) Value() (driver.Value, error) {
	return sliceWrapperValue(a), nil
}
type float32SliceWrapper []float32
func (a float32SliceWrapper) Value() (driver.Value, error) {
	return sliceWrapperValue(a), nil
}
type float64SliceWrapper []float64
func (a float64SliceWrapper) Value() (driver.Value, error) {
	return sliceWrapperValue(a), nil
}*/

func sliceWrapper(sliceI interface{}) (result []interface{}) {
	slice := reflect.ValueOf(sliceI)
	length := slice.Len()
	for i := 0; i < length; i++ {
		item := slice.Index(i)
		result = append(result, item.Interface())
	}

	return
}

// ValueForSQL generates the value argument for sql.Exec() [not-the-first arguments]
func (querier Querier) ValueForSQL(valueI interface{}) []interface{} {
	switch value := valueI.(type) {
	/*case []int:
		return []interface{}{intSliceWrapper(value)}
	case []int64:
		return []interface{}{int64SliceWrapper(value)}
	case []string:
		return []interface{}{stringSliceWrapper(value)}
	case []float32:
		return []interface{}{float32SliceWrapper(value)}
	case []float64:
		return []interface{}{float64SliceWrapper(value)}*/
	case []int, []int64, []string, []float32, []float64:
		return sliceWrapper(value)
	case int, string, float32, float64, int64:
		return []interface{}{value}
	case nil:
		return []interface{}{}
	default:
		driverValuer, ok := value.(DriverValuer)
		if ok {
			return []interface{}{driverValuer}
		}

		stringer, ok := value.(Stringer)
		if ok {
			return []interface{}{stringer.String()}
		}

		return []interface{}{fmt.Sprintf("%v", value)}
	}
}

func (querier Querier) SplitConditionByPlaceholders(condition string) []string {
	// TODO: use Dialects. Right now it's hacky MySQL solution only :(
	return strings.Split(condition, "?")
}

func (querier Querier) EscapeTableName(tableName string) string {
	return querier.Dialect.QuoteIdentifier(tableName)
}

func (querier Querier) ColumnDefinitionsOfStruct(structInfo StructInfo) (definitions []string) {
	for _, field := range structInfo.Fields {
		if field.Column == "" {
			continue
		}
		definitions = append(definitions, querier.Dialect.ColumnDefinitionForField(field))
	}

	return
}

func (querier Querier) CreateTableIfNotExists(structInfo StructInfo) (bool, error) {
	// TODO: correctly escape table name
	request := "CREATE TABLE IF NOT EXISTS `" + structInfo.SQLName + "` (" +
		strings.Join(querier.ColumnDefinitionsOfStruct(structInfo), ", ") +
		")"

	var postQueries []string
	for _, field := range structInfo.Fields {
		if field.Column == "" {
			continue
		}
		postQuery := querier.Dialect.ColumnDefinitionPostQueryForField(structInfo, field)
		if postQuery == "" {
			continue
		}
		postQueries = append(postQueries, postQuery)
	}

	_, err := querier.Exec(request + "; " + strings.Join(postQueries, "; "))
	return false, err
}

func (querier Querier) GetWhereTailForFilter(filter interface{}, columnNameByFieldName func(string) string, prefix string, imitateGorm bool) (tail string, whereTailArgs []interface{}, err error) {
	var whereTailStringParts []string

	v := reflect.ValueOf(filter)
	vT := v.Type()

	numField := v.NumField()

	placeholderCounter := 0
	for i := 0; i < numField; i++ {
		vTF := vT.Field(i)
		tag := vTF.Tag
		if tag.Get("sql") == "-" || tag.Get("reform") == "-" {
			continue
		}

		f := v.Field(i)
		fT := f.Type()

		var columnName string
		if imitateGorm {
			columnName = prefix + columnNameByFieldName(vTF.Name)
		} else {
			vs := vT.Field(i)
			columnName = prefix + strings.Split(vs.Tag.Get("reform"), ",")[0]
		}

		switch fT.Kind() {
		case reflect.Struct:
			var embedded string
			if imitateGorm {
				_, _, embedded, _ = ParseStructFieldGormTag(tag.Get("gorm"), "")
			} else {
				_, _, embedded, _ = ParseStructFieldTag(tag.Get("reform"))
			}

			switch embedded {
			case "embedded", "prefixed":
				nestedPrefix := prefix
				if embedded == "prefixed" {
					nestedPrefix += columnName + "__"
				}
				tailPart, args, er := querier.GetWhereTailForFilter(f.Interface(), columnNameByFieldName, nestedPrefix, imitateGorm)
				if er != nil {
					err = er
					return
				}
				if len(tailPart) > 0 {
					whereTailStringParts = append(whereTailStringParts, tailPart)
					whereTailArgs = append(whereTailArgs, args...)
				}
				continue
			case "":
				if reflect.DeepEqual(f.Interface(), reflect.Zero(fT).Interface()) {
					continue
				}
			default:
				panic(fmt.Errorf("Not implemented case: embedded == \"%v\": %v (%T)", embedded, vTF.Name, f.Interface()))
			}
		case reflect.Array, reflect.Slice, reflect.Map:
			if reflect.DeepEqual(f.Interface(), reflect.Zero(fT).Interface()) {
				continue
			}
		default:
			if f.Interface() == reflect.Zero(fT).Interface() {
				continue
			}
		}

		placeholderCounter++
		whereTailStringParts = append(whereTailStringParts, querier.EscapeTableName(columnName)+" = "+querier.Dialect.Placeholder(placeholderCounter))
		whereTailArgs = append(whereTailArgs, f.Interface())
	}

	tail = strings.Join(whereTailStringParts, " AND ")

	return
}

func (querier Querier) GetDialect() Dialect {
	return querier.Dialect
}

// check interface
var _ DBTX = (*Querier)(nil)
