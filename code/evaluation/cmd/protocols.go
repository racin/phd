package cmd

import (
	"fmt"

	"github.com/racin/phd/code/evaluation/protocols"
	"github.com/spf13/cobra"
)

var protocolsCmd = &cobra.Command{
	Use:   "protocols enable|disable|status protocols...",
	Short: "manage protocols",
	Long:  `The protocols command is used to enable and disable protocols.`,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctl, err := getController(cmd.Context())
		if err != nil {
			return err
		}

		switch args[0] {
		case "enable":
			return protocols.SetProtocols(cmd.Context(), ctl, args[1:], nil)
		case "disable":
			return protocols.SetProtocols(cmd.Context(), ctl, nil, args[1:])
		case "status":
			for _, protocol := range args[1:] {
				statuses, err := protocols.GetProtocolState(cmd.Context(), ctl, protocol)
				if err != nil {
					return err
				}
				fmt.Printf("%s: \n", protocol)
				printProtocolStatuses(statuses)
				fmt.Println()
			}
		default:
			return cmd.Usage()
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(protocolsCmd)
}

func printProtocolStatuses(statuses []protocols.State) {
	statusesPerLine := 50
	for i, status := range statuses {
		if i%statusesPerLine == 0 {
			fmt.Println()
		}
		switch status {
		case protocols.Enabled:
			fmt.Print("✅")
		case protocols.Disabled:
			fmt.Print("⛔")
		case protocols.Unknown:
			fallthrough
		default:
			fmt.Print("❓")
		}
	}
	fmt.Println()
}
