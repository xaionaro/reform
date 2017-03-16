// Package parse implements parsing of Go structs in files and runtime.
//
// This package, despite containing exported types, methods and functions,
// is an internal part of implementation of 'reform' command, also used by generated files,
// and not a part of public stable API.
package parse

import (
	"fmt"
	"github.com/jinzhu/gorm"
	r "github.com/xaionaro/reform"
	"reflect"
	"strings"
)

// AssertUpToDate checks that given StructInfo matches given object.
// It is used during program initialization to check that generated files are up-to-date.
func AssertUpToDate(si *r.StructInfo, obj interface{}) {
	msg := fmt.Sprintf(`reform:
		%s struct information is not up-to-date.
		Typically this means that %s type definition was changed, but 'reform' command / 'go generate' was not run.

		`, si.Type, si.Type)
	si2, err := Object(obj, si.SQLSchema, si.SQLName, si.ImitateGorm)
	if err != nil {
		panic(msg + err.Error())
	}
	if !reflect.DeepEqual(si, si2) {
		panic(msg)
	}
}

// parseStructFieldTag is used by both file and runtime parsers to parse "reform" tags
func parseStructFieldTag(tag string) (sqlName string, isPK bool, embedded string) {
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
func parseStructFieldGormTag(tag string, fieldName string) (sqlName string, isPK bool, embedded string, structFile string) {
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

	for _, part := range parts/*[1:]*/ {
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

// parseStructFieldSQLTag is used by both file and runtime parsers to parse "sql" tags
func parseStructFieldSQLTag(tag string) (isUnique bool, hasIndex bool) {
	parts := strings.Split(tag, ",")

	for _, part := range parts {
		switch part {
		case "unique_index":
			isUnique = true
		case "index":
			hasIndex = true
		default:
			// TODO: notify about the error
		}
	}

	return
}

// checkFields is used by both file and runtime parsers
func checkFields(res *r.StructInfo) error {
	if len(res.Fields) == 0 {
		return fmt.Errorf(`reform: %s has no reform-active fields (forgot to set tags "reform:"?), it is not allowed`, res.Type)
	}

	dupes := make(map[string]string)
	for _, f := range res.Fields {
		if f2, ok := dupes[f.Column]; ok {
			return fmt.Errorf(`reform: %s has reform-active field %s with duplicate column name %s (used by %s), it is not allowed`,
				res.Type, f.Name, f.Column, f2)
		}
		dupes[f.Column] = f.Name
	}

	return nil
}

