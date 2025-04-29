package path

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
)

var assertErrExpressionNotSupported assert.ErrorAssertionFunc = func(t assert.TestingT, err error, _ ...interface{}) bool {
	return assert.ErrorIs(t, err, ErrExpressionNotSupported)
}

func TestParse(t *testing.T) {
	cases := []struct {
		expression   string
		expectedPath Path
		assertErr    assert.ErrorAssertionFunc
	}{
		{
			expression:   rootExpression,
			expectedPath: Root(),
			assertErr:    assert.NoError,
		},
		{
			expression: ".some-key",
			expectedPath: Path{
				parts: []Part{
					rootPart{},
					mapPart{key: "some-key"},
				},
			},
			assertErr: assert.NoError,
		},
		{
			expression: ".*",
			expectedPath: Path{
				parts: []Part{
					rootPart{},
					mapPart{key: anyKey},
				},
			},
			assertErr: assert.NoError,
		},
		{
			expression: ".[5]",
			expectedPath: Path{
				parts: []Part{
					rootPart{},
					seqPart{index: 5},
				},
			},
			assertErr: assert.NoError,
		},
		{
			expression: ".[]",
			expectedPath: Path{
				parts: []Part{
					rootPart{},
					seqPart{index: anyIndex},
				},
			},
			assertErr: assert.NoError,
		},
		{
			expression: ".A_KEY[].other-key.thing[0]",
			expectedPath: Path{
				parts: []Part{
					rootPart{},
					mapPart{key: "A_KEY"},
					seqPart{index: anyIndex},
					mapPart{key: "other-key"},
					mapPart{key: "thing"},
					seqPart{index: 0},
				},
			},
			assertErr: assert.NoError,
		},
		{
			expression:   "no-leading-dot",
			expectedPath: Path{},
			assertErr:    assertErrExpressionNotSupported,
		},
		{
			expression:   ".someB@Dcharacter$",
			expectedPath: Path{},
			assertErr:    assertErrExpressionNotSupported,
		},
		{
			expression:   ".unmatched-bracket[",
			expectedPath: Path{},
			assertErr:    assertErrExpressionNotSupported,
		},
	}

	for _, tt := range cases {
		t.Run(tt.expression, func(t *testing.T) {
			p, err := Parse(tt.expression)
			tt.assertErr(t, err)

			if diff := cmp.Diff(tt.expectedPath, p, cmp.AllowUnexported(Path{}, rootPart{}, mapPart{}, seqPart{})); diff != "" {
				t.Errorf("got unexpected value from Parse (-want, +got):\n%s", diff)
			}
		})
	}
}
