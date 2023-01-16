package yam

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"

	"github.com/chainguard-dev/yam/pkg/yam/formatted"
	"gopkg.in/yaml.v3"
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

	originalBytes, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	file.Close()

	root := &yaml.Node{}
	decoder := yaml.NewDecoder(bytes.NewReader(originalBytes))
	err = decoder.Decode(root)
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)

	enc := formatted.NewEncoder(buf)
	enc.SetIndent(options.Indent)
	err = enc.SetGapExpressions(options.GapExpressions...)
	if err != nil {
		return err
	}
	err = enc.Encode(root)
	if err != nil {
		return err
	}

	want := buf.Bytes()
	got := originalBytes

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
