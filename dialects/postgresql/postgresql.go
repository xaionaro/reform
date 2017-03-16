// Package postgresql implements reform.Dialect for PostgreSQL.
package postgresql

import (
	"strconv"

	"github.com/xaionaro/reform"
)

type postgresql struct{}

func (postgresql) String() string {
	return "postgresql"
}

func (postgresql) Placeholder(index int) string {
	return "$" + strconv.Itoa(index)
}

func (postgresql) Placeholders(start, count int) []string {
	res := make([]string, count)
	for i := 0; i < count; i++ {
		res[i] = "$" + strconv.Itoa(start+i)
	}
	return res
}

func (postgresql) QuoteIdentifier(identifier string) string {
	return `"` + identifier + `"`
}

func (postgresql) LastInsertIdMethod() reform.LastInsertIdMethod {
	return reform.Returning
}

func (postgresql) SelectLimitMethod() reform.SelectLimitMethod {
	return reform.Limit
}

func (postgresql) DefaultValuesMethod() reform.DefaultValuesMethod {
	return reform.DefaultValues
}

func (postgresql) ColumnDefinitionForField(field reform.FieldInfo) string {
	panic("Is not implemented, yet")
	return ""
}

func (postgresql) ColumnDefinitionPostQueryForField(structInfo reform.StructInfo, field reform.FieldInfo) string {
	panic("Is not implemented, yet")
	return ""
}

// Dialect implements reform.Dialect for PostgreSQL.
var Dialect postgresql

// check interface
var _ reform.Dialect = Dialect
