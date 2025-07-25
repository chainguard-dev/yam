package formatted

import (
	"bytes"
	"strings"
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
		t.Chdir("testdata/empty-dir")

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

func TestDedupSequence(t *testing.T) {
	tests := []struct {
		name            string
		dedupExpression string
		nodePath        string
		inputValues     []string
		want            string
	}{
		{
			name:            "dedup enabled for matching path",
			dedupExpression: ".fruits",
			nodePath:        ".fruits",
			inputValues:     []string{"apple", "banana", "apple", "orange", "banana"},
			want:            "- apple\n- banana\n- orange\n",
		},
		{
			name:            "dedup disabled for non-matching path",
			dedupExpression: ".vegetables",
			nodePath:        ".fruits",
			inputValues:     []string{"apple", "banana", "apple", "orange", "banana"},
			want:            "- apple\n- banana\n- apple\n- orange\n- banana\n",
		},
		{
			name:            "dedup with no duplicates",
			dedupExpression: ".fruits",
			nodePath:        ".fruits",
			inputValues:     []string{"apple", "banana", "orange"},
			want:            "- apple\n- banana\n- orange\n",
		},
		{
			name:            "dedup with all duplicates",
			dedupExpression: ".fruits",
			nodePath:        ".fruits",
			inputValues:     []string{"apple", "apple", "apple"},
			want:            "- apple\n",
		},
		{
			name:            "dedup with empty sequence",
			dedupExpression: ".fruits",
			nodePath:        ".fruits",
			inputValues:     []string{},
			want:            "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var contentNodes []*yaml.Node
			for _, value := range tc.inputValues {
				contentNodes = append(contentNodes, &yaml.Node{Kind: yaml.ScalarNode, Value: value})
			}

			node := &yaml.Node{
				Kind:    yaml.SequenceNode,
				Content: contentNodes,
			}

			var out bytes.Buffer
			encoder := NewEncoder(&out)
			encoder = encoder.SetIndent(2)
			encoder, err := encoder.SetDedupExpressions(tc.dedupExpression)
			if err != nil {
				t.Fatalf("Failed to SetDedupExpressions for %s: %+v", tc.dedupExpression, err)
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
				t.Errorf("Deduplication failed (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDedupAndSortSequence(t *testing.T) {
	tests := []struct {
		name        string
		expressions []string
		inputValues []string
		want        string
	}{
		{
			name:        "sort then dedup",
			expressions: []string{".fruits"},
			inputValues: []string{"zebra", "apple", "banana", "apple", "zebra", "orange"},
			want:        "- apple\n- banana\n- orange\n- zebra\n",
		},
		{
			name:        "dedup only, no sort",
			expressions: []string{".vegetables"},
			inputValues: []string{"zebra", "apple", "banana", "apple", "zebra", "orange"},
			want:        "- zebra\n- apple\n- banana\n- apple\n- zebra\n- orange\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var contentNodes []*yaml.Node
			for _, value := range tc.inputValues {
				contentNodes = append(contentNodes, &yaml.Node{Kind: yaml.ScalarNode, Value: value})
			}

			node := &yaml.Node{
				Kind:    yaml.SequenceNode,
				Content: contentNodes,
			}

			var out bytes.Buffer
			encoder := NewEncoder(&out)
			encoder = encoder.SetIndent(2)

			if tc.name == "sort then dedup" {
				encoder, _ = encoder.SetSortExpressions(".fruits")
			}
			encoder, err := encoder.SetDedupExpressions(tc.expressions[0])
			if err != nil {
				t.Fatalf("Failed to SetDedupExpressions: %+v", err)
			}

			nodePath, err := path.Parse(".fruits")
			if err != nil {
				t.Fatalf("failed to parse path: %+v", err)
			}

			got, err := encoder.marshalSequence(node, nodePath)
			if err != nil {
				t.Errorf("Failed to marshal sequence: %+v", err)
			}

			if diff := cmp.Diff(tc.want, string(got)); diff != "" {
				t.Errorf("Sort and dedup combination failed (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSortingWithComments(t *testing.T) {
	t.Run("sorting works with YAML files containing comments", func(t *testing.T) {
		// Test YAML content with comment at the beginning and unsorted dependencies
		yamlContent := `#nolint:some-comment
package:
  dependencies:
    runtime:
      - z-item
      - a-item
      - m-item`

		expectedYaml := `#nolint:some-comment
package:
  dependencies:
    runtime:
      - a-item
      - m-item
      - z-item
`

		// Parse the YAML
		root := &yaml.Node{}
		err := yaml.NewDecoder(strings.NewReader(yamlContent)).Decode(root)
		require.NoError(t, err)

		// Create encoder with sort expression
		var buf bytes.Buffer
		encoder := NewEncoder(&buf)
		encoder, err = encoder.SetSortExpressions(".package.dependencies.runtime")
		require.NoError(t, err)

		// Encode and check result
		err = encoder.Encode(root)
		require.NoError(t, err)

		checkDiff(t, expectedYaml, buf.String())
	})

	t.Run("sorting works with complex YAML structure and comments", func(t *testing.T) {
		// Test with the more realistic structure similar to the original issue
		yamlContent := `#nolint:valid-pipeline-fetch-digest
package:
  name: test-package
  dependencies:
    runtime:
      - curl
      - ca-certificates
      - brotli
      - apache
environment:
  contents:
    packages:
      - wget
      - bash
      - autoconf`

		expectedYaml := `#nolint:valid-pipeline-fetch-digest
package:
  name: test-package
  dependencies:
    runtime:
      - apache
      - brotli
      - ca-certificates
      - curl
environment:
  contents:
    packages:
      - autoconf
      - bash
      - wget
`

		// Parse the YAML
		root := &yaml.Node{}
		err := yaml.NewDecoder(strings.NewReader(yamlContent)).Decode(root)
		require.NoError(t, err)

		// Create encoder with multiple sort expressions
		var buf bytes.Buffer
		encoder := NewEncoder(&buf)
		encoder, err = encoder.SetSortExpressions(".package.dependencies.runtime", ".environment.contents.packages")
		require.NoError(t, err)

		// Encode and check result
		err = encoder.Encode(root)
		require.NoError(t, err)

		checkDiff(t, expectedYaml, buf.String())
	})

	t.Run("path matching works correctly with key nodes containing comments", func(t *testing.T) {
		// Test that path matching works when key nodes themselves have comments
		// This tests the core bug fix where comments in keys broke path construction

		// Create a key node with a comment
		keyNode := &yaml.Node{
			Kind:        yaml.ScalarNode,
			Value:       "runtime",
			HeadComment: "# This is a comment on the key",
		}

		// Create the sequence to be sorted
		sequenceNode := &yaml.Node{
			Kind: yaml.SequenceNode,
			Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Value: "zebra"},
				{Kind: yaml.ScalarNode, Value: "alpha"},
				{Kind: yaml.ScalarNode, Value: "beta"},
			},
		}

		// Create a mapping with the commented key
		mappingNode := &yaml.Node{
			Kind: yaml.MappingNode,
			Content: []*yaml.Node{
				keyNode,
				sequenceNode,
			},
		}

		// Wrap in document
		root := &yaml.Node{
			Kind:    yaml.DocumentNode,
			Content: []*yaml.Node{mappingNode},
		}

		// Create encoder with sort expression
		var buf bytes.Buffer
		encoder := NewEncoder(&buf)
		encoder, err := encoder.SetSortExpressions(".runtime")
		require.NoError(t, err)

		// Encode
		err = encoder.Encode(root)
		require.NoError(t, err)

		result := buf.String()

		// Check that sorting happened - alpha should come before zebra
		require.Contains(t, result, "alpha")
		require.Contains(t, result, "beta")
		require.Contains(t, result, "zebra")

		// Ensure the items are in sorted order
		alphaPos := strings.Index(result, "alpha")
		betaPos := strings.Index(result, "beta")
		zebraPos := strings.Index(result, "zebra")

		require.True(t, alphaPos < betaPos, "alpha should come before beta")
		require.True(t, betaPos < zebraPos, "beta should come before zebra")

		// Ensure the comment is preserved
		require.Contains(t, result, "# This is a comment on the key")
	})
}

func TestDedupWithNonScalarNodes(t *testing.T) {
	node := &yaml.Node{
		Kind: yaml.SequenceNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "apple"},
			{Kind: yaml.MappingNode, Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Value: "type"},
				{Kind: yaml.ScalarNode, Value: "fruit"},
			}},
			{Kind: yaml.ScalarNode, Value: "apple"},
			{Kind: yaml.MappingNode, Content: []*yaml.Node{
				{Kind: yaml.ScalarNode, Value: "type"},
				{Kind: yaml.ScalarNode, Value: "fruit"},
			}},
		},
	}

	var out bytes.Buffer
	encoder := NewEncoder(&out)
	encoder = encoder.SetIndent(2)
	encoder, err := encoder.SetDedupExpressions(".items")
	if err != nil {
		t.Fatalf("Failed to SetDedupExpressions: %+v", err)
	}

	nodePath, err := path.Parse(".items")
	if err != nil {
		t.Fatalf("failed to parse path: %+v", err)
	}

	got, err := encoder.marshalSequence(node, nodePath)
	if err != nil {
		t.Errorf("Failed to marshal sequence: %+v", err)
	}

	expected := "- apple\n- type: fruit\n- type: fruit\n"
	if diff := cmp.Diff(expected, string(got)); diff != "" {
		t.Errorf("Non-scalar preservation failed (-want +got):\n%s", diff)
	}
}

func TestMarshalMappingWithMissingValue(t *testing.T) {
	// Test that the encoder doesn't crash when a mapping has a key without a corresponding value
	// This tests the bounds checking fix for accessing node.Content[i+1]

	// Create a malformed mapping node with odd number of content items (key without value)
	keyNode := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Tag:   "!!str",
		Value: "key",
	}

	mappingNode := &yaml.Node{
		Kind:    yaml.MappingNode,
		Content: []*yaml.Node{keyNode}, // Only key, no value - this would cause index out of bounds
	}

	var out bytes.Buffer
	encoder := NewEncoder(&out)
	encoder = encoder.SetIndent(2)

	nodePath, err := path.Parse(".")
	if err != nil {
		t.Fatalf("failed to parse path: %+v", err)
	}

	// This should not panic and should handle the missing value gracefully
	result, err := encoder.marshalMapping(mappingNode, nodePath)
	if err != nil {
		t.Errorf("marshalMapping failed: %+v", err)
	}

	// The result should be empty since malformed keys (without values) are skipped
	expected := ""
	if string(result) != expected {
		t.Errorf("unexpected result: got %q, want %q", string(result), expected)
	}
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
