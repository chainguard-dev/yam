package yam

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"

	"github.com/chainguard-dev/yam/pkg/rwfs"
	"github.com/chainguard-dev/yam/pkg/yam/formatted"
	"gopkg.in/yaml.v3"
)

type FormatOptions struct {
	Indent         int
	GapExpressions []string
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
		return fmt.Errorf("unable to set gap expression for encoder: %w", err)
	}

	err = enc.Encode(root)
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

	_, err = buf.WriteTo(writeableFile)
	if err != nil {
		return err
	}

	writeableFile.Close()

	return nil
}
