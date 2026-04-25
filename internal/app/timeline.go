package app

import (
	"fmt"

	"github.com/asobitov2005/OpsDiff/internal/report"
	"github.com/spf13/cobra"
)

func newTimelineCommand(options *RootOptions) *cobra.Command {
	var format string
	var from string
	var limit int
	var prometheusFile string
	var argoCDFile string

	command := &cobra.Command{
		Use:   "timeline",
		Short: "Show a filtered incident timeline from Kubernetes events and pod signals",
		RunE: func(cmd *cobra.Command, _ []string) error {
			duration, err := parseLookback(from)
			if err != nil {
				return err
			}

			timeline, err := collectTimelineWithImports(cmd.Context(), options, options.Namespace, duration, limit, prometheusFile, argoCDFile)
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
	command.Flags().StringVar(&prometheusFile, "prometheus-file", "", "Path to a Prometheus alert export JSON file")
	command.Flags().StringVar(&argoCDFile, "argocd-file", "", "Path to an ArgoCD sync event export JSON file")

	return command
}
