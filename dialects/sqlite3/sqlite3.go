// Package sqlite3 implements reform.Dialect for SQLite3.
package sqlite3

import (
	"github.com/xaionaro/reform"
)

type sqlite3 struct{}

func (sqlite3) String() string {
	return "sqlite3"
}

func (sqlite3) Placeholder(index int) string {
	return "?"
}

func (sqlite3) Placeholders(start, count int) []string {
	res := make([]string, count)
	for i := 0; i < count; i++ {
		res[i] = "?"
	}
	return res
}

func (sqlite3) QuoteIdentifier(identifier string) string {
	return `"` + identifier + `"`
}

func (sqlite3) LastInsertIdMethod() reform.LastInsertIdMethod {
	return reform.LastInsertId
}

func (sqlite3) SelectLimitMethod() reform.SelectLimitMethod {
	return reform.Limit
}

func (sqlite3) DefaultValuesMethod() reform.DefaultValuesMethod {
	return reform.DefaultValues
}

func (sqlite3) ColumnTypeForField(field reform.FieldInfo) string {
	switch field.Type {
	case "time.Time", "extime.Time":
		return "datetime"
	case "int":
		return "integer"
	case "string":
		return "text"
	default:
		return "text"
	}
}

func (sqlite3) ColumnDefinitionForField(field reform.FieldInfo) string {
	canBeNull := false
	fieldType := field.Type
	if fieldType[0:1] == "*" {
		canBeNull = true
		fieldType = fieldType[1:]
	}

	columnType := Dialect.ColumnTypeForField(field)

	definition := field.Column + " " + columnType // TODO: escape everything

	if field.IsPK {
		definition += " PRIMARY KEY"
		if fieldType == "int" {
			definition += " AUTOINCREMENT"
		}
	}

	if field.IsUnique {
		definition += " UNIQUE"
	}

	if !canBeNull {
		definition += " NOT NULL"
	}

	return definition
}

func (sqlite3) ColumnDefinitionPostQueryForField(structInfo reform.StructInfo, field reform.FieldInfo) string {
	if field.HasIndex {
		return "CREATE INDEX IF NOT EXISTS idx_" + structInfo.SQLName + "_" + field.Column + " ON " + structInfo.SQLName + "(" + field.Column + ")" // TODO: escape everything
	}

	return ""
}

// Dialect implements reform.Dialect for SQLite3.
var Dialect sqlite3

// check interface
var _ reform.Dialect = Dialect
