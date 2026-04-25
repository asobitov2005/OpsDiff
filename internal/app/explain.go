package app

import (
	"fmt"
	"time"

	"github.com/asobitov2005/OpsDiff/internal/explain"
	"github.com/asobitov2005/OpsDiff/internal/kube"
	"github.com/asobitov2005/OpsDiff/internal/report"
	"github.com/asobitov2005/OpsDiff/internal/store"
	"github.com/spf13/cobra"
)

func newExplainCommand(options *RootOptions) *cobra.Command {
	var format string
	var from string
	var limit int
	var top int

	command := &cobra.Command{
		Use:   "explain <before> <after>",
		Short: "Rank likely causes by correlating snapshot diffs with runtime timeline signals",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			duration, err := time.ParseDuration(from)
			if err != nil {
				return fmt.Errorf("parse --from duration: %w", err)
			}
			if duration <= 0 {
				return fmt.Errorf("--from must be greater than zero")
			}

			before, err := store.ReadSnapshot(args[0])
			if err != nil {
				return err
			}
			after, err := store.ReadSnapshot(args[1])
			if err != nil {
				return err
			}

			namespace := options.Namespace
			if namespace == "default" && after.Namespace != "" && after.Namespace != "default" {
				namespace = after.Namespace
			}

			collector := kube.NewCollector(options.Kubeconfig)
			timeline, err := collector.CollectTimeline(cmd.Context(), namespace, duration, limit)
			if err != nil {
				return err
			}

			result := explain.NewEngine().Explain(before, after, timeline, args[0], args[1], top)
			rendered, err := report.RenderExplain(result, format)
			if err != nil {
				return err
			}

			_, err = fmt.Fprintln(cmd.OutOrStdout(), rendered)
			return err
		},
	}

	command.Flags().StringVar(&from, "from", "2h", "Relative lookback window such as 30m, 2h, or 24h")
	command.Flags().StringVar(&format, "format", "table", "Output format: table, json, markdown")
	command.Flags().IntVar(&limit, "limit", 200, "Maximum number of timeline events to collect")
	command.Flags().IntVar(&top, "top", 5, "Maximum number of ranked candidates to render")

	return command
}
