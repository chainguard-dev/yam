package tester

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/chainguard-dev/yam/pkg/rwfs"
	"github.com/google/go-cmp/cmp"
)

const expectedSuffix = "_expected"

type FS struct {
	fixtures map[string]*testFile
}

func NewFS(fixtures ...string) (*FS, error) {
	fsys := new(FS)

	for _, f := range fixtures {
		originalBytes, err := os.ReadFile(f)
		if err != nil {
			return nil, fmt.Errorf("unable to load fixture %q into new tester.FS: %w", f, err)
		}

		expectedFile := expectedName(f)
		expectedBytes, err := os.ReadFile(expectedFile)
		if err != nil {
			return nil, fmt.Errorf("unable to load fixture %q into new tester.FS: no expected file %q: %w", f, expectedFile, err)
		}

		tf := new(testFile)
		tf.originalRead = bytes.NewBuffer(originalBytes)
		tf.expectedRead = bytes.NewBuffer(expectedBytes)
		tf.writtenBack = new(bytes.Buffer)

		fsys.fixtures = make(map[string]*testFile)
		fsys.fixtures[f] = tf
	}

	return fsys, nil
}

func expectedName(original string) string {
	dir, file := filepath.Split(original)
	parts := strings.SplitN(file, ".", 2)

	expectedFile := strings.Join([]string{parts[0] + expectedSuffix, parts[1]}, ".")
	return filepath.Join(dir, expectedFile)
}

func (fsys *FS) Open(name string) (rwfs.File, error) {
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
		want := tf.expectedRead.Bytes()
		got := tf.writtenBack.Bytes()

		diff := cmp.Diff(want, got)

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

type testFile struct {
	originalRead, expectedRead, writtenBack *bytes.Buffer
}

func (t *testFile) Stat() (fs.FileInfo, error) {
	panic("don't call stat!")
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
