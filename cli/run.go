package cli

import (
	"github.com/NethermindEth/eigenlayer/pkg/daemon"
	"github.com/spf13/cobra"
)

func RunCmd(d daemon.Daemon) *cobra.Command {
	var instanceId string
	cmd := cobra.Command{
		Use:   "run [INSTANCE_ID]",
		Short: "Start an AVS node instance",
		Long:  "Start an AVS node instance. The instance ID is required as the unique argument. INSTANCE_ID is required as the unique argument, and it is the combination of the instance repository name and the instance tag computed during the installation, like this: [repository-name]-[tag].",
		Args:  cobra.ExactArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			instanceId = args[0]
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return d.Run(instanceId)
		},
	}
	return &cmd
}