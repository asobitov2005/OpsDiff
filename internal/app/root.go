package app

import "github.com/spf13/cobra"

var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

type RootOptions struct {
	Kubeconfig string
	Namespace  string
}

func NewRootCommand() *cobra.Command {
	options := &RootOptions{}

	command := &cobra.Command{
		Use:           "opsdiff",
		Short:         "Find what changed before your Kubernetes incident.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	command.PersistentFlags().StringVar(&options.Kubeconfig, "kubeconfig", "", "Path to kubeconfig file")
	command.PersistentFlags().StringVarP(&options.Namespace, "namespace", "n", "default", "Namespace to inspect")

	command.AddCommand(
		newSnapshotCommand(options),
		newCompareCommand(),
		newDoctorCommand(options),
		newVersionCommand(),
	)

	return command
}
