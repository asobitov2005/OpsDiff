package app

import (
	"fmt"

	"github.com/asobitov2005/OpsDiff/internal/kube"
	"github.com/asobitov2005/OpsDiff/internal/store"
	"github.com/spf13/cobra"
)

func newSnapshotCommand(options *RootOptions) *cobra.Command {
	var out string

	command := &cobra.Command{
		Use:   "snapshot",
		Short: "Capture a normalized Kubernetes snapshot",
		RunE: func(cmd *cobra.Command, _ []string) error {
			collector := kube.NewCollector(options.Kubeconfig)
			snapshot, err := collector.CollectSnapshot(cmd.Context(), options.Namespace)
			if err != nil {
				return err
			}

			if err := store.WriteSnapshot(out, snapshot); err != nil {
				return err
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Snapshot written to %s (%d resources)\n", out, len(snapshot.Resources))
			return err
		},
	}

	command.Flags().StringVarP(&out, "out", "o", "", "Output path for the snapshot JSON file")
	_ = command.MarkFlagRequired("out")

	return command
}
