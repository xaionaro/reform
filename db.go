package reform

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// DBInterface is a subset of *sql.DB used by reform.
// Can be used together with NewDBFromInterface for easier integration with existing code or for passing test doubles.
type DBInterface interface {
	DBTX
	Begin() (*sql.Tx, error)
}

// check interface
var _ DBInterface = (*sql.DB)(nil)

// DB represents a connection to SQL database.
type DB struct {
	*Querier
	db DBInterface
}

// NewDB creates new DB object for given SQL database connection.
func NewDB(db *sql.DB, dialect Dialect, logger Logger) *DB {
	return NewDBFromInterface(db, dialect, logger)
}

// NewDBFromInterface creates new DB object for given DBInterface.
// Can be used for easier integration with existing code or for passing test doubles.
func NewDBFromInterface(db DBInterface, dialect Dialect, logger Logger) *DB {
	newDB := DB{db: db}
	newDB.Querier = newQuerier(db, dialect, logger, &newDB)
	return &newDB
}

// DBInterface returns DBInterface associated with a given DB object.
func (db *DB) DBInterface() DBInterface {
	return db.db
}

// Begin starts a transaction.
func (db *DB) Begin() (*TX, error) {
	db.logBefore("BEGIN", nil)
	start := time.Now()
	tx, err := db.db.Begin()
	db.logAfter("BEGIN", nil, time.Since(start), err)
	if err != nil {
		return nil, err
	}
	return NewTX(tx, db.Dialect, db.Logger, db), nil
}

// InTransaction wraps function execution in transaction, rolling back it in case of error or panic,
// committing otherwise.
func (db *DB) InTransaction(f func(t *TX) error) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	var committed bool
	defer func() {
		if !committed {
			// always return f() or Commit() error, not possible Rollback() error
			_ = tx.Rollback()
		}
	}()

	err = f(tx)
	if err == nil {
		err = tx.Commit()
	}
	if err == nil {
		committed = true
	}
	return err
}

// OperatorAndPlaceholderOfValueForSQL generates an operator and placeholder for a value intor a condition into SQL query (for exampel "= ?") for the first argument of sql.Exec()
func (db DB) OperatorAndPlaceholderOfValueForSQL(valueI interface{}, placeholderCounter int) string {
	switch valueI.(type) {
	case []int, []string, []float32, []float64, []int64:
		return " IN ("+db.Dialect.Placeholder(placeholderCounter)+")"
	case int, string, float32, float64, int64:
		return " = "+db.Dialect.Placeholder(placeholderCounter)
	case nil:
		return " IS NULL"
	default:
		return " = "+db.Dialect.Placeholder(placeholderCounter)
	}
}

// ValueForSQL generates the value argument for sql.Exec() [not-the-first arguments]
func (db DB) ValueForSQL(valueI interface{}) interface{} {
	switch value := valueI.(type) {
	case []int, []string, []float32, []float64, []int64:
		return `"`+strings.Replace(strings.Trim(fmt.Sprintf("%v", value), "[]"), ` `, `", "`, -1)+`"`
	case int, string, float32, float64, int64:
		return value
	case nil:
		return nil
	default:
		stringer, ok := value.(Stringer)
		if !ok {
			return fmt.Sprintf("%v", value)
		} else {
			return stringer.String()
		}
	}
}

// check interface
var _ DBTX = (*DB)(nil)
