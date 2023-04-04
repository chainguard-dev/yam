package tester

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/chainguard-dev/yam/pkg/rwfs"
	"github.com/google/go-cmp/cmp"
	"github.com/samber/lo"
)

const expectedSuffix = "_expected"
const specialFileContentForSkippingDiff = "# skip"

var expectedSuffixWithYAML = expectedSuffix + ".yaml"

type FS struct {
	fixtures map[string]*testFile
}

func NewFS(fixtures ...string) (*FS, error) {
	realDirFS := os.DirFS(".")
	testerFS := new(FS)

	for _, f := range fixtures {
		stat, err := fs.Stat(realDirFS, f)
		if err != nil {
			return nil, fmt.Errorf("unable to stat file %q: %w", f, err)
		}

		if stat.IsDir() {
			err := fs.WalkDir(realDirFS, f, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}

				if d.Type().IsDir() {
					testerFS.addDir(path)
					return nil
				}

				if d.Type().IsRegular() {
					if strings.HasSuffix(path, expectedSuffixWithYAML) {
						// this is a special file for this tester.FS! Skip.
						return nil
					}

					err := testerFS.addFixtureFileFromOS(path)
					if err != nil {
						return fmt.Errorf("unable to create new tester.FS: %w", err)
					}
				}

				return nil
			})
			if err != nil {
				return nil, fmt.Errorf("unable to walk fixture directory %q: %w", f, err)
			}

			continue
		}

		err = testerFS.addFixtureFileFromOS(f)
		if err != nil {
			return nil, fmt.Errorf("unable to add fixture file %q to new tester.FS: %w", f, err)
		}
	}

	return testerFS, nil
}

func expectedName(original string) string {
	dir, file := filepath.Split(original)
	parts := strings.SplitN(file, ".", 2)

	expectedFile := strings.Join([]string{parts[0] + expectedSuffix, parts[1]}, ".")
	return filepath.Join(dir, expectedFile)
}

func (fsys *FS) Open(name string) (fs.File, error) {
	if f, ok := fsys.fixtures[name]; ok {
		return f, nil
	}

	return nil, os.ErrNotExist
}

func (fsys *FS) OpenRW(name string) (rwfs.File, error) {
	if f, ok := fsys.fixtures[name]; ok {
		return f, nil
	}

	return nil, os.ErrNotExist
}

func (fsys *FS) Truncate(string, int64) error {
	// TODO: decide if there's a reason for anything but a no-op
	return nil
}

func (fsys *FS) Diff(name string) string {
	if tf, ok := fsys.fixtures[name]; ok {
		want := tf.expectedRead
		got := tf.writtenBack

		if want.String() == specialFileContentForSkippingDiff {
			return ""
		}

		diff := cmp.Diff(want.Bytes(), got.Bytes())

		if diff == "" {
			return ""
		}

		return fmt.Sprintf(
			"unexpected result (-want, +got):\n%s\n",
			diff,
		)
	}

	return fmt.Sprintf("unable to find test file %q in tester.FS", name)
}

func (fsys *FS) DiffAll() string {
	fixtureFiles := lo.Keys(fsys.fixtures)
	sort.Strings(fixtureFiles)

	var result string
	for _, ff := range fixtureFiles {
		if fsys.fixtures[ff].isDir {
			continue
		}

		if diff := fsys.Diff(ff); diff != "" {
			result += fmt.Sprintf("\ndiff found for %q:\n", ff)
			result += diff
		}
	}

	return result
}

func (fsys *FS) addTestFile(path string, tf *testFile) {
	if fsys.fixtures == nil {
		fsys.fixtures = make(map[string]*testFile)
	}

	fsys.fixtures[path] = tf
}

func (fsys *FS) addFixtureFileFromOS(path string) error {
	originalBytes, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("unable to load fixture %q into tester.FS: %w", path, err)
	}

	expectedFile := expectedName(path)
	expectedBytes, err := os.ReadFile(expectedFile)
	if err != nil {
		return fmt.Errorf("unable to load fixture %q into tester.FS: no expected file %q: %w", path, expectedFile, err)
	}

	tf := new(testFile)
	tf.originalRead = bytes.NewBuffer(originalBytes)
	tf.expectedRead = bytes.NewBuffer(expectedBytes)
	tf.writtenBack = new(bytes.Buffer)
	tf.path = path

	fsys.addTestFile(path, tf)

	return nil
}

func (fsys *FS) addDir(path string) {
	// For dirs, we'll punt to the real os FS.

	tf := new(testFile)
	tf.isDir = true
	tf.path = path

	fsys.addTestFile(path, tf)
}

type testFile struct {
	path                                    string
	isDir                                   bool
	originalRead, expectedRead, writtenBack *bytes.Buffer
}

func (t *testFile) ReadDir(n int) ([]fs.DirEntry, error) {
	if !t.isDir {
		return nil, fmt.Errorf("not a directory")
	}

	dirEntries, err := os.ReadDir(t.path)
	if err != nil {
		return nil, err
	}

	filteredDirEntries := lo.Filter(dirEntries, func(e os.DirEntry, _ int) bool {
		return !strings.HasSuffix(e.Name(), expectedSuffixWithYAML)
	})

	return filteredDirEntries, nil
}

func (t *testFile) Stat() (fs.FileInfo, error) {
	return os.Stat(t.path)
}

func (t *testFile) Read(p []byte) (int, error) {
	return t.originalRead.Read(p)
}

func (t *testFile) Close() error {
	return nil
}

func (t *testFile) Write(p []byte) (n int, err error) {
	return t.writtenBack.Write(p)
}
