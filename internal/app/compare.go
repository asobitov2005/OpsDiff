package app

import (
	"fmt"

	"github.com/asobitov2005/OpsDiff/internal/diff"
	"github.com/asobitov2005/OpsDiff/internal/report"
	"github.com/asobitov2005/OpsDiff/internal/store"
	"github.com/spf13/cobra"
)

func newCompareCommand() *cobra.Command {
	var format string

	command := &cobra.Command{
		Use:   "compare <before> <after>",
		Short: "Compare two OpsDiff snapshot files",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			before, err := store.ReadSnapshot(args[0])
			if err != nil {
				return err
			}

			after, err := store.ReadSnapshot(args[1])
			if err != nil {
				return err
			}

			result := diff.NewEngine().Compare(before, after, args[0], args[1])
			rendered, err := report.RenderCompare(result, format)
			if err != nil {
				return err
			}

			_, err = fmt.Fprintln(cmd.OutOrStdout(), rendered)
			return err
		},
	}

	command.Flags().StringVar(&format, "format", "table", "Output format: table, json, markdown")
	return command
}
