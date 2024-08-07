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

func TestEncoder_Encode(t *testing.T) {
	// Sample data as yaml.Node
	node := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "update"},
			{Kind: yaml.MappingNode, Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Value: "enabled"},
				{Kind: yaml.ScalarNode, Value: "true"},
				{Kind: yaml.ScalarNode, Value: "git"},
				{Kind: yaml.MappingNode, Style: yaml.FlowStyle},
				{Kind: yaml.ScalarNode, Value: "schedule"},
				{Kind: yaml.MappingNode, Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "daily"},
					{Kind: yaml.ScalarNode, Value: "true"},
					{Kind: yaml.ScalarNode, Value: "reason"},
					{Kind: yaml.ScalarNode, Value: "upstream does not maintain tags or releases, it uses a branch"},
				}},
			}},
		},
	}

	// Sample data as user-defined type
	type Schedule struct {
		Daily  bool   `yaml:"daily"`
		Reason string `yaml:"reason"`
	}
	type Update struct {
		Enabled  bool     `yaml:"enabled"`
		Git      struct{} `yaml:"git"`
		Schedule Schedule `yaml:"schedule"`
	}
	type Document struct {
		Update Update `yaml:"update"`
	}
	document := Document{
		Update: Update{
			Enabled: true,
			Schedule: Schedule{
				Daily:  true,
				Reason: "upstream does not maintain tags or releases, it uses a branch",
			},
		},
	}

	// Expected YAML output
	expectedYAML := `update:
  enabled: true
  git: {}
  schedule:
    daily: true
    reason: upstream does not maintain tags or releases, it uses a branch
`

	t.Run("yaml.Node", func(t *testing.T) {
		var outNode bytes.Buffer
		encoderNode := NewEncoder(&outNode)
		err := encoderNode.Encode(node)
		require.NoError(t, err)

		checkDiff(t, expectedYAML, outNode.String())
	})

	t.Run("user-defined type", func(t *testing.T) {
		// Encode user-defined type
		var outStruct bytes.Buffer
		encoderStruct := NewEncoder(&outStruct)
		err := encoderStruct.Encode(document)
		require.NoError(t, err)

		checkDiff(t, expectedYAML, outStruct.String())
	})
}

func checkDiff(t *testing.T, expected, actual any) {
	t.Helper()

	if diff := cmp.Diff(expected, actual); diff != "" {
		t.Errorf(`unexpected document (-want +got):
%s

full expected:

%s

full actual:

%s`, diff, expected, actual)
	}
}
