package parse

import (
	"fmt"
	r "github.com/xaionaro/reform"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"regexp"
	"strings"
)

var magicReformComment = regexp.MustCompile(`reform:([0-9A-Za-z_\.]+)`)
var magicReformOptionsComment = regexp.MustCompile(`reformOptions:([0-9A-Za-z_\.,]+)`)

func fileGoType(x ast.Expr, printOnError ...interface{}) string {
	switch t := x.(type) {
	case *ast.SliceExpr:
		return "[]" + fileGoType(t.X, printOnError...)
	case *ast.StarExpr:
		return "*" + fileGoType(t.X, printOnError...)
	case *ast.SelectorExpr:
		return fileGoType(t.X, printOnError...) + "." + t.Sel.String()
	case *ast.Ident:
		s := t.String()
		if s == "byte" {
			return "uint8"
		}
		return s
	case *ast.ArrayType:
		return "[" + fileGoType(t.Len, printOnError...) + "]" + fileGoType(t.Elt, printOnError...)
	case *ast.BasicLit:
		return t.Value
	case nil:
		return ""
	default:
		panic(fmt.Sprintf("reform: fileGoType: unhandled '%s'/'%T' (%#v: %v, %v). Please report this bug. Additional info: %v", x, x, x, x.Pos(), x.End(), printOnError))
	}
}

func getFieldTag(f *ast.Field) reflect.StructTag {
	if f.Tag != nil {
		tag := f.Tag.Value
		if len(tag) >= 3 {
			return reflect.StructTag(tag[1 : len(tag)-1]) // strip quotes
		}
	}

	return reflect.StructTag("")
}

