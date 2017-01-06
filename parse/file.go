package parse

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"regexp"
	"strings"
)

var magicReformComment = regexp.MustCompile(`reform:([0-9A-Za-z_\.]+)`)
var magicReformOptionsComment = regexp.MustCompile(`reformOptions:([0-9A-Za-z_\.,]+)`)

func fileGoType(x ast.Expr) string {
	switch t := x.(type) {
	case *ast.StarExpr:
		return "*" + fileGoType(t.X)
	case *ast.Ident:
		return t.String()
	default:
		panic(fmt.Sprintf("reform: fileGoType: unhandled '%s' (%#v). Please report this bug.", x, x))
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

func parseStructTypeSpec(ts *ast.TypeSpec, str *ast.StructType, imitateGorm bool) (*StructInfo, error) {
	res := &StructInfo{
		Type:         ts.Name.Name,
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

		// getting field name
		var fieldName string
		if len(f.Names) == 0 {
			if imitateGorm {
				fieldName = fileGoType(f.Type)
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
		if imitateGorm {
			column, isPK = parseStructFieldGormTag(tagString, fieldName)
		} else {
			column, isPK = parseStructFieldTag(tagString)
		}
		if column == "" {
			return nil, fmt.Errorf(`reform: %s has field %s with invalid "reform:"/"gorm:" tag value, it is not allowed`, res.Type, f.Type)
		}
		var pkType string
		if isPK {
			pkType = fileGoType(f.Type)
			if strings.HasPrefix(pkType, "*") {
				return nil, fmt.Errorf(`reform: %s has pointer field %s with a primary field tag, it is not allowed`, res.Type, f.Type)
			}
			if res.PKFieldIndex >= 0 {
				return nil, fmt.Errorf(`reform: %s has field %s with primary field tag (first used by %s), it is not allowed`, res.Type, f.Type, res.Fields[res.PKFieldIndex].Name)
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

	if len(res.Fields) == 0 {
		return nil, fmt.Errorf(`reform: %s has no reform-active fields (forgot to add tag "reform:"?), it is not allowed`, res.Type)
	}

	err := checkFields(res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// File parses given file and returns found structs information.
func File(path string) ([]StructInfo, error) {
	// parse file
	fset := token.NewFileSet()
	fileNode, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	// consider only top-level struct type declarations with magic comment
	var res []StructInfo
	for _, decl := range fileNode.Decls {
		// ast.Print(fset, decl)

		gd, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}

		imitateGorm := false

		for _, spec := range gd.Specs {
			// ast.Print(fset, spec)

			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			// magic comment may be attached to "type Foo struct" (TypeSpec)
			// or to "type (" (GenDecl)
			doc := ts.Doc
			if doc == nil && len(gd.Specs) == 1 {
				doc = gd.Doc
			}
			if doc == nil {
				continue
			}

			optsMatches := magicReformOptionsComment.FindStringSubmatch(doc.Text())
			if len(optsMatches) >= 2 {
				switch optsMatches[1] {
				case "imitateGorm":
					imitateGorm = true
				}
			}

			// ast.Print(fset, doc)
			sm := magicReformComment.FindStringSubmatch(doc.Text())
			if len(sm) < 2 {
				continue
			}
			parts := strings.SplitN(sm[1], ".", 2)
			var schema string
			if len(parts) == 2 {
				schema = parts[0]
			}
			table := parts[len(parts)-1]

			str, ok := ts.Type.(*ast.StructType)
			if !ok {
				continue
			}
			if str.Incomplete {
				continue
			}

			// ast.Print(fset, ts)
			s, err := parseStructTypeSpec(ts, str, imitateGorm)
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
