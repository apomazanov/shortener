package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"golang.org/x/tools/go/packages"
)

type kind int

const (
	kindPrimitive kind = iota
	kindSlice
	kindMap
	kindPointerPrimitive
	kindPointerStruct
	kindStruct
)

type field struct {
	Name string
	Type string
	Kind kind
}

type structData struct {
	Name   string
	Fields []field
}

func main() {
	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles |
			packages.NeedSyntax,
		Dir: ".",
	}

	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		slog.Error("Failed to load packages", "error", err)
		os.Exit(1)
	}

	for _, pkg := range pkgs {
		if len(pkg.Errors) > 0 {
			for _, pkgErr := range pkg.Errors {
				slog.Warn("Package warning", "pkg", pkg.PkgPath, "error", pkgErr)
			}
			continue
		}

		processPackage(pkg)
	}
}

func processPackage(pkg *packages.Package) {

	var data []structData
	var dirPath string

	for _, file := range pkg.Syntax {
		fileToken := pkg.Fset.File(file.Pos())
		if fileToken == nil {
			continue
		}

		filename := fileToken.Name()
		if strings.HasSuffix(filename, "_test.go") || strings.HasSuffix(filename, ".gen.go") {
			continue
		}

		// first file fills directory name
		if dirPath == "" {
			dirPath = filepath.Dir(filename)
		}

		ast.Inspect(file, func(n ast.Node) bool {

			// skipping functions with all content
			if _, ok := n.(*ast.FuncDecl); ok {
				return false
			}

			// looking for GenDecl
			if genDecl, ok := n.(*ast.GenDecl); ok {
				// looking for TYPE decl. If other type, skip content
				if genDecl.Tok != token.TYPE {
					return false
				}

				newData := processGenDecl(genDecl)
				data = append(data, newData...)
			}

			return true
		})
	}

	if len(data) > 0 && dirPath != "" {
		if err := generateResetFile(dirPath, pkg.Name, data); err != nil {
			slog.Error("Failed to create file", "dir", dirPath, "error", err)
		}
	}
}

func processGenDecl(genDecl *ast.GenDecl) []structData {
	var data []structData

	isGroupComment := checkResetComment(genDecl.Doc)

	for _, spec := range genDecl.Specs {
		typeSpec, ok := spec.(*ast.TypeSpec)
		if !ok {
			continue
		}

		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			continue
		}

		isStructComment := checkResetComment(typeSpec.Doc)

		if isGroupComment || isStructComment {
			structData := structData{
				Name:   typeSpec.Name.Name,
				Fields: extractFields(structType),
			}

			data = append(data, structData)
		}
	}

	return data
}

func checkResetComment(doc *ast.CommentGroup) bool {
	if doc == nil {
		return false
	}

	return strings.HasPrefix(doc.Text(), "generate:reset")
}

func extractFields(structType *ast.StructType) []field {
	var fields []field

	if structType.Fields == nil {
		return fields
	}

	for _, f := range structType.Fields.List {
		fType, kind := analyzeType(f.Type)

		for _, name := range f.Names {
			fields = append(fields, field{
				Name: name.Name,
				Type: fType,
				Kind: kind,
			})
		}
	}

	return fields
}

func analyzeType(expr ast.Expr) (string, kind) {
	switch t := expr.(type) {
	case *ast.Ident:
		if isBuiltinPrimitive(t.Name) {
			return t.Name, kindPrimitive
		}
		return t.Name, kindStruct
	case *ast.ArrayType:
		return "", kindSlice
	case *ast.MapType:
		return "", kindMap
	case *ast.StarExpr:
		elemType, elemKind := analyzeType(t.X)
		if elemKind == kindPrimitive {
			return elemType, kindPointerPrimitive
		}
		return elemType, kindPointerStruct
	case *ast.SelectorExpr:
		if xIdent, ok := t.X.(*ast.Ident); ok {
			return xIdent.Name + "." + t.Sel.Name, kindStruct
		}
		return t.Sel.Name, kindStruct
	default:
		return "", kindPrimitive
	}
}

func isBuiltinPrimitive(typeName string) bool {
	switch typeName {
	case "string", "bool",
		"int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
		"float32", "float64", "complex64", "complex128", "byte", "rune":
		return true
	default:
		return false
	}
}

type templateData struct {
	PackageName string
	Structs     []structData
}

func generateResetFile(dirPath, pkgName string, structs []structData) error {

	funcMap := template.FuncMap{
		"zeroValue":          getZeroValue,
		"isNumeric":          isNumeric,
		"isPrimitive":        func(k kind) bool { return k == kindPrimitive },
		"isSlice":            func(k kind) bool { return k == kindSlice },
		"isMap":              func(k kind) bool { return k == kindMap },
		"isPointerPrimitive": func(k kind) bool { return k == kindPointerPrimitive },
		"isPointerStruct":    func(k kind) bool { return k == kindPointerStruct },
		"isStruct":           func(k kind) bool { return k == kindStruct },
	}

	tmpl, err := template.New("reset").Funcs(funcMap).Parse(resetTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	data := templateData{
		PackageName: pkgName,
		Structs:     structs,
	}

	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		// В случае ошибки выводим сгенерированный код в лог для отладки
		slog.Error("Generated invalid code", "raw_code", buf.String())
		return fmt.Errorf("formatting failed: %w", err)
	}

	targetPath := filepath.Join(dirPath, "reset.gen.go")
	return os.WriteFile(targetPath, formatted, 0644)
}

func getZeroValue(primitiveType string) string {
	switch primitiveType {
	case "string":
		return `""`
	case "bool":
		return "false"
	default:
		return "0"
	}
}

func isNumeric(typeName string) bool {
	switch typeName {
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
		"float32", "float64", "complex64", "complex128", "byte", "rune":
		return true
	default:
		return false
	}
}

const resetTemplate = `// Code generated by reset generator. DO NOT EDIT.

package {{ .PackageName }}

{{ range .Structs }}
func (rs *{{ .Name }}) Reset() {
	if rs == nil {
		return
	}
	{{ range .Fields }}
		{{ if isPrimitive .Kind }}
			{{ if eq .Type "string" }}
				rs.{{ .Name }} = ""
			{{ else if eq .Type "bool" }}
				rs.{{ .Name }} = false
			{{ else if isNumeric .Type }}
				rs.{{ .Name }} = 0
			{{ else }}
				if v, ok := any(&rs.{{ .Name }}).(interface{ Reset() }); ok {
					v.Reset()
				} else {
					var zero {{ .Type }}
					rs.{{ .Name }} = zero
				}
			{{ end }}
		{{ else if isSlice .Kind }}
			rs.{{ .Name }} = rs.{{ .Name }}[:0]
		{{ else if isMap .Kind }}
			clear(rs.{{ .Name }})
		{{ else if isPointerPrimitive .Kind }}
			if rs.{{ .Name }} != nil {
				*rs.{{ .Name }} = {{ zeroValue .Type }}
			}
		{{ else if isPointerStruct .Kind }}
			if resetter, ok := any(rs.{{ .Name }}).(interface{ Reset() }); ok && rs.{{ .Name }} != nil {
				resetter.Reset()
			}
		{{ else if isStruct .Kind }}
			if resetter, ok := any(&rs.{{ .Name }}).(interface{ Reset() }); ok {
				resetter.Reset()
			} else {
				var zero {{ .Type }}
				rs.{{ .Name }} = zero
			}
		{{ end }}
	{{ end }}
}
{{ end }}
`