func parseStructTypeSpec(ts *ast.TypeSpec, str *ast.StructType, imitateGorm bool, fieldsPath []r.FieldInfo, forceParse bool) (*r.StructInfo, error) {
	var prefix string
	if len(fieldsPath) > 0 {
		prefix = fieldsPath[len(fieldsPath)-1].Column + "__"
	}

	var typeName string
	if ts != nil {
		typeName = ts.Name.Name
	}

	res := &r.StructInfo{
		Type:         typeName,
		PKFieldIndex: -1,
	}

	var n int
	for _, f := range str.Fields.List {
		// skip if tag "sql" is equals to "-"
		tag := getFieldTag(f)
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

		// getting type
		fType := fileGoType(f.Type, typeName, *f, fieldsPath)

		// getting field name
		var fieldName string
		if len(f.Names) == 0 {
			if imitateGorm {
				fieldName = fType
			} else {
				return nil, fmt.Errorf(`reform: %s has reform-active anonymous field "%s", it is not allowed`, res.Type, f.Type)
			}

			// check for exported name
			fieldNameFirstCharacter := fieldName[0:1]
			if fieldNameFirstCharacter != strings.ToUpper(fieldNameFirstCharacter) {
				return nil, fmt.Errorf(`reform: %s has non-exported reform-active field "%s", it is not allowed`, res.Type, f.Type)
			}
		} else {
			fieldName = f.Names[0].Name

			// check for exported name
			if !f.Names[0].IsExported() {
				return nil, fmt.Errorf(`reform: %s has non-exported reform-active field "%s", it is not allowed`, res.Type, f.Type)
			}

			if len(f.Names) > 1 {
				panic(fmt.Sprintf("reform: %d names: %#v. Please report this bug.", len(f.Names), f.Names))
			}
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
			return nil, fmt.Errorf(`reform: %s has field %s (of type %s) that is the primary key and an embedded structure in the same time`, res.Type, fieldName, f.Type)
		}
		if column == "" {
			return nil, fmt.Errorf(`reform: %s has field %s (of type %s) with invalid "reform:"/"gorm:" tag value, it is not allowed`, res.Type, fieldName, f.Type)
		}

		if isPK {
			if strings.HasPrefix(fType, "*") {
				return nil, fmt.Errorf(`reform: %s has pointer field %s (of type %s) with a primary field tag, it is not allowed`, res.Type, fieldName, f.Type)
			}
			if strings.HasPrefix(fType, "[") {
				return nil, fmt.Errorf(`reform: %s has slice field %s with with "pk" label in "reform:" tag, it is not allowed`, res.Type, name.Name)
			}
			if res.PKFieldIndex >= 0 {
				return nil, fmt.Errorf(`reform: %s has field %s (of type %s) with primary field tag (first used by %s), it is not allowed`, res.Type, fieldName, f.Type, res.Fields[res.PKFieldIndex].Name)
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
			if structFile == "" {
				return nil, fmt.Errorf(`reform: %s has field %s of type %s but the file with the referenced structure is not set`, res.Type, fieldName, f.Type)
			}

			ident := f.Type.(*ast.Ident)

			var nestedFieldsPath []r.FieldInfo
			switch embedded {
			case "embedded":
				nestedFieldsPath = fieldsPath
			case "prefixed":
				nestedFieldsPath = append(fieldsPath, fieldInfo)
			default:
				return nil, fmt.Errorf(`reform: unknown "embedded" value: %v`, embedded)
			}
			structInfos, err := file(structFile, &imitateGorm, nestedFieldsPath, true)
			if err != nil {
				return nil, fmt.Errorf(`reform: %s has field %s of type %s that uses file %s. Got error while parsing the file: %s`, res.Type, fieldName, f.Type, structFile, err.Error())
			}

			found := false
			for _, structInfo := range structInfos {
				if structInfo.Type == ident.String() {
					nestedFields := structInfo.Fields
					//fmt.Printf("nestedFields == %v\n", nestedFields)
					res.Fields = append(res.Fields, nestedFields...)
					found = true
					break
				}
			}
			if !found {
				return nil, fmt.Errorf(`reform: %s has field %s that references to file %s, but the file doesn't have a structure %s`, res.Type, fieldName, structFile, f.Type)
			}
		}
		if isPK {
			res.PKFieldIndex = n
		}
		n++
	}

	if forceParse {	// TODO: Re-enable checkes and error reporting for forceParse == true
		return res, nil
	}

	if len(res.Fields) == 0 {
		return nil, fmt.Errorf(`reform: %s has no reform-active fields (forgot to add tag "reform:"?), it is not allowed`, res.Type)
	}

	if err := checkFields(res); err != nil {
		return nil, err
	}

	return res, nil
}

// File parses given file and returns found structs information.
func File(path string) ([]r.StructInfo, error) {
	return file(path, nil, []r.FieldInfo{}, false)
}

func file(path string, forceImitateGorm *bool, fieldsPath []r.FieldInfo, forceParse bool) ([]r.StructInfo, error) {
	// parse file
	fset := token.NewFileSet()
	fileNode, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	// consider only top-level struct type declarations with magic comment
	var res []r.StructInfo
	for _, decl := range fileNode.Decls {
		// ast.Print(fset, decl)

		gd, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}

		imitateGorm := false

		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			doc := ts.Doc
			if doc == nil && len(gd.Specs) == 1 {
				doc = gd.Doc
			}
			// magic comment may be attached to "type Foo struct" (TypeSpec)
			// or to "type (" (GenDecl)

			if !forceParse {
				if doc == nil {
					continue
				}
			}

			if doc != nil {
				optsMatches := magicReformOptionsComment.FindStringSubmatch(doc.Text())
				if len(optsMatches) >= 2 {
					switch optsMatches[1] {
					case "imitateGorm":
						imitateGorm = true
					}
				}
			}

			var sm []string
			if doc != nil {
				sm = magicReformComment.FindStringSubmatch(doc.Text())
			}
			if !forceParse {
				if len(sm) < 2 {
					continue
				}
			}
			var parts []string
			if len(sm) >= 2 {
				parts = strings.SplitN(sm[1], ".", 2)
			}
			var schema string
			if len(parts) == 2 {
				schema = parts[0]
			}
			var table string
			if len(parts) >= 1 {
				table = parts[len(parts)-1]
			}

			str, ok := ts.Type.(*ast.StructType)
			if !ok {
				continue
			}
			if str.Incomplete {
				continue
			}

			if forceImitateGorm != nil {
				imitateGorm = *forceImitateGorm
			}

			// ast.Print(fset, ts)
			s, err := parseStructTypeSpec(ts, str, imitateGorm, fieldsPath, forceParse)
			if err != nil {
				return nil, err
			}
			s.SQLSchema = schema
			s.SQLName = table
			s.ImitateGorm = imitateGorm
			res = append(res, *s)
		}
	}

	return res, nil
}
