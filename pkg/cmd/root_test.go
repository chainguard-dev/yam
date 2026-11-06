package cmd

import (
	"strings"
	"testing"

	"github.com/chainguard-dev/yam/pkg/yam/formatted"
	"github.com/stretchr/testify/require"
)

// TestComputeFormatOptionsIndent covers the indent precedence between the
// --indent flag, a config file and the default. A config file that omits
// indent decodes to 0 and used to replace the default (#269).
func TestComputeFormatOptionsIndent(t *testing.T) {
	tests := []struct {
		name   string
		config string // contents of a config file; empty means no config file
		args   []string
		want   int
	}{{
		name: "no config file and no flag uses the default indent",
		want: 2,
	}, {
		name:   "config file without indent uses the default indent",
		config: "gap:\n  - .\n",
		want:   2,
	}, {
		name:   "config file with only quote expressions uses the default indent",
		config: "quote:\n  - .package.version\n",
		want:   2,
	}, {
		name:   "config file indent is used",
		config: "indent: 4\n",
		want:   4,
	}, {
		name:   "flag overrides config file indent",
		config: "indent: 4\n",
		args:   []string{"--indent", "3"},
		want:   3,
	}, {
		name:   "flag is used when config file has no indent",
		config: "gap:\n  - .\n",
		args:   []string{"--indent", "3"},
		want:   3,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfg *formatted.EncodeOptions
			if tt.config != "" {
				var err error
				cfg, err = formatted.ReadConfigFrom(strings.NewReader(tt.config))
				require.NoError(t, err)
			}

			cmd := Root()
			require.NoError(t, cmd.ParseFlags(tt.args))

			got := computeFormatOptions(cfg, cmd).EncodeOptions.Indent
			if got != tt.want {
				t.Errorf("config %q, args %v: indent = %d, want %d", tt.config, tt.args, got, tt.want)
			}
		})
	}
}
