// Package parse implements parsing of Go structs in files and runtime.
//
// This package, despite containing exported types, methods and functions,
// is an internal part of implementation of 'reform' command, also used by generated files,
// and not a part of public stable API.
package parse

import (
	"fmt"
	"github.com/jinzhu/gorm"
	"reflect"
	"strings"
)

// FieldInfo represents information about struct field.
type FieldInfo struct {
	Name   string // field name as defined in source file, e.g. Name
	PKType string // primary key field type as defined in source file, e.g. string
	Column string // SQL database column name from "reform:" struct field tag, e.g. name
}

// StructInfo represents information about struct.
type StructInfo struct {
	Type         string      // struct type as defined in source file, e.g. User
	SQLSchema    string      // SQL database schema name from magic "reform:" comment, e.g. public
	SQLName      string      // SQL database view or table name from magic "reform:" comment, e.g. users
	Fields       []FieldInfo // fields info
	PKFieldIndex int         // index of primary key field in Fields, -1 if none
	ImitateGorm  bool        // act like GORM (https://github.com/jinzhu/gorm)
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
		FieldInfo{Name: "LogAuthor", PKType: "*string", Column: "log_author"},
		FieldInfo{Name: "LogAction", PKType: "string", Column: "log_action"},
		FieldInfo{Name: "LogDate", PKType: "time.Time", Column: "log_date"},
		FieldInfo{Name: "LogComment", PKType: "string", Column: "log_comment"},
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

// AssertUpToDate checks that given StructInfo matches given object.
// It is used during program initialization to check that generated files are up-to-date.
func AssertUpToDate(si *StructInfo, obj interface{}) {
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

// parseStructFieldTag is used by both file and runtime parsers
func parseStructFieldTag(tag string) (sqlName string, isPK bool) {
	parts := strings.Split(tag, ",")
	if len(parts) == 0 || len(parts) > 2 {
		return
	}

	if len(parts) == 2 {
		switch parts[1] {
		case "pk":
			isPK = true
		default:
			return
		}
	}

	sqlName = parts[0]
	return
}

// convert structure field name to table field name using GORM rules
func toGormFieldName(fieldName string) (gormFieldName string) {
	return gorm.ToDBName(fieldName)
}

// parseStructFieldGormTag is the same as parseStructFieldTag() but for case if option "imitateGorm" is enabled
func parseStructFieldGormTag(tag string, fieldName string) (sqlName string, isPK bool) {
	defer func() {
		if sqlName == "" {
			sqlName = toGormFieldName(fieldName)
		}
	}()

	parts := strings.Split(tag, ",")
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
		switch part {
		case "primary_key":
			isPK = true
		default:
			// TODO: Notify about the error
		}
	}

	return
}

// checkFields is used by both file and runtime parsers
func checkFields(res *StructInfo) error {
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
