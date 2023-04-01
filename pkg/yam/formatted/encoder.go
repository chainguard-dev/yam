package formatted

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"

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

type Encoder struct {
	w          io.Writer
	indentSize int
	yamlEnc    *yaml.Encoder
	gapPaths   []path.Path
}

func NewEncoder(w io.Writer) *Encoder {
	yamlEnc := yaml.NewEncoder(w)
	yamlEnc.SetIndent(defaultIndentSize)

	enc := &Encoder{
		w:          w,
		yamlEnc:    yamlEnc,
		indentSize: defaultIndentSize,
	}

	return enc
}

func (enc *Encoder) SetIndent(spaces int) {
	enc.indentSize = spaces
	enc.yamlEnc.SetIndent(spaces)
}

func (enc *Encoder) SetGapExpressions(expressions ...string) error {
	for _, expr := range expressions {
		p, err := path.Parse(expr)
		if err != nil {
			return fmt.Errorf("unable to parse expression %q: %w", expr, err)
		}

		enc.gapPaths = append(enc.gapPaths, p)
	}

	return nil
}

func (enc *Encoder) Encode(node *yaml.Node) error {
	bytes, err := enc.Marshal(node)
	if err != nil {
		return err
	}

	_, err = enc.w.Write(bytes)
	if err != nil {
		return err
	}

	return nil
}

func (enc *Encoder) Marshal(node *yaml.Node) ([]byte, error) {
	return enc.marshal(node, path.Root())
}

func (enc *Encoder) marshal(node *yaml.Node, nodePath path.Path) ([]byte, error) {
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

func (enc *Encoder) marshalMapping(node *yaml.Node, nodePath path.Path) ([]byte, error) {
	// Note: A mapping node's content items are laid out as key-value pairs!

	var result []byte
	var latestKey string
	for i, item := range node.Content {
		if isMapKeyIndex(i) {
			rawKeyBytes, err := enc.marshal(item, nodePath)
			if err != nil {
				return nil, err
			}

			// assume the key can be a string (this isn't always true in YAML, but we'll see how far this gets us)
			key := bytes.TrimSuffix(rawKeyBytes, newline)
			latestKey = string(key)

			keyBytes := bytes.Join([][]byte{
				key,
				colon,
			}, nil)

			if nextItem := node.Content[i+1]; nextItem.Kind == yaml.ScalarNode && nextItem.Tag != "!!null" { // TODO: check that there is a value node for this key node
				// render in same line
				keyBytes = append(keyBytes, space...)
			} else {
				keyBytes = append(keyBytes, newline...)
			}

			result = append(result, keyBytes...)
			continue
		}

		nodePathForValue := nodePath.AppendMapPart(latestKey)

		valueBytes, err := enc.marshal(item, nodePathForValue)
		if err != nil {
			return nil, err
		}

		isFinalMapValue := i == len(node.Content)-1

		// This was the key's value node, so add a gap if configured to do so.
		// We shouldn't add a newline after the final map value, though.
		if enc.matchesAnyGapPath(nodePath) && !isFinalMapValue {
			valueBytes = append(valueBytes, newline...)
		}

		if item.Kind == yaml.MappingNode || item.Kind == yaml.SequenceNode {
			valueBytes = enc.applyIndent(valueBytes)
		} else {
			valueBytes = enc.handleMultilineStringIndentation(valueBytes)
		}

		result = append(result, valueBytes...)
	}

	return result, nil
}

func isMapKeyIndex(i int) bool {
	return i%2 == 0
}

func (enc *Encoder) marshalSequence(node *yaml.Node, nodePath path.Path) ([]byte, error) {
	var lines [][]byte

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

func (enc *Encoder) applyIndent(content []byte) []byte {
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

func (enc *Encoder) applyIndentExceptFirstLine(content []byte) []byte {
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

func (enc *Encoder) matchesAnyGapPath(testSubject path.Path) bool {
	for _, gp := range enc.gapPaths {
		if gp.Matches(testSubject) {
			return true
		}
	}

	return false
}

func (enc *Encoder) handleMultilineStringIndentation(content []byte) []byte {
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

func (enc *Encoder) indentString() string {
	return strings.Repeat(" ", enc.indentSize)
}
