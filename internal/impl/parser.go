package impl

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
	"github.com/fatih/structtag"
	"github.com/kazhuravlev/structspec/internal/errorsh"
	"github.com/kazhuravlev/structspec/internal/impl/assets"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/imports"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"
)

var errNotAStruct = errors.New("not a struct")

var (
	structsTpl = template.Must(template.New("structs.go.tpl").Parse(assets.StructsTemplate))
)

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

func parseFiles(source, ignore string) ([]Struct, error) {
	dirEntries, err := os.ReadDir(source)
	if err != nil {
		return nil, errorsh.Wrap(err, "read source directory")
	}

	var structs []Struct
	for _, entry := range dirEntries {
		if entry.IsDir() {
			continue
		}

		if entry.Name() == ignore {
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

func filterStructs(structs []Struct, includedRe, excludedRe []string) []Struct {
	isIncluded := func(string) bool { return true }
	isExcluded := func(string) bool { return false }

	if len(includedRe) != 0 {
		isIncluded = buildIndexFunc(includedRe)
	}
	if len(excludedRe) != 0 {
		isExcluded = buildIndexFunc(excludedRe)
	}

	res := make([]Struct, 0, len(structs))
	for _, structObj := range structs {
		// NOTE: exclude rule has high priority
		if isExcluded(structObj.Name) {
			continue
		}

		if isIncluded(structObj.Name) {
			res = append(res, structObj)
		}
	}

	return res
}

func renderTo(tplData TemplateData, outFilename string) error {
	rendered, err := render(tplData, structsTpl)
	if err != nil {
		return errorsh.Wrap(err, "render output")
	}

	if outFilename == "" {
		fmt.Println(rendered)
		return nil
	}

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

func buildIndexFunc(slice []string) func(string) bool {
	patternMatchers := make([]func(string) bool, len(slice))
	for i := range slice {
		patternMatchers[i] = regexp.MustCompile(slice[i]).MatchString
	}

	return func(s string) bool {
		for _, patternMatcher := range patternMatchers {
			if patternMatcher(s) {
				return true
			}
		}

		return false
	}
}

func adaptTemplateData(structs []Struct, packageName string, tags []string) TemplateData {
	var data []StructData
	for _, structObj := range structs {
		data = append(data, adaptStruct(structObj, tags))
	}

	return TemplateData{
		PackageName: packageName,
		Structs:     data,
	}
}

func adaptStruct(structObj Struct, tags []string) StructData {
	// Collect all fields with all tags
	fieldsByTag := make(map[string][]FieldFromTag)
	for _, field := range structObj.Fields {
		for _, tag := range field.Tags {
			fieldsByTag[tag.Key] = append(fieldsByTag[tag.Key], FieldFromTag{
				Name:  field.Name,
				Value: tag.Name, // TODO: add auto-generation of name?
			})
		}
	}

	// Filter fields by requested tags if it is necessary
	if len(tags) != 0 {
		tagsMap := make(map[string]struct{}, len(tags))
		for _, tagKey := range tags {
			tagsMap[tagKey] = struct{}{}
		}

		for tagKey := range fieldsByTag {
			if _, ok := tagsMap[tagKey]; !ok {
				delete(fieldsByTag, tagKey)
			}
		}
	}

	var structTags []StructTag
	// Adapt collected tags and fields to slice + sort slice
	{
		for tagKey, fields := range fieldsByTag {
			structTags = append(structTags, StructTag{
				Name:   toPublic(tagKey),
				Fields: fields,
			})
		}

		sort.SliceStable(structTags, func(i, j int) bool {
			return structTags[i].Name < structTags[j].Name
		})
	}

	return StructData{
		Name: structObj.Name,
		Tags: structTags,
	}
}

func toPublic(s string) string {
	// NOTE: will not working with non-ascii
	if len(s) <= 1 {
		return strings.ToUpper(s)
	}

	return fmt.Sprintf("%s%s", strings.ToUpper(s[:1]), s[1:])
}

type TemplateData struct {
	PackageName string
	Structs     []StructData
}

type StructData struct {
	Name string
	Tags []StructTag
}

type StructTag struct {
	Name   string // name of tag with public access. ex: Json, Sql, Pg
	Fields []FieldFromTag
}

type FieldFromTag struct {
	Name  string // name of field from Go structure. ex: UserID
	Value string // name of field from tag. ex: user_id
}

type Struct struct {
	Name   string
	Fields []Field
}

type Field struct {
	Name string
	Tags map[string]structtag.Tag
}
