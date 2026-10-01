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

// TestKeyComments covers comments that yaml.v3 attaches to a mapping key node
// as a line or foot comment. Rendering them together with the key used to put
// the colon inside the comment, producing invalid YAML (e.g. "- runs\n  # c: |")
// or YAML that silently parsed to different data.
func TestKeyComments(t *testing.T) {
	tests := []struct {
		name  string
		gaps  []string
		input string
		want  string
	}{{
		// Seen in melange configs formatted after a renovate bump.
		name: "comment block before a sequence item after a nested mapping",
		gaps: []string{".", ".pipeline"},
		input: `pipeline:
  - uses: git-checkout
    with:
      expected-commit: abc
  # head comment

  - runs: |
      echo hi
`,
		want: `pipeline:
  - uses: git-checkout
    with:
      expected-commit: abc

  - runs: |
      echo hi
    # head comment
`,
	}, {
		name: "comment after the last item's scalar value",
		gaps: []string{".pipeline"},
		input: `pipeline:
  - runs: |
      hi
  # c

  - uses: y
`,
		want: `pipeline:
  - runs: |
      hi

  - uses: y
    # c
`,
	}, {
		name: "comment after a nested mapping inside a mapping",
		input: `a:
  b:
    x: 1
  # c

  d: 2
`,
		want: `a:
  b:
    x: 1
  # c
  d: 2
`,
	}, {
		name: "comment after a nested mapping at the top level",
		gaps: []string{"."},
		input: `a:
  x: 1
# c

d: 2
`,
		want: `a:
  x: 1
# c

d: 2
`,
	}, {
		name: "comment between keys of a sequence item",
		input: `pipeline:
  - with:
      x: 1
    # c

    uses: y
`,
		want: `pipeline:
  - with:
      x: 1
    # c
    uses: y
`,
	}, {
		name: "line comment on a key with a block value",
		input: `a: # c
  x: 1
`,
		want: `a: # c
  x: 1
`,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := &yaml.Node{}
			require.NoError(t, yaml.Unmarshal([]byte(tt.input), root))

			var buf bytes.Buffer
			enc, err := NewEncoder(&buf).SetGapExpressions(tt.gaps...)
			require.NoError(t, err)
			require.NoError(t, enc.Encode(root))

			checkDiff(t, tt.want, buf.String())

			var want, got any
			require.NoError(t, yaml.Unmarshal([]byte(tt.input), &want))
			require.NoError(t, yaml.Unmarshal(buf.Bytes(), &got), "formatted output must parse")
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("formatted output decodes to different data (-want +got):\n%s", diff)
			}
		})
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

