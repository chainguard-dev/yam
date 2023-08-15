package formatted

import (
	"bytes"
	"os"
	"testing"

	"github.com/chainguard-dev/yam/pkg/yam/formatted/path"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const (
	sorted   = "- a\n- b\n- c\n"
	unsorted = "- c\n- a\n- b\n"
)

func TestEncoder_AutomaticConfig(t *testing.T) {
	t.Run("gracefully handles missing config file", func(t *testing.T) {
		err := os.Chdir("testdata/empty-dir")
		require.NoError(t, err)

		w := new(bytes.Buffer)

		assert.NotPanics(t, func() {
			_ = NewEncoder(w).AutomaticConfig()
		})
	})
}

func TestSortingSequence(t *testing.T) {
	tests := []struct {
		sortExpression string
		nodePath       string
		want           string
	}{
		{sortExpression: ".sorted", nodePath: ".sorted", want: sorted},
		{sortExpression: ".notsorted", nodePath: ".sorted", want: unsorted},
		{sortExpression: ".sorted", nodePath: ".notsorted", want: unsorted},
	}
	for _, tc := range tests {
		// Create an unsorted sequence.
		node := &yaml.Node{
			Kind: yaml.SequenceNode,
			Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Value: "c"},
				{Kind: yaml.ScalarNode, Value: "a"},
				{Kind: yaml.ScalarNode, Value: "b"},
			},
		}

		var out bytes.Buffer
		encoder := NewEncoder(&out)
		encoder = encoder.SetIndent(2)
		encoder, err := encoder.SetSortExpressions(tc.sortExpression)
		if err != nil {
			t.Fatalf("Failed to SetSortExpressions for %s: %+v", tc.sortExpression, err)
		}
		nodePath, err := path.Parse(tc.nodePath)
		if err != nil {
			t.Fatalf("failed to parse path: %+v", err)
		}
		got, err := encoder.marshalSequence(node, nodePath)
		if err != nil {
			t.Errorf("Failed to marshal sequence: %+v", err)
		}
		if diff := cmp.Diff(tc.want, string(got)); diff != "" {
			t.Errorf("Not sorted: %s", diff)
		}
	}
}
