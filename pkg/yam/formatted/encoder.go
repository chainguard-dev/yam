package formatted

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/chainguard-dev/yam/pkg/util"
	"github.com/chainguard-dev/yam/pkg/yam/formatted/path"
	"gopkg.in/yaml.v3"
)

var (
	newline   = []byte("\n")
	colon     = []byte(":")
	space     = []byte(" ")
	dashSpace = []byte("- ")
)

const defaultIndentSize = 2

// EncodeOptions describes the set of configuration options used to adjust the
// behavior of yam's YAML encoder.
type EncodeOptions struct {
	// Indent specifies how many spaces to use per-indentation
	Indent int `yaml:"indent"`

	// GapExpressions specifies a list of yq-style paths for which the path's YAML
	// element's children elements should be separated by an empty line
	GapExpressions []string `yaml:"gap"`

	// SortExpressions specifies a list of yq-style paths for which the path's YAML
	// element's children elements should be sorted
	SortExpressions []string `yaml:"sort"`
}

// Encoder is an implementation of a YAML encoder that applies a configurable
// formatting to the YAML data as it's written out to the encoder's io.Writer.
type Encoder struct {
	w          io.Writer
	indentSize int
	yamlEnc    *yaml.Encoder
	gapPaths   []path.Path
	sortPaths  []path.Path
}

// NewEncoder returns a new encoder that can write formatted YAML to the given
// io.Writer.
func NewEncoder(w io.Writer) Encoder {
	yamlEnc := yaml.NewEncoder(w)
	yamlEnc.SetIndent(defaultIndentSize)

	enc := Encoder{
		w:          w,
		yamlEnc:    yamlEnc,
		indentSize: defaultIndentSize,
	}

	return enc
}

// AutomaticConfig configures the encoder using a `.yam.yaml` config file in the
// current working directory, if one exists. This method is meant to work on a
// "best effort" basis, and all errors are silently ignored.
func (enc Encoder) AutomaticConfig() Encoder {
	options, err := ReadConfig()
	if err != nil {
		// Didn't find a config to apply, but that's okay.
		return enc
	}

	enc = enc.SetIndent(options.Indent)
	enc, _ = enc.SetGapExpressions(options.GapExpressions...)

	return enc
}

// ReadConfig tries to load a yam encoder config from a `.yam.yaml` file in the
// current working directory. It returns an error if it wasn't able to open or
// unmarshal the file.
func ReadConfig() (*EncodeOptions, error) {
	f, err := os.Open(util.ConfigFileName)
	if err != nil {
		return nil, fmt.Errorf("unable to open yam config: %w", err)
	}

	config, err := ReadConfigFrom(f)
	if err != nil {
		return nil, err
	}

	return config, nil
}

// ReadConfigFrom loads a yam encoder config from the given io.Reader. It
// returns an error if it wasn't able to unmarshal the data.
func ReadConfigFrom(r io.Reader) (*EncodeOptions, error) {
	options := EncodeOptions{}

	err := yaml.NewDecoder(r).Decode(&options)
	if err != nil {
		return nil, fmt.Errorf("unable to read yam config: %w", err)
	}

	return &options, nil
}

// SetIndent configures the encoder to use the provided number of spaces for
// each indentation.
func (enc Encoder) SetIndent(spaces int) Encoder {
	enc.indentSize = spaces
	enc.yamlEnc.SetIndent(spaces)
	return enc
}

// SetGapExpressions takes 0 or more YAML path expressions (e.g. "." or
// ".something.foo") and configures the encoder to insert empty lines ("gaps")
// in between the children elements of the YAML nodes referenced by the path
// expressions.
func (enc Encoder) SetGapExpressions(expressions ...string) (Encoder, error) {
	for _, expr := range expressions {
		p, err := path.Parse(expr)
		if err != nil {
			return Encoder{}, fmt.Errorf("unable to parse expression %q: %w", expr, err)
		}

		enc.gapPaths = append(enc.gapPaths, p)
	}

	return enc, nil
}

