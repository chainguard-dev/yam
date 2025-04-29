package yam

import (
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/chainguard-dev/yam/pkg/rwfs"
	"github.com/chainguard-dev/yam/pkg/util"
)

func Format(fsys rwfs.FS, paths []string, options FormatOptions) error {
	// "No paths" means "look at all files in the current directory".
	if len(paths) == 0 {
		paths = append(paths, ".")
	}

	for _, p := range paths {
		stat, err := fs.Stat(fsys, p)
		if err != nil {
			return fmt.Errorf("unable to stat %q: %w", p, err)
		}

		if stat.IsDir() {
			errDir := formatDir(fsys, p, options)
			if errDir != nil {
				return fmt.Errorf("unable to format directory %q: %w", p, errDir)
			}

			continue
		}

		err = formatSingleFile(fsys, p, options)
		if err != nil {
			return err
		}
	}

	return nil
}

func formatDir(fsys rwfs.FS, dirPath string, options FormatOptions) error {
	dirEntries, err := fs.ReadDir(fsys, dirPath)
	if err != nil {
		return err
	}

	for _, file := range dirEntries {
		if !file.Type().IsRegular() {
			continue
		}

		p := filepath.Join(dirPath, file.Name())
		errsingleFile := formatSingleFile(fsys, p, options)
		if errsingleFile != nil {
			return errsingleFile
		}
	}

	return nil
}

func formatSingleFile(fsys rwfs.FS, path string, options FormatOptions) error {
	// Immediately skip files that aren't YAML files
	if !util.IsYAML(path) {
		return nil
	}

	p := filepath.Clean(path)
	file, err := fsys.Open(p)
	if err != nil {
		return err
	}

	defer file.Close()

	formatted, err := applyFormatting(file, options)
	if err != nil {
		return fmt.Errorf("unable to format %q: %w", path, err)
	}

	err = fsys.Truncate(p, 0)
	if err != nil {
		return err
	}
	writeableFile, err := fsys.OpenRW(p)
	if err != nil {
		return err
	}

	_, err = formatted.WriteTo(writeableFile)
	if err != nil {
		return err
	}

	_ = writeableFile.Close()

	return nil
}
