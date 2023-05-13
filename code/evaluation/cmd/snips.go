package cmd

import (
	"fmt"
	"strconv"

	"github.com/racin/phd/code/evaluation/snips"
	"github.com/spf13/cobra"
)

var snipsCmd = &cobra.Command{
	Use:   "snips nonce",
	Short: "start snips with given nonce",
	Long:  `Starts the SNIPS protocol with the given nonce.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctl, err := getController(cmd.Context())
		if err != nil {
			return err
		}
		nonce, err := strconv.ParseUint(args[0], 10, 64)
		if err != nil {
			return err
		}
		dur, err := snips.StartSNIPS(cmd.Context(), ctl, requestID, nonce)

		fmt.Printf("Finished in %s\n", dur)
		return
	},
}

func init() {
	rootCmd.AddCommand(snipsCmd)
	resetRequestFlags(snipsCmd)
}
