package yam

import (
	"path/filepath"

	"github.com/chainguard-dev/yam/pkg/rwfs"
)

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

	formatted, err := applyFormatting(file, options)
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
