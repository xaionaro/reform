package parse

import (
	"fmt"
	"reflect"
	"strings"
)

func objectGoType(t reflect.Type, structT reflect.Type) string {
	switch t.Kind() {
	case reflect.Ptr:
		return "*" + objectGoType(t.Elem(), structT)
	}

	s := t.String()

	// drop package name from qualified identifier if type is defined in the same package
	if strings.Contains(s, ".") && t.PkgPath() == structT.PkgPath() {
		s = strings.Join(strings.Split(s, ".")[1:], ".")
	}

	return s
}

// Object extracts struct information from given object.
func Object(obj interface{}, schema, table string, imitateGorm bool) (res *StructInfo, err error) {
	// convert any panic to error
	defer func() {
		p := recover()
		switch p := p.(type) {
		case error:
			err = p
		case nil:
			// nothing
		default:
			err = fmt.Errorf("%s", p)
		}
	}()

	t := reflect.ValueOf(obj).Elem().Type()
	res = &StructInfo{
		Type:         t.Name(),
		SQLSchema:    schema,
		SQLName:      table,
		PKFieldIndex: -1,
	}

	var n int
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)

		// skip if tag "sql" is equals to "-"
		tag := f.Tag
		if tag.Get("sql") == "-" || tag.Get("reform") == "-" {
			continue
		}

		var tagString string
		if imitateGorm {
			// consider tag "gorm:" if is set
			tagString = tag.Get("gorm")
		} else {
			// consider only fields with "reform:" tag
			tagString = tag.Get("reform")
			if len(tagString) == 0 {
				continue
			}
		}

		// getting field name
		var fieldName string
		if f.Anonymous {
			if imitateGorm {
				fieldName = f.Name
			} else {
				return nil, fmt.Errorf(`reform: %s has reform-active anonymous field "%s", it is not allowed`, res.Type, f.Name)
			}
		} else {
			fieldName = f.Name
		}

		// check for exported name
		if f.PkgPath != "" {
			return nil, fmt.Errorf(`reform: %s has non-exported reform-active field "%s", it is not allowed`, res.Type, fieldName)
		}

		// parse tag and type
		var column string
		var isPK bool
		if imitateGorm {
			column, isPK = parseStructFieldGormTag(tagString, fieldName)
		} else {
			column, isPK = parseStructFieldTag(tagString)
		}
		if column == "" {
			return nil, fmt.Errorf(`reform: %s has field %s with invalid "reform:"/"gorm:" tag value, it is not allowed`, res.Type, fieldName)
		}
		var pkType string
		if isPK {
			pkType = objectGoType(f.Type, t)
			if strings.HasPrefix(pkType, "*") {
				return nil, fmt.Errorf(`reform: %s has pointer field %s with a primary field tag, it is not allowed`, res.Type, fieldName)
			}
			if res.PKFieldIndex >= 0 {
				return nil, fmt.Errorf(`reform: %s has field %s with primary field tag (first used by %s), it is not allowed`, res.Type, fieldName, res.Fields[res.PKFieldIndex].Name)
			}
		}

		res.Fields = append(res.Fields, FieldInfo{
			Name:   fieldName,
			PKType: pkType,
			Column: column,
		})
		if isPK {
			res.PKFieldIndex = n
		}
		n++
	}

	err = checkFields(res)
	if err != nil {
		return nil, err
	}

	return
}
