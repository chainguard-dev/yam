package cmd

import (
	"errors"
	"os"

	osAdapter "github.com/chainguard-dev/yam/pkg/rwfs/os"
	"github.com/chainguard-dev/yam/pkg/yam"
	"github.com/chainguard-dev/yam/pkg/yam/formatted"
	"github.com/spf13/cobra"
)

const (
	flagIndent       = "indent"
	flagGap          = "gap"
	flagFinalNewline = "final-newline"
	flagTrimLines    = "trim-lines"
	flagLint         = "lint"
)

func Root() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "yam <file>...",
		Short:         "format YAML files",
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	cmd.Flags().Int(flagIndent, 2, "number of spaces used to indent a line")
	cmd.Flags().StringSlice(flagGap, nil, "YAML path expression to a mapping or sequence node whose children should be separated by empty lines")
	cmd.Flags().Bool(flagFinalNewline, true, "ensure file ends with a final newline character")
	cmd.Flags().Bool(flagTrimLines, true, "trim any trailing spaces from each line")
	cmd.Flags().Bool(flagLint, false, "don't modify files, but exit 1 if files aren't formatted")

	cmd.RunE = runRoot

	return cmd
}

func runRoot(cmd *cobra.Command, args []string) error {
	var err error

	encoderConfig, err := formatted.ReadConfig()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	formatOptions := computeFormatOptions(encoderConfig, cmd)
	doLint, _ := cmd.Flags().GetBool(flagLint)

	if doLint {
		fsys := os.DirFS(".")

		err = yam.Lint(fsys, args, yam.ExecDiff, formatOptions)
		if err != nil {
			return err
		}

		return nil
	}

	fsys := osAdapter.DirFS(".")
	err = yam.Format(fsys, args, formatOptions)
	if err != nil {
		return err
	}

	return nil
}

// computeFormatOptions produces a new yam.FormatOptions using an optional
// provided config (unmarshalled from a file) and flags from a Cobra command.
// CLI flag values take priority over config file values, which take priority
// over default values.
func computeFormatOptions(cfg *formatted.EncodeOptions, cmd *cobra.Command) yam.FormatOptions {
	flags := cmd.Flags()

	var indent = 2
	if flag := flags.Lookup(flagIndent); flag.Changed {
		indent, _ = flags.GetInt(flagIndent)
	} else if cfg != nil {
		indent = cfg.Indent
	}

	var gapExpressions []string
	if flag := flags.Lookup(flagGap); flag.Changed {
		gapExpressions, _ = flags.GetStringSlice(flagGap)
	} else if cfg != nil {
		gapExpressions = cfg.GapExpressions
	}

	var finalNewline = true
	if flag := flags.Lookup(flagFinalNewline); flag.Changed {
		finalNewline, _ = flags.GetBool(flagFinalNewline)
	}

	var trimLines = true
	if flag := flags.Lookup(flagTrimLines); flag.Changed {
		trimLines, _ = flags.GetBool(flagTrimLines)
	}

	return yam.FormatOptions{
		EncodeOptions: formatted.EncodeOptions{
			Indent:         indent,
			GapExpressions: gapExpressions,
		},
		FinalNewline:           finalNewline,
		TrimTrailingWhitespace: trimLines,
	}
}
