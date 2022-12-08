package impl

//go:generate options-gen -from-struct=Options
type Options struct {
	source         string
	includeStructs []string
	ignoreStructs  []string
	tags           []string
	outPackage     string
	outFilename    string
}
