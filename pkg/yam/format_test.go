package yam

import (
	"testing"

	"github.com/chainguard-dev/yam/pkg/rwfs/tester"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_formatPath(t *testing.T) {
	cases := []struct {
		fixture string
	}{
		{
			fixture: "testdata/format/simple.yaml",
		},
		{
			fixture: "testdata/format/acl.yaml",
		},
	}

	options := FormatOptions{
		Indent: 2,
		GapExpressions: []string{
			".",
		},
	}

	for _, tt := range cases {
		t.Run(tt.fixture, func(t *testing.T) {
			fsys, err := tester.NewFS(tt.fixture)
			require.NoError(t, err)

			err = formatPath(fsys, tt.fixture, options)
			assert.NoError(t, err)

			if diff := fsys.Diff(tt.fixture); diff != "" {
				t.Errorf(diff)
			}
		})
	}
}
