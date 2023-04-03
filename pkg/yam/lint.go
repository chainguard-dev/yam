package yam

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
)

func Lint(fsys fs.FS, paths []string, handler DiffHandler, options FormatOptions) error {
	for _, path := range paths {
		err := lintPath(fsys, path, handler, options)
		if err != nil {
			// TODO: lint all paths, not just up until the first error
			return err
		}
	}

	return nil
}

func lintPath(fsys fs.FS, path string, handler DiffHandler, options FormatOptions) error {
	cleaned := filepath.Clean(path)
	file, err := fsys.Open(cleaned)
	if err != nil {
		return err
	}

	// Save the original content for comparison after applying the formatting.
	original := new(bytes.Buffer)
	tee := io.TeeReader(file, original)

	defer file.Close()

	formatted, err := apply(tee, options)
	if err != nil {
		return err
	}

	want := formatted.Bytes()
	got := original.Bytes()

	if !bytes.Equal(want, got) {
		if handler != nil {
			err := handler(want, got)
			if err != nil {
				return fmt.Errorf("unable to handle diff: %w", err)
			}
		}

		return fmt.Errorf("%s is not formatted correctly", path)
	}

	return nil
}
