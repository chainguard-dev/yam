package cmd

import (
	"fmt"
	"io"
	"os"

	osAdapter "github.com/chainguard-dev/yam/pkg/rwfs/os"
	"github.com/chainguard-dev/yam/pkg/util"
	"github.com/chainguard-dev/yam/pkg/yam"
	"github.com/chainguard-dev/yam/pkg/yam/formatted"
	"github.com/spf13/cobra"
)

const (
	flagIndent       = "indent"
	flagGap          = "gap"
	flagSort         = "sort"
	flagFinalNewline = "final-newline"
	flagTrimLines    = "trim-lines"
	flagLint         = "lint"
	flagConfig       = "config"
	flagQuote        = "quote"
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
	cmd.Flags().StringSlice(flagSort, nil, "YAML path expression to a mapping or sequence node whose children should be sorted")
	cmd.Flags().Bool(flagFinalNewline, true, "ensure file ends with a final newline character")
	cmd.Flags().Bool(flagTrimLines, true, "trim any trailing spaces from each line")
	cmd.Flags().Bool(flagLint, false, "don't modify files, but exit 1 if files aren't formatted")
	cmd.Flags().StringP(flagConfig, "c", "", "path to a yam configuration YAML file")
	cmd.Flags().StringSlice(flagQuote, nil, "YAML path expression to a node that should be quoted")

	cmd.RunE = runRoot

	return cmd
}

func runRoot(cmd *cobra.Command, args []string) error {
	encoderConfig, err := getConfig(cmd)
	if err != nil {
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

func getConfig(cmd *cobra.Command) (*formatted.EncodeOptions, error) {
	var r io.Reader
	if v, _ := cmd.Flags().GetString(flagConfig); v != "" {
		f, err := os.Open(v)
		if err != nil {
			return nil, fmt.Errorf("opening configuration file: %w", err)
		}
		r = f
	} else {
		f, err := os.Open(util.ConfigFileName)
		if err != nil {
			// This is a default best-effort attempt, no need to bubble up the error.
			return nil, nil
		}
		r = f
	}

	if r == nil {
		return nil, nil
	}

	cfg, err := formatted.ReadConfigFrom(r)
	if err != nil {
		return nil, fmt.Errorf("reading configuration: %w", err)
	}
	return cfg, nil
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

	var sortExpressions []string
	if flag := flags.Lookup(flagSort); flag.Changed {
		sortExpressions, _ = flags.GetStringSlice(flagSort)
	} else if cfg != nil {
		sortExpressions = cfg.SortExpressions
	}

	var quoteExpressions []string
	if flag := flags.Lookup(flagQuote); flag.Changed {
		quoteExpressions, _ = flags.GetStringSlice(flagQuote)
	} else if cfg != nil {
		quoteExpressions = cfg.QuoteExpressions
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
			Indent:           indent,
			GapExpressions:   gapExpressions,
			SortExpressions:  sortExpressions,
			QuoteExpressions: quoteExpressions,
		},
		FinalNewline:           finalNewline,
		TrimTrailingWhitespace: trimLines,
	}
}
