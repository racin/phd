package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/racin/phd/code/evaluation/neighbor"
	"github.com/racin/phd/code/evaluation/util"
	"github.com/spf13/cobra"
)

var neighborCmd = &cobra.Command{
	Use:   "neighbor [verify | print]",
	Short: "gathers neighborhood information",
	Long:  `Queries each peer for their neighborhood information and creates a relation graph.`,
	Args:  cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctl, err := getController(cmd.Context())
		if err != nil {
			return err
		}
		neighborMap, reverseNeighborMap, err := neighbor.GetNeighborMap(cmd.Context(), ctl, requestID)

		neighbor.VerifyNeighborMaps(neighborMap, reverseNeighborMap)
		return
	},
}

var verifySubCmd = &cobra.Command{
	Use:   "verify",
	Short: "verifies neighborhood information",
	Long:  `Verifies that all edges in the connection graph are bidirectional.`,
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctl, err := getController(cmd.Context())
		if err != nil {
			return err
		}
		neighborMap, reverseNeighborMap, err := neighbor.GetNeighborMap(cmd.Context(), ctl, requestID)

		neighbor.VerifyNeighborMaps(neighborMap, reverseNeighborMap)
		return
	},
}

var printSubCmd = &cobra.Command{
	Use:   "print summary|raw|count|revsummary|revraw|revcount",
	Short: "print neighborhood information",
	Long:  `Prints neighborhood information in a human-readable format.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		if args[0] != "summary" && args[0] != "raw" && args[0] != "count" && args[0] != "revsummary" && args[0] != "revraw" && args[0] != "revcount" {
			fmt.Println("Invalid first parameter. Valid options: [ summary | raw | count | revsummary | revraw | revcount ]")
			return nil
		}
		ctl, err := getController(cmd.Context())
		if err != nil {
			return err
		}

		var output io.Writer
		if requestOutput != "" {
			f, err := os.OpenFile(requestOutput, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, util.FilePerm)
			if err != nil {
				return fmt.Errorf("failed to open output file: %w", err)
			}
			defer util.SafeClose(&err, f)
			output = f
		} else {
			output = os.Stdout
		}

		neighborMap, revNeighborMap, err := neighbor.GetNeighborMap(cmd.Context(), ctl, requestID)
		switch args[0] {
		case "raw":
			util.PrintJson(output, neighborMap)
		case "summary":
			cdStats := util.CalcStats(neighborMap)
			fmt.Fprintf(output, "Chunk stats. total: %d, unique: %d, min: %d, max: %d, median: %d mean: %f\n",
				cdStats.Total, cdStats.Unique, cdStats.Min, cdStats.Max, cdStats.Median, cdStats.Mean)
		case "count":
			util.PrintJson(output, util.CountMap(neighborMap))
		case "revraw":
			util.PrintJson(output, revNeighborMap)
		case "revsummary":
			cdStats := util.CalcStats(revNeighborMap)
			fmt.Fprintf(output, "Chunk stats. total: %d, unique: %d, min: %d, max: %d, median: %d mean: %f\n",
				cdStats.Total, cdStats.Unique, cdStats.Min, cdStats.Max, cdStats.Median, cdStats.Mean)
		case "revcount":
			util.PrintJson(output, util.CountMap(revNeighborMap))
		}
		return
	},
}

func init() {
	rootCmd.AddCommand(neighborCmd)
	resetRequestFlags(neighborCmd)

	neighborCmd.AddCommand(verifySubCmd)
	neighborCmd.AddCommand(printSubCmd)
	printSubCmd.Flags().StringVarP(&requestOutput, "output", "o", "", "File to write response body to (defaults to stdout)")

}
