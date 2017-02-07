package reform

import (
	"database/sql"
	"fmt"
	"reflect"
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
			panic("Unknown type of method: \""+methodName+"\"")
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
func (q *Querier) WithTag(format string, a ...interface{}) *Querier {
	newQ := newQuerier(q.dbtx, q.Dialect, q.Logger, q.dbForCallbacks)
	if len(a) == 0 {
		newQ.tag = format
	} else {
		newQ.tag = fmt.Sprintf(format, a...)
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
func (q *Querier) Query(query string, args ...interface{}) (*sql.Rows, error) {
	q.logBefore(query, args)
	start := time.Now()
	rows, err := q.dbtx.Query(query, args...)
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

// check interface
var _ DBTX = (*Querier)(nil)
