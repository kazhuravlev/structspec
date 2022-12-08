package impl

import (
	"github.com/kazhuravlev/structspec/internal/errorsh"
	"github.com/kazhuravlev/structspec/internal/impl/assets"
	"text/template"
)

var (
	structsTpl = template.Must(template.New("structs.go.tpl").Parse(assets.StructsTemplate))
)

func Generate(opts Options) error {
	if err := opts.Validate(); err != nil {
		return errorsh.Wrap(err, "bad configuration")
	}

	var outPackageName string
	if opts.outPackage != "" {
		outPackageName = opts.outPackage
	} else {
		packageName, err := parsePackageName(opts.source)
		if err != nil {
			return errorsh.Wrap(err, "parse package name from source")
		}

		outPackageName = packageName
	}

	allStructs, err := parseFiles(opts.source)
	if err != nil {
		return errorsh.Wrap(err, "parse files")
	}

	targetStructs := filterStructs(allStructs, opts.includeStructs, opts.ignoreStructs)

	if err := renderTo(targetStructs, outPackageName, opts.outFilename); err != nil {
		return errorsh.Wrap(err, "render output")
	}

	return nil
}
