package yam

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/chainguard-dev/yam/pkg/util"
)

var (
	ErrDidNotPassLintCheck = errors.New("input did not pass the lint check")
)

func Lint(fsys fs.FS, paths []string, handler DiffHandler, options FormatOptions) error {
	// "No paths" means "look at all files in the current directory".
	if len(paths) == 0 {
		paths = append(paths, ".")
	}

	var pathsThatFailedLinting []string

	for _, p := range paths {
		stat, err := fs.Stat(fsys, p)
		if err != nil {
			return err
		}

		if stat.IsDir() {
			err := lintDir(fsys, p, handler, options)
			if err != nil {
				var e errLintCheckFailed
				if errors.As(err, &e) {
					pathsThatFailedLinting = append(pathsThatFailedLinting, e.paths...)
					continue
				}

				// An error more serious than lint check failure has occurred.
				return err
			}

			continue
		}

		err = lintSingleFile(fsys, p, handler, options)
		if err != nil {
			if errors.As(err, &errLintCheckFailed{}) {
				pathsThatFailedLinting = append(pathsThatFailedLinting, p)
				continue
			}

			// An error more serious than lint check failure has occurred.
			return err
		}
	}

	if len(pathsThatFailedLinting) >= 1 {
		return ErrDidNotPassLintCheck
	}

	return nil
}

func lintDir(fsys fs.FS, dirPath string, handler DiffHandler, options FormatOptions) error {
	dirEntries, err := fs.ReadDir(fsys, dirPath)
	if err != nil {
		return err
	}

	var dirEntriesThatFailedLinting []string

	for _, file := range dirEntries {
		if !file.Type().IsRegular() {
			continue
		}

		p := filepath.Join(dirPath, file.Name())
		err := lintSingleFile(fsys, p, handler, options)
		if err != nil {
			if errors.As(err, &errLintCheckFailed{}) {
				dirEntriesThatFailedLinting = append(dirEntriesThatFailedLinting, p)
				continue
			}

			return err
		}
	}

	if len(dirEntriesThatFailedLinting) >= 1 {
		return newErrLintCheckFailed(dirEntriesThatFailedLinting...)
	}

	return nil
}

func lintSingleFile(fsys fs.FS, path string, handler DiffHandler, options FormatOptions) error {
	// Immediately skip files that aren't YAML files
	if !util.IsYAML(path) {
		return nil
	}

	cleaned := filepath.Clean(path)
	file, err := fsys.Open(cleaned)
	if err != nil {
		return err
	}

	// Save the original content for comparison after applying the formatting.
	original := new(bytes.Buffer)
	tee := io.TeeReader(file, original)

	defer file.Close()

	formatted, err := applyFormatting(tee, options)
	if err != nil {
		return fmt.Errorf("unable to format %q: %w", path, err)
	}

	want := formatted.Bytes()
	got := original.Bytes()

	if !bytes.Equal(want, got) {
		fmt.Fprintf(os.Stderr, "%s has a diff from the expected formatting\n", path)

		if handler != nil {
			err := handler(want, got)
			if err != nil {
				return fmt.Errorf("unable to handle diff: %w", err)
			}
		}

		return newErrLintCheckFailed(path)
	}

	return nil
}

type errLintCheckFailed struct {
	paths []string
}

func newErrLintCheckFailed(paths ...string) errLintCheckFailed {
	return errLintCheckFailed{paths: paths}
}

func (e errLintCheckFailed) Error() string {
	return fmt.Sprintf("the following was not formatted correctly: %s", strings.Join(e.paths, ", "))
}
