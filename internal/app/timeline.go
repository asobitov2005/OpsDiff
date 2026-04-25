package app

import (
	"fmt"
	"time"

	"github.com/asobitov2005/OpsDiff/internal/kube"
	"github.com/asobitov2005/OpsDiff/internal/report"
	"github.com/spf13/cobra"
)

func newTimelineCommand(options *RootOptions) *cobra.Command {
	var format string
	var from string
	var limit int

	command := &cobra.Command{
		Use:   "timeline",
		Short: "Show a filtered incident timeline from Kubernetes events and pod signals",
		RunE: func(cmd *cobra.Command, _ []string) error {
			duration, err := time.ParseDuration(from)
			if err != nil {
				return fmt.Errorf("parse --from duration: %w", err)
			}
			if duration <= 0 {
				return fmt.Errorf("--from must be greater than zero")
			}

			collector := kube.NewCollector(options.Kubeconfig)
			timeline, err := collector.CollectTimeline(cmd.Context(), options.Namespace, duration, limit)
			if err != nil {
				return err
			}

			rendered, err := report.RenderTimeline(timeline, format)
			if err != nil {
				return err
			}

			_, err = fmt.Fprintln(cmd.OutOrStdout(), rendered)
			return err
		},
	}

	command.Flags().StringVar(&from, "from", "2h", "Relative lookback window such as 30m, 2h, or 24h")
	command.Flags().StringVar(&format, "format", "table", "Output format: table, json, markdown")
	command.Flags().IntVar(&limit, "limit", 200, "Maximum number of timeline events to render")

	return command
}
