package yam

import (
	"path/filepath"

	"github.com/chainguard-dev/yam/pkg/rwfs"
)

type FormatOptions struct {
	// Indent specifies how many spaces to use per-indentation
	Indent int

	// GapExpressions specifies a list of yq-style paths for which the path's YAML
	// element's children elements should be separated by an empty line
	GapExpressions []string

	// FinalNewline specifies whether to ensure the input has a final newline before
	// further formatting is applied.
	FinalNewline bool

	// TrimTrailingWhitespace specifies whether to trim any trailing space
	// characters from each line before further formatting is applied.
	TrimTrailingWhitespace bool
}

func Format(fsys rwfs.FS, paths []string, options FormatOptions) error {
	for _, path := range paths {
		err := formatPath(fsys, path, options)
		if err != nil {
			// TODO: format all paths, not just up until the first error
			return err
		}
	}

	return nil
}

func formatPath(fsys rwfs.FS, path string, options FormatOptions) error {
	p := filepath.Clean(path)
	file, err := fsys.Open(p)
	if err != nil {
		return err
	}

	defer file.Close()

	formatted, err := apply(file, options)
	if err != nil {
		return err
	}

	err = fsys.Truncate(p, 0)
	if err != nil {
		return err
	}
	writeableFile, err := fsys.Open(p)
	if err != nil {
		return err
	}

	_, err = formatted.WriteTo(writeableFile)
	if err != nil {
		return err
	}

	writeableFile.Close()

	return nil
}