// SetSortExpressions takes 0 or more YAML path expressions (e.g. "." or
// ".something.foo") and configures the encoder to sort the arrays.
func (enc Encoder) SetSortExpressions(expressions ...string) (Encoder, error) {
	for _, expr := range expressions {
		p, err := path.Parse(expr)
		if err != nil {
			return Encoder{}, fmt.Errorf("unable to parse expression %q: %w", expr, err)
		}

		enc.sortPaths = append(enc.sortPaths, p)
	}

	return enc, nil
}

// UseOptions configures the encoder to use the configuration from the given
// EncodeOptions.
func (enc Encoder) UseOptions(options EncodeOptions) (Encoder, error) {
	enc = enc.SetIndent(options.Indent)
	enc, err := enc.SetGapExpressions(options.GapExpressions...)
	if err != nil {
		return Encoder{}, err
	}
	enc, err = enc.SetSortExpressions(options.SortExpressions...)
	if err != nil {
		return Encoder{}, err
	}

	return enc, nil
}

// Encode writes out the formatted YAML from the given yaml.Node to the
// encoder's io.Writer.
func (enc Encoder) Encode(node *yaml.Node) error {
	b, err := enc.marshalRoot(node)
	if err != nil {
		return err
	}

	_, err = enc.w.Write(b)
	if err != nil {
		return err
	}

	return nil
}

func (enc Encoder) marshalRoot(node *yaml.Node) ([]byte, error) {
	return enc.marshal(node, path.Root())
}

func (enc Encoder) marshal(node *yaml.Node, nodePath path.Path) ([]byte, error) {
	switch node.Kind {
	case yaml.DocumentNode:
		var bytes []byte
		for _, inner := range node.Content {
			innerBytes, err := enc.marshal(inner, nodePath)
			if err != nil {
				return nil, err
			}
			bytes = append(bytes, innerBytes...)
		}
		return bytes, nil

	case yaml.MappingNode:
		// Handle empty mappings explicitly
		//if len(node.Content) == 0 {
		//	return []byte("{}"), nil
		//}

		// Marshal non-empty mappings
		return enc.marshalMapping(node, nodePath)

	case yaml.SequenceNode:
		return enc.marshalSequence(node, nodePath)

	case yaml.ScalarNode:
		if node.Tag == "!!null" {
			return nil, nil
		}
		return yaml.Marshal(node)

	default:
		return yaml.Marshal(node)
	}
}

func (enc Encoder) marshalMapping(node *yaml.Node, nodePath path.Path) ([]byte, error) {
	// Note: A mapping node's content items are laid out as key-value pairs!

	var result []byte
	var latestKey string
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]

		rawKeyBytes, err := enc.marshal(keyNode, nodePath)
		if err != nil {
			return nil, err
		}

		key := bytes.TrimSuffix(rawKeyBytes, newline)
		latestKey = string(key)

		keyBytes := bytes.Join([][]byte{
			key,
			colon,
		}, nil)

		// Check for empty mapping node and handle it
		if valueNode.Kind == yaml.MappingNode && len(valueNode.Content) == 0 {
			keyBytes = append(keyBytes, space...)
			keyBytes = append(keyBytes, []byte("{}")...)
		} else if valueNode.Kind == yaml.ScalarNode && valueNode.Tag != "!!null" {
			// Render scalar values in the same line
			keyBytes = append(keyBytes, space...)
		} else {
			keyBytes = append(keyBytes, newline...)
		}

		result = append(result, keyBytes...)

		valueBytes, err := enc.marshal(valueNode, nodePath.AppendMapPart(latestKey))
		if err != nil {
			return nil, err
		}

		isFinalMapValue := (i + 1) == len(node.Content)-1

		// Avoid adding a newline after the final map value
		if enc.matchesAnyGapPath(nodePath) && !isFinalMapValue {
			valueBytes = append(valueBytes, newline...)
		}

		if valueNode.Kind == yaml.MappingNode || valueNode.Kind == yaml.SequenceNode {
			valueBytes = enc.applyIndent(valueBytes)
		} else {
			valueBytes = enc.handleMultilineStringIndentation(valueBytes)
		}

		result = append(result, valueBytes...)
	}

	return result, nil
}