func TestQuoteScalars(t *testing.T) {
	tests := []struct {
		name  string
		quote []string
		input string
		want  string
		wantV any // the decoded value of "v" in the formatted output
	}{{
		name:  "float is quoted as a string",
		quote: []string{".v"},
		input: "v: 5.12\n",
		want:  "v: \"5.12\"\n",
		wantV: "5.12",
	}, {
		name:  "int is quoted as a string",
		quote: []string{".v"},
		input: "v: 42\n",
		want:  "v: \"42\"\n",
		wantV: "42",
	}, {
		name:  "hex int keeps its original text",
		quote: []string{".v"},
		input: "v: 0x1F\n",
		want:  "v: \"0x1F\"\n",
		wantV: "0x1F",
	}, {
		name:  "octal int keeps its original text",
		quote: []string{".v"},
		input: "v: 0o17\n",
		want:  "v: \"0o17\"\n",
		wantV: "0o17",
	}, {
		name:  "infinity is quoted",
		quote: []string{".v"},
		input: "v: .inf\n",
		want:  "v: \".inf\"\n",
		wantV: ".inf",
	}, {
		name:  "NaN is quoted",
		quote: []string{".v"},
		input: "v: .NaN\n",
		want:  "v: \".NaN\"\n",
		wantV: ".NaN",
	}, {
		name:  "exponent float keeps its original text",
		quote: []string{".v"},
		input: "v: 1e3\n",
		want:  "v: \"1e3\"\n",
		wantV: "1e3",
	}, {
		name:  "underscore-separated int keeps its original text",
		quote: []string{".v"},
		input: "v: 1_000\n",
		want:  "v: \"1_000\"\n",
		wantV: "1_000",
	}, {
		name:  "bool is quoted as a string",
		quote: []string{".v"},
		input: "v: true\n",
		want:  "v: \"true\"\n",
		wantV: "true",
	}, {
		name:  "plain string is quoted",
		quote: []string{".v"},
		input: "v: hello\n",
		want:  "v: \"hello\"\n",
		wantV: "hello",
	}, {
		// yaml.v3 already reads yes as a string, but YAML 1.1 consumers read
		// it as a bool, so it still needs quoting.
		name:  "YAML 1.1 bool word is quoted",
		quote: []string{".v"},
		input: "v: yes\n",
		want:  "v: \"yes\"\n",
		wantV: "yes",
	}, {
		name:  "timestamp is quoted as a string",
		quote: []string{".v"},
		input: "v: 2024-01-01\n",
		want:  "v: \"2024-01-01\"\n",
		wantV: "2024-01-01",
	}, {
		name:  "single-quoted string becomes double-quoted",
		quote: []string{".v"},
		input: "v: '5.12'\n",
		want:  "v: \"5.12\"\n",
		wantV: "5.12",
	}, {
		name:  "already double-quoted value is unchanged",
		quote: []string{".v"},
		input: "v: \"5.12\"\n",
		want:  "v: \"5.12\"\n",
		wantV: "5.12",
	}, {
		name:  "line comment on a quoted value is kept",
		quote: []string{".v"},
		input: "v: 5.12 # keep\n",
		want:  "v: \"5.12\" # keep\n",
		wantV: "5.12",
	}, {
		name:  "null on a quote path stays null",
		quote: []string{".v"},
		input: "v: ~\nw: 1\n",
		want:  "v:\nw: 1\n",
		wantV: nil,
	}, {
		name:  "anchored scalar keeps its anchor",
		quote: []string{".v"},
		input: "v: &a 1.5\n",
		want:  "v: &a \"1.5\"\n",
		wantV: "1.5",
	}, {
		// Pinned on purpose: quoting a block scalar flattens it to one escaped
		// line. Change this row deliberately if that behavior changes.
		name:  "block literal becomes a double-quoted string",
		quote: []string{".v"},
		input: "v: |\n  5.12\n",
		want:  "v: \"5.12\\n\"\n",
		wantV: "5.12\n",
	}, {
		name:  "any-index path quotes every sequence item",
		quote: []string{".v[]"},
		input: "v:\n  - 1.0\n  - 2\n",
		want:  "v:\n  - \"1.0\"\n  - \"2\"\n",
		wantV: []any{"1.0", "2"},
	}, {
		name:  "specific-index path quotes only that sequence item",
		quote: []string{".v[1]"},
		input: "v:\n  - 1.0\n  - 2\n",
		want:  "v:\n  - 1.0\n  - \"2\"\n",
		wantV: []any{1.0, "2"},
	}, {
		// Quoting must not override a type the author spelled out.
		name:  "explicit float tag is kept",
		quote: []string{".v"},
		input: "v: !!float 5.12\n",
		want:  "v: !!float \"5.12\"\n",
		wantV: 5.12,
	}, {
		name:  "explicit int tag is kept",
		quote: []string{".v"},
		input: "v: !!int 42\n",
		want:  "v: !!int \"42\"\n",
		wantV: 42,
	}, {
		name:  "explicit bool tag is kept",
		quote: []string{".v"},
		input: "v: !!bool true\n",
		want:  "v: !!bool \"true\"\n",
		wantV: true,
	}, {
		name:  "explicit str tag is kept",
		quote: []string{".v"},
		input: "v: !!str 5.12\n",
		want:  "v: !!str \"5.12\"\n",
		wantV: "5.12",
	}, {
		name:  "every quote expression is applied",
		quote: []string{".a", ".v"},
		input: "a: 1\nv: 2\n",
		want:  "a: \"1\"\nv: \"2\"\n",
		wantV: "2",
	}, {
		name:  "non-matching path leaves the value unquoted",
		quote: []string{".other"},
		input: "v: 5.12\n",
		want:  "v: 5.12\n",
		wantV: 5.12,
	}, {
		// The repro in #100 used a same-depth path with a different parent.
		name:  "same-depth path under a different parent leaves the value unquoted",
		quote: []string{".other.x"},
		input: "v:\n  x: 5.12\n",
		want:  "v:\n  x: 5.12\n",
		wantV: map[string]any{"x": 5.12},
	}, {
		name:  "deeper path leaves a top-level value unquoted",
		quote: []string{".package.v"},
		input: "v: 5.12\n",
		want:  "v: 5.12\n",
		wantV: 5.12,
	}, {
		name:  "path to a sequence does not quote its items",
		quote: []string{".v"},
		input: "v:\n  - 1.0\n",
		want:  "v:\n  - 1.0\n",
		wantV: []any{1.0},
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			format := func(input string) string {
				t.Helper()
				root := &yaml.Node{}
				require.NoError(t, yaml.Unmarshal([]byte(input), root))

				var buf bytes.Buffer
				enc, err := NewEncoder(&buf).SetQuoteExpressions(tt.quote...)
				require.NoError(t, err)
				require.NoError(t, enc.Encode(root))
				return buf.String()
			}

			out := format(tt.input)
			checkDiff(t, tt.want, out)

			if again := format(out); again != out {
				t.Errorf("quote %v: formatting again changed the output:\nfirst:  %q\nsecond: %q", tt.quote, out, again)
			}

			var got map[string]any
			require.NoError(t, yaml.Unmarshal([]byte(out), &got), "formatted output must parse:\n%s", out)
			v, ok := got["v"]
			if !ok {
				t.Fatalf("quote %v on %q: key v missing from output %q", tt.quote, tt.input, out)
			}
			if diff := cmp.Diff(tt.wantV, v); diff != "" {
				t.Errorf("quote %v on %q: decoded value changed (-want +got):\n%s", tt.quote, tt.input, diff)
			}
		})
	}
}
