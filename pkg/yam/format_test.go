package yam

import (
	"testing"

	"github.com/chainguard-dev/yam/pkg/rwfs/tester"
	"github.com/chainguard-dev/yam/pkg/yam/formatted"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testOptions = FormatOptions{
	EncodeOptions: formatted.EncodeOptions{
		Indent: 2,
		GapExpressions: []string{
			".",
		},
		QuoteExpressions: []string{
			".quotes.version",
		},
	},
	TrimTrailingWhitespace: true,
	FinalNewline:           true,
}

var testOptionsWithDedup = FormatOptions{
	EncodeOptions: formatted.EncodeOptions{
		Indent: 2,
		DedupExpressions: []string{
			".fruits",
			".vegetables",
			".mixed",
			".nested_map.nested",
		},
	},
	TrimTrailingWhitespace: true,
	FinalNewline:           true,
}

func Test_formatSingleFile(t *testing.T) {
	cases := []struct {
		fixture string
	}{
		{
			fixture: "testdata/format/simple.yaml",
		},
		{
			fixture: "testdata/format/acl.yaml",
		},
		{
			fixture: "testdata/format/comments.yaml",
		},
		{
			fixture: "testdata/format/whitespace_issues.yaml",
		},
		{
			fixture: "testdata/format/update.yaml",
		},
		{
			fixture: "testdata/format/quotes.yaml",
		},
	}

	for _, tt := range cases {
		t.Run(tt.fixture, func(t *testing.T) {
			fsys, err := tester.NewFS(tt.fixture)
			require.NoError(t, err)

			err = formatSingleFile(fsys, tt.fixture, testOptions)
			assert.NoError(t, err)

			if diff := fsys.Diff(tt.fixture); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func Test_formatSingleFileWithDedup(t *testing.T) {
	t.Run("testdata/format/dedup.yaml", func(t *testing.T) {
		fsys, err := tester.NewFS("testdata/format/dedup.yaml")
		require.NoError(t, err)

		err = formatSingleFile(fsys, "testdata/format/dedup.yaml", testOptionsWithDedup)
		assert.NoError(t, err)

		if diff := fsys.Diff("testdata/format/dedup.yaml"); diff != "" {
			t.Error(diff)
		}
	})
}

func TestFormat(t *testing.T) {
	cases := []struct {
		name  string
		paths []string
	}{
		{
			name: "multiple files and dirs",
			paths: []string{
				"testdata/dir-scenario-1/a.yaml",
				"testdata/dir-scenario-1/subdir2",
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			fsys, err := tester.NewFS("testdata/dir-scenario-1")
			require.NoError(t, err)

			err = Format(fsys, tt.paths, testOptions)
			require.NoError(t, err)

			if diff := fsys.DiffAll(); diff != "" {
				t.Error(diff)
			}
		})
	}
}
