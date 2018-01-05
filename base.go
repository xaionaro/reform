package reform

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"github.com/jinzhu/gorm"
	"strings"
)

var (
	// ErrNoRows is returned from various methods when query produced no rows.
	ErrNoRows = sql.ErrNoRows

	// ErrTxDone is returned from Commit() and Rollback() TX methods when transaction is already
	// committed or rolled back.
	ErrTxDone = sql.ErrTxDone

	// ErrNoPK is returned from various methods when primary key is required and not set.
	ErrNoPK = errors.New("reform: no primary key")
)

// FieldInfo represents information about struct field.
type FieldInfo struct {
	Name       string      // field name as defined in source file, e.g. Name
	IsPK       bool        // is this field a primary key field
	IsUnique   bool        // this field uses unique index in RDBMS
	HasIndex   bool        // this field uses index in RDBMS
	Type       string      // field type as defined in source file, e.g. string
	Column     string      // SQL database column name from "reform:" struct field tag, e.g. name
	FieldsPath []FieldInfo // A path to the field via nested structures
}

func (f FieldInfo) FullName() string {
	var prefix string
	for _, step := range f.FieldsPath {
		prefix += step.Name + "."
	}

	return prefix + f.Name
}

// StructInfo represents information about struct.
type StructInfo struct {
	Type            string      // struct type as defined in source file, e.g. User
	SQLSchema       string      // SQL database schema name from magic "reform:" comment, e.g. public
	SQLName         string      // SQL database view or table name from magic "reform:" comment, e.g. users
	Fields          []FieldInfo // fields info
	PKFieldIndex    int         // index of primary key field in Fields, -1 if none
	ImitateGorm     bool        // act like GORM (https://github.com/jinzhu/gorm)
	SkipMethodOrder bool        // do not create method Order()
}

// Columns returns a new slice of column names.
func (s *StructInfo) Columns() []string {
	res := make([]string, len(s.Fields))
	for i, f := range s.Fields {
		res[i] = f.Column
	}
	return res
}

func (s *StructInfo) UnPointer() StructInfo {
	return *s
}

func (s StructInfo) ToLog() *StructInfo {
	s.SQLName += "_log"
	s.Fields = append(s.Fields, []FieldInfo{
		FieldInfo{Name: "LogAuthor", Type: "*string", Column: "log_author"},
		FieldInfo{Name: "LogAction", Type: "string", Column: "log_action"},
		FieldInfo{Name: "LogDate", Type: "time.Time", Column: "log_date"},
		FieldInfo{Name: "LogComment", Type: "string", Column: "log_comment"},
	}...)

	return &s
}

// IsTable returns true if this object represent information for table, false for view.
func (s *StructInfo) IsTable() bool {
	return s.PKFieldIndex >= 0
}

// PKField returns a primary key field, panics for views.
func (s *StructInfo) PKField() FieldInfo {
	if !s.IsTable() {
		panic("reform: not a table")
	}
	return s.Fields[s.PKFieldIndex]
}

// View represents SQL database view or table.
type View interface {
	// Schema returns a schema name in SQL database.
	Schema() string

	// Name returns a view or table name in SQL database.
	Name() string

	// Columns returns a new slice of column names for that view or table in SQL database.
	Columns() []string

	// ColumnNameByFieldName returns the column name by a given field name
	ColumnNameByFieldName(string) string

	// NewStruct makes a new struct for that view or table.
	NewStruct() Struct
}

// Table represents SQL database table with single-column primary key.
// It extends View.
type Table interface {
	View

	// NewRecord makes a new record for that table.
	NewRecord() Record

	// PKColumnIndex returns an index of primary key column for that table in SQL database.
	PKColumnIndex() uint

	// Tries to create the table if it doesn't exist
	CreateTableIfNotExists(*DB) (bool, error)
}

// Struct represents a row in SQL database view or table.
type Struct interface {
	// String returns a string representation of this struct or record.
	String() string

	// Values returns a slice of struct or record field values.
	// Returned interface{} values are never untyped nils.
	Values() []interface{}

	// Pointers returns a slice of pointers to struct or record fields.
	// Returned interface{} values are never untyped nils.
	Pointers() []interface{}

	// FieldPointerByName return the pointer to the field of the Struct by the field name
	FieldPointerByName(string) interface{}

	// FieldPointersByNames return pointers to fields of the Struct by field names
	FieldPointersByNames([]string) []interface{}

	// View returns View object for that struct.
	View() View
}

// Record represents a row in SQL database table with single-column primary key.
type Record interface {
	Struct

	// Table returns Table object for that record.
	Table() Table

	// PKValue returns a value of primary key for that record.
	// Returned interface{} value is never untyped nil.
	PKValue() interface{}

	// PKPointer returns a pointer to primary key field for that record.
	// Returned interface{} value is never untyped nil.
	PKPointer() interface{}

	// HasPK returns true if record has non-zero primary key set, false otherwise.
	HasPK() bool

	// SetPK sets record primary key.
	SetPK(pk interface{})
}

// DBTX is an interface for database connection or transaction.
// It's implemented by *sql.DB, *sql.Tx, *DB, *TX and *Querier.
type DBTX interface {
	// Exec executes a query without returning any rows.
	// The args are for any placeholder parameters in the query.
	Exec(query string, args ...interface{}) (sql.Result, error)

	// Query executes a query that returns rows, typically a SELECT.
	// The args are for any placeholder parameters in the query.
	Query(query string, args ...interface{}) (*sql.Rows, error)

	// QueryRow executes a query that is expected to return at most one row.
	// QueryRow always returns a non-nil value. Errors are deferred until Row's Scan method is called.
	QueryRow(query string, args ...interface{}) *sql.Row
}

