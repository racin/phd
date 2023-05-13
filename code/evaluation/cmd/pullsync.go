package cmd

import (
	"fmt"

	"github.com/racin/phd/code/evaluation/pullsync"
	"github.com/spf13/cobra"
)

var pullsyncCmd = &cobra.Command{
	Use:   "pullsync",
	Short: "start snips with given nonce",
	Long:  `Starts the SNIPS protocol with the given nonce.`,
}

var cursorsSubCmd = &cobra.Command{
	Use:   "cursors peer|all",
	Short: "clear cursors",
	Long:  `Clear the cursors for all peers, or a specific if given.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctl, err := getController(cmd.Context())
		if err != nil {
			return err
		}
		dur, err := pullsync.ClearPullSync(cmd.Context(), ctl, requestID, args[0])

		fmt.Printf("Finished in %s\n", dur)
		return
	},
}

var notifySubCmd = &cobra.Command{
	Use:   "notify",
	Short: "start snips with given nonce",
	Long:  `Starts the SNIPS protocol with the given nonce.`,
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctl, err := getController(cmd.Context())
		if err != nil {
			return err
		}
		dur, err := pullsync.StartPullSync(cmd.Context(), ctl, requestID)

		fmt.Printf("Finished in %s\n", dur)
		return
	},
}

func init() {
	rootCmd.AddCommand(pullsyncCmd)
	pullsyncCmd.AddCommand(cursorsSubCmd, notifySubCmd)
	resetRequestFlags(pullsyncCmd)
}
