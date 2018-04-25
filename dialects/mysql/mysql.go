// Package mysql implements reform.Dialect for MySQL.
package mysql

import (
	"github.com/xaionaro/reform"
)

type mysql struct{}

func (mysql) String() string {
	return "mysql"
}

func (mysql) Placeholder(index int) string {
	return "?"
}

func (mysql) Placeholders(start, count int) []string {
	res := make([]string, count)
	for i := 0; i < count; i++ {
		res[i] = "?"
	}
	return res
}

func (mysql) QuoteIdentifier(identifier string) string {
	return "`" + identifier + "`"
}

func (mysql) LastInsertIdMethod() reform.LastInsertIdMethod {
	return reform.LastInsertId
}

func (mysql) SelectLimitMethod() reform.SelectLimitMethod {
	return reform.Limit
}

func (mysql) DefaultValuesMethod() reform.DefaultValuesMethod {
	return reform.EmptyLists
}

func (mysql) ColumnTypeForField(field reform.FieldInfo) string {
	switch field.Type {
	case "time.Time", "extime.Time":
		return "timestamp"
	case "int":
		return "integer"
	case "int64":
		return "bigint"
	case "string":
		return "text"
	default:
		return "text"
	}
}

func (mysql) ColumnDefinitionForField(field reform.FieldInfo) string {
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
			definition += " AUTO_INCREMENT"
		}
	}

	if field.IsUnique {
		panic("Is not implemented, yet")
	}

	if !canBeNull {
		definition += " NOT NULL"
	}

	return definition
}

func (mysql) ColumnDefinitionPostQueryForField(structInfo reform.StructInfo, field reform.FieldInfo) string {
	if field.HasIndex {
		panic("Is not implemented, yet")
	}

	return ""
}

// Dialect implements reform.Dialect for MySQL.
var Dialect mysql

// check interface
var _ reform.Dialect = Dialect
