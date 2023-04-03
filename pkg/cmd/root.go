package cmd

import (
	"os"

	osAdapter "github.com/chainguard-dev/yam/pkg/rwfs/os"
	"github.com/chainguard-dev/yam/pkg/yam"
	"github.com/spf13/cobra"
)

func Root() *cobra.Command {
	p := &rootParams{}
	cmd := &cobra.Command{
		Use:           "yam <file>...",
		Short:         "format YAML files",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			options := yam.FormatOptions{
				Indent:                 p.indentSize,
				GapExpressions:         p.gaps,
				TrimTrailingWhitespace: true,
				FinalNewline:           true,
			}

			if p.lint {
				fsys := os.DirFS(".")

				err := yam.Lint(fsys, args, yam.ExecDiff, options)
				if err != nil {
					return err
				}

				return nil
			}

			fsys := osAdapter.DirFS(".")
			err := yam.Format(fsys, args, options)
			if err != nil {
				return err
			}

			return nil
		},
	}

	p.addToCmd(cmd)

	return cmd
}

type rootParams struct {
	lint       bool
	indentSize int
	gaps       []string
}

func (p *rootParams) addToCmd(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&p.lint, "lint", false, "don't modify files, but exit 1 if files aren't formatted")
	cmd.Flags().IntVar(&p.indentSize, "indent", 2, "number of spaces used to indent a line")
	cmd.Flags().StringSliceVar(&p.gaps, "gap", nil, "YAML path expression to a mapping or sequence node whose children should be separated by empty lines")
}