// LastInsertIdMethod is a method of receiving primary key of last inserted row.
type LastInsertIdMethod int

const (
	// LastInsertId is method using sql.Result.LastInsertId().
	LastInsertId LastInsertIdMethod = iota

	// Returning is method using "RETURNING id" SQL syntax.
	Returning

	// OutputInserted is method using "OUTPUT INSERTED.id" SQL syntax.
	OutputInserted
)

// SelectLimitMethod is a method of limiting the number of rows in a query result.
type SelectLimitMethod int

const (
	// Limit is a method using "LIMIT N" SQL syntax.
	Limit SelectLimitMethod = iota

	// SelectTop is a method using "SELECT TOP N" SQL syntax.
	SelectTop
)

// DefaultValuesMethod is a method of inserting of row with all default values.
type DefaultValuesMethod int

const (
	// DefaultValues is a method using "DEFAULT VALUES"
	DefaultValues DefaultValuesMethod = iota

	// EmptyLists is a method using "() VALUES ()"
	EmptyLists
)

// Dialect represents differences in various SQL dialects.
type Dialect interface {
	// String returns dialect name.
	String() string

	// Placeholder returns representation of placeholder parameter for given index,
	// typically "?" or "$1".
	Placeholder(index int) string

	// Placeholders returns representation of placeholder parameters for given start index and count,
	// typically []{"?", "?"} or []{"$1", "$2"}.
	Placeholders(start, count int) []string

	// QuoteIdentifier returns quoted database identifier,
	// typically "identifier" or `identifier`.
	QuoteIdentifier(identifier string) string

	// LastInsertIdMethod returns a method of receiving primary key of last inserted row.
	LastInsertIdMethod() LastInsertIdMethod

	// SelectLimitMethod returns a method of limiting the number of rows in a query result.
	SelectLimitMethod() SelectLimitMethod

	// DefaultValuesMethod returns a method of inserting of row with all default values.
	DefaultValuesMethod() DefaultValuesMethod

	// ColumnDefinitionForField returns a string of column definition for a field
	ColumnDefinitionForField(FieldInfo) string

	// ColumnDefinitionForField returns a string of queries that should be executes after creating the field (like "CREATE INDEX" in sqlite)
	ColumnDefinitionPostQueryForField(StructInfo, FieldInfo) string
}

// Stringer represents any object with method "String() string" to stringify it's value
type Stringer interface {
	// Returns stringifier representation of the object
	String() string
}

// DriverValuer represents any object with method "Value() (driver.Value, error)" to SQL-ify it's value
type DriverValuer interface {
	// Returns SQL-fied representation of the object
	Value() (driver.Value, error)
}

// AfterDBer represents any object with method "AfterDB()" to run routines required to be done after changing DB via method DB()
type AfterDBer interface {
	// Runs routines required to be done after changing DB via method DB()
	AfterDB()
}

// Scope represents abstract scopes
type ScopeAbstract interface {
	// Get all entered parameters via method Where() (a slice of sub-slices, while every sub-slice complies to every Where() call)
	GetWhere() [][]interface{}

	// Get all entered parameters via method Order()
	GetOrder() []string

	// Get all entered parameters via method Group()
	GetGroup() []string

	// Get the last entered limit via method Limit()
	GetLimit() int
}
type Scope interface {
	ScopeAbstract

	// Sets all scope-related parameters to be equal as in passed scope (as an argument)
	ISetScope(Scope) Scope

	GetDB() *DB

	Create() error
	Update() error
	Save() error
	Delete() error
}
type GormImitateScope interface {
	ScopeAbstract

	// Sets all scope-related parameters to be equal as in passed scope (as an argument)
	ISetReformScope(GormImitateScope) GormImitateScope

	GetReformDB() *DB

	ReformCreate() error
	ReformUpdate() error
	ReformSave() error
	ReformDelete() error
}

// parseStructFieldTag is used by both file and runtime parsers to parse "reform" tags
func ParseStructFieldTag(tag string) (sqlName string, isPK bool, embedded string) {
	parts := strings.Split(tag, ",")
	if len(parts) == 0 {
		return
	}

	sqlName = parts[0]

	if len(parts) > 1 {
		parts = parts[1:]
		for _, part := range parts {
			subParts := strings.Split(part, ":")
			switch subParts[0] {
			case "pk":
				isPK = true
			case "embedded":
				embedded = subParts[1]
			default:
				// TODO: notify about the error
				return
			}
		}
	}

	return
}

// convert structure field name to table field name using GORM rules
func toGormFieldName(fieldName string) (gormFieldName string) {
	return gorm.ToDBName(fieldName)
}

// parseStructFieldGormTag is the same as parseStructFieldTag() but to parse "gorm" tags (it's for case if option "imitateGorm" is enabled)
func ParseStructFieldGormTag(tag string, fieldName string) (sqlName string, isPK bool, embedded string, structFile string) {
	defer func() {
		if sqlName == "" {
			sqlName = toGormFieldName(fieldName)
		}
	}()

	parts := strings.Split(tag, ";")
	if len(parts) <= 1 {
		isPK = fieldName == "Id"
	}

	if len(parts) < 1 {
		return
	}

	/*sqlName = parts[0]

	if len(parts) < 2 {
		return
	}*/

	for _, part := range parts /*[1:]*/ {
		subParts := strings.Split(part, ":")
		switch subParts[0] {
		case "primary_key":
			isPK = true
		case "column":
			sqlName = subParts[1]
		case "embedded":
			embedded = subParts[1]
		case "file":
			structFile = subParts[1]
		default:
			// TODO: Notify about the error
		}
	}

	return
}

// check interface
var (
	_ DBTX = (*sql.DB)(nil)
	_ DBTX = (*sql.Tx)(nil)
)
