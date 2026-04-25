package app

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/asobitov2005/OpsDiff/internal/report"
	"github.com/spf13/cobra"
)

func newReportCommand(options *RootOptions) *cobra.Command {
	var out string
	var from string
	var limit int
	var top int
	var prometheusFile string
	var argoCDFile string

	command := &cobra.Command{
		Use:   "report <before> <after>",
		Short: "Generate an HTML incident report from snapshots and correlated signals",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if out == "" {
				return fmt.Errorf("--out is required")
			}

			result, timeline, err := buildExplainContext(cmd.Context(), options, args, IncidentOptions{
				From:           from,
				Limit:          limit,
				Top:            top,
				PrometheusFile: prometheusFile,
				ArgoCDFile:     argoCDFile,
			})
			if err != nil {
				return err
			}

			html, err := report.RenderIncidentHTML(result, timeline)
			if err != nil {
				return err
			}

			if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
				return fmt.Errorf("create report directory for %q: %w", out, err)
			}

			if err := os.WriteFile(out, []byte(html), 0o644); err != nil {
				return fmt.Errorf("write report %q: %w", out, err)
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "HTML report written to %s\n", out)
			return err
		},
	}

	command.Flags().StringVar(&out, "out", "", "Output path for the HTML report")
	command.Flags().StringVar(&from, "from", "2h", "Relative lookback window such as 30m, 2h, or 24h")
	command.Flags().IntVar(&limit, "limit", 200, "Maximum number of timeline events to collect")
	command.Flags().IntVar(&top, "top", 5, "Maximum number of ranked candidates to render")
	command.Flags().StringVar(&prometheusFile, "prometheus-file", "", "Path to a Prometheus alert export JSON file")
	command.Flags().StringVar(&argoCDFile, "argocd-file", "", "Path to an ArgoCD sync event export JSON file")

	return command
}
