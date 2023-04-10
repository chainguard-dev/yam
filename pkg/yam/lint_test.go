package yam

import (
	"testing"

	"github.com/chainguard-dev/yam/pkg/rwfs/os"
	"github.com/chainguard-dev/yam/pkg/yam/formatted"
	"github.com/stretchr/testify/assert"
)

func didNotPassLintCheck(t assert.TestingT, err error, i ...interface{}) bool {
	return assert.ErrorIs(t, err, ErrDidNotPassLintCheck)
}

func TestLint(t *testing.T) {
	cases := []struct {
		name      string
		paths     []string
		opts      FormatOptions
		assertErr assert.ErrorAssertionFunc
	}{
		{
			name:  "single correct file",
			paths: []string{"fruits.yaml"},
			opts: FormatOptions{
				EncodeOptions: formatted.EncodeOptions{
					Indent:         4,
					GapExpressions: []string{".fruits"},
				},
				FinalNewline:           true,
				TrimTrailingWhitespace: true,
			},
			assertErr: assert.NoError,
		},
		{
			name:  "single incorrect file",
			paths: []string{"vegetables.yaml"},
			opts: FormatOptions{
				EncodeOptions: formatted.EncodeOptions{
					Indent:         4,
					GapExpressions: []string{".vegetables"},
				},
				FinalNewline:           true,
				TrimTrailingWhitespace: true,
			},
			assertErr: didNotPassLintCheck,
		},
		{
			name:  "directory with correct files",
			paths: []string{"all-correct-files"},
			opts: FormatOptions{
				EncodeOptions: formatted.EncodeOptions{
					Indent:         4,
					GapExpressions: []string{".fruits", ".vegetables"},
				},
				FinalNewline:           true,
				TrimTrailingWhitespace: true,
			},
			assertErr: assert.NoError,
		},
		{
			name:  "directory with an incorrect file",
			paths: []string{"has-incorrect-file"},
			opts: FormatOptions{
				EncodeOptions: formatted.EncodeOptions{
					Indent:         2,
					GapExpressions: []string{".fruits", ".vegetables", ".candies.colorful"},
				},
				FinalNewline:           true,
				TrimTrailingWhitespace: true,
			},
			assertErr: didNotPassLintCheck,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			fsys := os.DirFS("testdata/lint")

			err := Lint(fsys, tt.paths, ExecDiff, tt.opts)
			tt.assertErr(t, err)
		})
	}
}
