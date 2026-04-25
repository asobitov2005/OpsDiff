package app

import (
	"fmt"

	"github.com/asobitov2005/OpsDiff/internal/kube"
	"github.com/asobitov2005/OpsDiff/internal/report"
	"github.com/spf13/cobra"
)

func newDoctorCommand(options *RootOptions) *cobra.Command {
	var format string

	command := &cobra.Command{
		Use:   "doctor",
		Short: "Validate kubeconfig, cluster connectivity, and read permissions",
		RunE: func(cmd *cobra.Command, _ []string) error {
			checks := kube.RunDoctor(cmd.Context(), options.Kubeconfig, options.Namespace)
			rendered, err := report.RenderDoctor(checks, format)
			if err != nil {
				return err
			}

			if _, err := fmt.Fprintln(cmd.OutOrStdout(), rendered); err != nil {
				return err
			}

			if kube.HasFailures(checks) {
				return fmt.Errorf("doctor checks failed")
			}

			return nil
		},
	}

	command.Flags().StringVar(&format, "format", "table", "Output format: table or json")
	return command
}
