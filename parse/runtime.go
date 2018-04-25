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

		// check for exported name
		if f.PkgPath != "" {
			return nil, fmt.Errorf(`reform: %s has non-exported reform-active field "%s", it is not allowed`, res.Type, fieldName)
		}

		// parse tag and type
		var column string
		var isPK bool
		var embedded string
		var structFile string
		if imitateGorm {
			column, isPK, embedded, structFile = r.ParseStructFieldGormTag(tagString, fieldName)
		} else {
			column, isPK, embedded = r.ParseStructFieldTag(tagString)
			structFile = embedded
		}
		if isPK && (embedded != "") {
			return nil, fmt.Errorf(`reform: %s has field %s that is the primary key and an embedded structure in the same time`, res.Type, f.Type)
		}
		if column == "" && embedded == "" {
			return nil, fmt.Errorf(`reform: %s has field %s with invalid "reform:"/"gorm:" tag value, it is not allowed`, res.Type, fieldName)
		}
		fType := objectGoType(f.Type, t)
		if isPK {
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

		isUnique, hasIndex := parseStructFieldSQLTag(tag.Get("sql"))

		fieldInfo := r.FieldInfo{
			Name:       fieldName,
			IsPK:       isPK,
			IsUnique:   isUnique,
			HasIndex:   hasIndex,
			Type:       fType,
			Column:     prefix+column,
			FieldsPath: fieldsPath,
		}

		if embedded == "" {
			res.Fields = append(res.Fields, fieldInfo)
		} else {
			var nestedFieldsPath []r.FieldInfo
			switch embedded {
			case "embedded":
				nestedFieldsPath = fieldsPath
			case "prefixed":
				nestedFieldsPath = append(fieldsPath, fieldInfo)
			default:
				return nil, fmt.Errorf(`reform: unknown "embedded" value: %v`, embedded)
			}

			structInfo, err := object(f.Type, "", "", imitateGorm, nestedFieldsPath)
			if err != nil {
				return nil, fmt.Errorf(`reform: %s has field %s of type %s. Got error while getting structure information of the object: %s`, res.Type, fieldName, f.Type, structFile, err.Error())
			}
			res.Fields = append(res.Fields, structInfo.Fields...)
		}
		if isPK {
			res.PKFieldIndex = n
		}
		n++
	}

	if err = checkFields(res); err != nil {
		return nil, err
	}

	return
}