func (enc Encoder) marshalSequence(node *yaml.Node, nodePath path.Path) ([]byte, error) {
	var lines [][]byte

	// Sort the sequence if configured to do so before marshalling.
	if node.Kind == yaml.SequenceNode && enc.matchesAnySortPath(nodePath) {
		sort.Slice(node.Content, func(i int, j int) bool {
			return node.Content[i].Value < node.Content[j].Value
		})
	}

	for i, item := range node.Content {
		// For scalar items, pull out the head comment, so we can control its encoding
		// here, rather than delegate it to the underlying encoder.
		var extractedHeadComment string
		if item.HeadComment != "" {
			extractedHeadComment = item.HeadComment + "\n"
			item.HeadComment = ""
		}

		itemBytes, err := enc.marshal(item, nodePath.AppendSeqPart(i))
		if err != nil {
			return nil, err
		}

		if item.Kind == yaml.ScalarNode {
			// Print head comment first. Then continue.
			itemBytes = bytes.Join([][]byte{
				[]byte(extractedHeadComment),
				dashSpace,
				itemBytes,
			}, nil)
		} else {
			itemBytes = enc.applyIndentExceptFirstLine(itemBytes)

			// Precede with a dash.
			itemBytes = bytes.Join([][]byte{
				[]byte(extractedHeadComment),
				dashSpace,
				itemBytes,
			}, nil)
		}

		lines = append(lines, itemBytes)
	}

	var sep []byte
	if enc.matchesAnyGapPath(nodePath) {
		sep = newline
	}

	return bytes.Join(lines, sep), nil
}

func (enc Encoder) applyIndent(content []byte) []byte {
	var processedLines []string

	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()

		// We don't indent empty lines.
		if line != "" {
			line = enc.indentString() + line
		}
		processedLines = append(processedLines, line)
	}

	result := []byte(strings.Join(processedLines, "\n") + "\n")

	return result
}

func (enc Encoder) applyIndentExceptFirstLine(content []byte) []byte {
	var processedLines []string

	scanner := bufio.NewScanner(bytes.NewReader(content))
	isFirstLine := true
	for scanner.Scan() {
		line := scanner.Text()

		if isFirstLine {
			processedLines = append(processedLines, line)
			isFirstLine = false
			continue
		}

		// We don't indent empty lines.
		if line != "" {
			line = enc.indentString() + line
		}
		processedLines = append(processedLines, line)
	}

	return []byte(strings.Join(processedLines, "\n") + "\n")
}

func (enc Encoder) matchesAnyGapPath(testSubject path.Path) bool {
	for _, gp := range enc.gapPaths {
		if gp.Matches(testSubject) {
			return true
		}
	}

	return false
}

func (enc Encoder) matchesAnySortPath(testSubject path.Path) bool {
	for _, sp := range enc.sortPaths {
		if sp.Matches(testSubject) {
			return true
		}
	}
	return false
}

func (enc Encoder) handleMultilineStringIndentation(content []byte) []byte {
	// For some reason, yaml.Marshal seemed to be indenting non-first lines twice.

	lines := bytes.Split(content, newline)
	if len(lines) == 1 {
		return content
	}

	for i := 1; i < len(lines); i++ { // i.e. starting with second line
		lines[i] = bytes.TrimPrefix(lines[i], []byte(enc.indentString()))
	}

	return bytes.Join(lines, newline)
}

func (enc Encoder) indentString() string {
	return strings.Repeat(" ", enc.indentSize)
}
