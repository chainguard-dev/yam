package yam

import (
	"bufio"
	"bytes"
	"fmt"
	"io"

	"github.com/chainguard-dev/yam/pkg/yam/formatted"
	"gopkg.in/yaml.v3"
)

func apply(input io.Reader, options FormatOptions) (*bytes.Buffer, error) {
	b, err := io.ReadAll(input)
	if err != nil {
		return nil, err
	}

	if options.TrimTrailingWhitespace {
		b = trimTrailingWhitespace(b)
	}

	if options.FinalNewline {
		b = ensureFinalNewline(b)
	}

	root := &yaml.Node{}
	decoder := yaml.NewDecoder(bytes.NewReader(b))
	err = decoder.Decode(root)
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	enc := formatted.NewEncoder(buf)
	enc.SetIndent(options.Indent)
	err = enc.SetGapExpressions(options.GapExpressions...)
	if err != nil {
		return nil, fmt.Errorf("unable to set gap expression for encoder: %w", err)
	}

	err = enc.Encode(root)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

func trimTrailingWhitespace(in []byte) []byte {
	var out []byte

	scanner := bufio.NewScanner(bytes.NewReader(in))
	for scanner.Scan() {
		line := scanner.Bytes()
		trimmedLine := bytes.TrimRight(line, " \t")
		out = append(out, trimmedLine...)

		// Add back the newline, which is stripped during scanning.
		out = append(out, []byte("\n")...)
	}

	return out
}

func ensureFinalNewline(in []byte) []byte {
	var newline = []byte("\n")

	if len(in) == 0 {
		return newline
	}

	if in[len(in)-1] == newline[0] {
		return in
	}

	return append(in, newline...)
}
