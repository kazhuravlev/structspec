package main

import (
	"github.com/kazhuravlev/structspec/internal/impl"
	"github.com/urfave/cli/v3"
	"os"
)

const (
	argSrcDir         = "src"
	argFilesPattern   = "files"
	argIncludeStructs = "structs"
	argIgnoreStructs  = "ignore"
	argTag            = "tag"
	argOutFilename    = "out-file"
	argOutPackage     = "out-pkg"
)

var (
	// TODO: get from CI
	version = "__from_source__"
)

func main() {
	a := cli.NewApp()
	a.Version = version
	a.Name = "structspec"
	a.Usage = "Generate structs specification. github.com/kazhuravlev/structspec"
	a.Commands = []*cli.Command{
		{
			Name:   "gen",
			Usage:  "Generate and write result",
			Action: cmdGenerate,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     argSrcDir,
					Value:    "",
					Usage:    "Source directory",
					Required: true,
				},
				&cli.StringSliceFlag{
					Name:     argFilesPattern,
					Value:    cli.NewStringSlice("*"),
					Usage:    "Include only specified files (glob syntax)",
					Required: false,
				},
				&cli.StringSliceFlag{
					Name:     argIncludeStructs,
					Value:    nil,
					Usage:    "Which structs should be included. Default: all founded",
					Required: false,
				},
				&cli.StringSliceFlag{
					Name:     argIgnoreStructs,
					Value:    nil,
					Usage:    "Which structs should be ignored. Default: no one",
					Required: false,
				},
				&cli.StringSliceFlag{
					Name:     argTag,
					Value:    nil,
					Usage:    "Which tags should be used for generation. Default: all founded",
					Required: false,
				},
				&cli.StringFlag{
					Name:     argOutFilename,
					Value:    "",
					Usage:    "Output filename",
					Required: false,
				},
				&cli.StringFlag{
					Name:     argOutPackage,
					Value:    "",
					Usage:    "Output package name",
					Required: false,
				},
			},
		},
	}

	if err := a.Run(os.Args); err != nil {
		panic("cannot run command: " + err.Error())
	}
}

func cmdGenerate(c *cli.Context) error {
	sourceDirectory := c.String(argSrcDir)
	includedFiles := c.StringSlice(argFilesPattern)
	outFilename := c.String(argOutFilename)
	outPackage := c.String(argOutPackage)
	includeStructs := c.StringSlice(argIncludeStructs)
	ignoreStructs := c.StringSlice(argIgnoreStructs)
	tags := c.StringSlice(argTag)

	err := impl.Generate(impl.NewOptions(
		impl.WithSource(sourceDirectory),
		impl.WithIncludedFiles(includedFiles),
		impl.WithOutFilename(outFilename),
		impl.WithOutPackage(outPackage),
		impl.WithIncludeStructs(includeStructs),
		impl.WithIgnoreStructs(ignoreStructs),
		impl.WithTags(tags),
	))
	if err != nil {
		return err
	}

	return nil
}
