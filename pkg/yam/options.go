package yam

import "github.com/chainguard-dev/yam/pkg/yam/formatted"

type FormatOptions struct {
	// EncodeOptions specifies the encoder-specific format options.
	EncodeOptions formatted.EncodeOptions

	// FinalNewline specifies whether to ensure the input has a final newline before
	// further formatting is applied.
	FinalNewline bool `mapstructure:"final-newline"`

	// TrimTrailingWhitespace specifies whether to trim any trailing space
	// characters from each line before further formatting is applied.
	TrimTrailingWhitespace bool `mapstructure:"trim-lines"`
}
