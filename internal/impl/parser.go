package impl

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
	"github.com/fatih/structtag"
	"github.com/kazhuravlev/structspec/internal/errorsh"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/imports"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

var errNotAStruct = errors.New("not a struct")

func parsePackageName(source string) (string, error) {
	const mode = packages.NeedTypes | packages.NeedTypesInfo | packages.NeedName
	pkgs, err := packages.Load(&packages.Config{Mode: mode}, source)
	if err != nil {
		return "", errorsh.Wrap(err, "load packages")
	}

	if len(pkgs) == 0 {
		return "", errors.New("has no packages in source dir")
	}

	return pkgs[0].Name, nil
}

func parseFiles(source string) ([]Struct, error) {
	dirEntries, err := os.ReadDir(source)
	if err != nil {
		return nil, errorsh.Wrap(err, "read source directory")
	}

	var structs []Struct
	for _, entry := range dirEntries {
		if entry.IsDir() {
			continue
		}

		fileContents, err := os.ReadFile(filepath.Join(source, entry.Name()))
		if err != nil {
			return nil, errorsh.Wrap(err, "cannot read file")
		}

		dstFile, err := decorator.Parse(string(fileContents))
		if err != nil {
			return nil, errorsh.Wrap(err, "cannot parse source file")
		}

		fileStructs, err := parseFile(dstFile)
		if err != nil {
			return nil, errorsh.Wrap(err, "cannot read contexts")
		}

		structs = append(structs, fileStructs...)
	}

	return structs, nil
}

func parseFile(f *dst.File) ([]Struct, error) {
	var errInspect error
	var structs []Struct
	dst.Inspect(f, func(node dst.Node) bool {
		typeDecl, ok := node.(*dst.TypeSpec)
		if !ok {
			return true
		}

		structObj, err := parseStruct(typeDecl)
		if err == nil {
			structs = append(structs, *structObj)
			return true
		}

		if errors.Is(err, errNotAStruct) {
			return true
		}

		errInspect = err
		return false
	})
	if errInspect != nil {
		return nil, errorsh.Wrap(errInspect, "cannot process request")
	}

	return structs, nil
}

func parseStruct(typeDecl *dst.TypeSpec) (*Struct, error) {
	structDecl, ok := typeDecl.Type.(*dst.StructType)
	if !ok {
		return nil, errNotAStruct
	}

	structName := typeDecl.Name

	fields := make([]Field, 0, len(structDecl.Fields.List))
	for _, dstField := range structDecl.Fields.List {
		field, err := parseStructField(dstField)
		if err != nil {
			return nil, errorsh.Wrapf(err, "parse field '%v' of struct '%s'", dstField, structName)
		}

		fields = append(fields, *field)
	}

	return &Struct{
		Name:   structName.Name,
		Fields: fields,
	}, nil
}

func parseStructField(f *dst.Field) (*Field, error) {
	fieldName := f.Names[0].Name

	var tagLine string
	if f.Tag != nil {
		tagLine = strings.Trim(f.Tag.Value, "`")
	}

	tags, err := structtag.Parse(tagLine)
	if err != nil {
		return nil, errorsh.Wrap(err, "cannot parse tag")
	}

	tagsMap := make(map[string]structtag.Tag)
	for _, tag := range tags.Tags() {
		tagsMap[tag.Key] = *tag
	}

	return &Field{
		Name: fieldName,
		Tags: tagsMap,
	}, nil
}

func filterStructs(structs []Struct, included, excluded []string) []Struct {
	includedIndex := slice2map(included)
	excludedIndex := slice2map(excluded)

	res := make([]Struct, 0, len(structs))
	for _, structObj := range structs {
		// NOTE: exclude rule has high priority
		if _, ok := excludedIndex[structObj.Name]; ok {
			continue
		}

		// NOTE: empty `includedIndex` means "include all"
		if len(includedIndex) == 0 {
			res = append(res, structObj)
			continue
		}

		if _, ok := includedIndex[structObj.Name]; ok {
			res = append(res, structObj)
			continue
		}
	}

	return res
}

func renderTo(structs []Struct, packageName, outFilename string) error {
	templateContext := struct {
		PackageName string
		Structs     []Struct
	}{
		PackageName: packageName,
		Structs:     structs,
	}

	rendered, err := render(templateContext, structsTpl)
	if err != nil {
		return errorsh.Wrap(err, "render output")
	}

	if outFilename == "" {
		fmt.Println(rendered)
		return nil
	}

	// TODO: write to file
	if err := writeFile(outFilename, rendered); err != nil {
		return errorsh.Wrap(err, "write output")
	}

	return nil
}

func render(data interface{}, t *template.Template) (string, error) {
	buf := new(bytes.Buffer)
	if err := t.Execute(buf, data); err != nil {
		return "", err
	}

	var result string
	formatted, err := imports.Process("", buf.Bytes(), nil)
	if err != nil {
		return "", errorsh.Wrap(err, "cannot optimize imports")
	}

	result = string(formatted)

	return result, nil
}

func writeFile(filename, body string) error {
	dirPath := filepath.Dir(filename)

	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		if err := os.MkdirAll(dirPath, 0o755); err != nil {
			return errorsh.Wrap(err, "mkdir")
		}
	} else if err != nil {
		return errorsh.Wrap(err, "dir stat")
	}

	if err := os.WriteFile(filename, []byte(body), 0o600); err != nil {
		return errorsh.Wrap(err, "write out file")
	}

	return nil
}

func slice2map(slice []string) map[string]struct{} {
	res := make(map[string]struct{}, len(slice))
	for i := range slice {
		res[slice[i]] = struct{}{}
	}

	return res
}

type Struct struct {
	Name   string
	Fields []Field
}

type Field struct {
	Name string
	Tags map[string]structtag.Tag
}
