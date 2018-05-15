package reform

import (
	"database/sql"
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

// check interface
var _ DBTX = (*DB)(nil)
