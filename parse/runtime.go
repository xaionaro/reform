package parse

import (
	"fmt"
	r "github.com/xaionaro/reform"
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
func Object(obj interface{}, schema, table string, imitateGorm bool) (res *r.StructInfo, err error) {
	return object(reflect.ValueOf(obj).Elem().Type(), schema, table, imitateGorm, []r.FieldInfo{})
}

func object(t reflect.Type, schema, table string, imitateGorm bool, fieldsPath []r.FieldInfo) (res *r.StructInfo, err error) {
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

	res = &r.StructInfo{
		Type:         t.Name(),
		SQLSchema:    schema,
		SQLName:      table,
		PKFieldIndex: -1,
	}

	var prefix string
	if len(fieldsPath) > 0 {
		prefix = fieldsPath[len(fieldsPath)-1].Column + "__"
	}

	var n int
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)

		// skip if tag "sql" or "reform" is equals to "-"
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
		fieldName := f.Name

		// getting FieldInfo
		fType := objectGoType(f.Type, t)
		fieldInfo := r.FieldInfo{
			Name:       fieldName,
			Type:       fType,
			FieldsPath: fieldsPath,
		}
		fieldInfo.ConsiderTag(imitateGorm, fieldName, tag)

		// check for exported name
		if f.PkgPath != "" {
			return nil, fmt.Errorf(`reform: %s has non-exported reform-active field "%s", it is not allowed`, res.Type, fieldName)
		}

		if fieldInfo.IsPK && (fieldInfo.Embedded != "") {
			return nil, fmt.Errorf(`reform: %s has field %s that is the primary key and an embedded structure in the same time`, res.Type, f.Type)
		}
		if fieldInfo.Column == "" && fieldInfo.Embedded == "" {
			return nil, fmt.Errorf(`reform: %s has field %s with invalid "reform:"/"gorm:" tag value, it is not allowed`, res.Type, fieldName)
		}
		if fieldInfo.IsPK {
			if strings.HasPrefix(fType, "*") {
				return nil, fmt.Errorf(`reform: %s has pointer field %s with "pk" label in "reform:" tag, it is not allowed`, res.Type, fieldName)
			}
			if strings.HasPrefix(fType, "[") {
				return nil, fmt.Errorf(`reform: %s has slice field %s with with "pk" label in "reform:" tag, it is not allowed`, res.Type, f.Name)
			}
			if res.PKFieldIndex >= 0 {
				return nil, fmt.Errorf(`reform: %s has field %s with primary field tag (first used by %s), it is not allowed`, res.Type, fieldName, res.Fields[res.PKFieldIndex].Name)
			}
		}

		fieldInfo.Column = prefix + fieldInfo.Column

		if fieldInfo.Embedded == "" {
			res.Fields = append(res.Fields, fieldInfo)
		} else {
			var nestedFieldsPath []r.FieldInfo
			switch fieldInfo.Embedded {
			case "embedded":
				nestedFieldsPath = fieldsPath
			case "prefixed":
				nestedFieldsPath = append(fieldsPath, fieldInfo)
			default:
				return nil, fmt.Errorf(`reform: unknown "embedded" value: %v`, fieldInfo.Embedded)
			}

			structInfo, err := object(f.Type, "", "", imitateGorm, nestedFieldsPath)
			if err != nil {
				return nil, fmt.Errorf(`reform: %s has field %s of type %s. Got error while getting structure information of the object: %s`, res.Type, fieldName, f.Type, fieldInfo.StructFile, err.Error())
			}
			res.Fields = append(res.Fields, structInfo.Fields...)
		}
		if fieldInfo.IsPK {
			res.PKFieldIndex = n
		}
		n++
	}

	if err = checkFields(res); err != nil {
		return nil, err
	}

	return
}
