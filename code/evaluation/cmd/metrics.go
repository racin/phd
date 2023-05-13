package cmd

import (
	"fmt"

	"github.com/racin/phd/code/evaluation/metrics"
	"github.com/racin/phd/code/evaluation/util"
	"github.com/spf13/cobra"
)

var (
	metricsFile string
)

var metricsCmd = &cobra.Command{
	Use:   "metrics [metric...]",
	Short: "Read metrics",
	Long: `Command metrics reads metrics from the bees.
By default, this command prints all metrics.
Optionally, the name of specific metrics may be specified.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctl, err := getController(cmd.Context())
		if err != nil {
			return err
		}

		var metricsS metrics.MetricsSlice

		if metricsFile == "" {
			metricsS, err = metrics.GetMetrics(cmd.Context(), ctl)
		} else {
			err = util.ReadJSONFile(metricsFile, &metricsS)
		}
		if err != nil {
			return err
		}

		metrics := metrics.Sum(metricsS)

		if len(args) == 0 {
			for k, v := range metrics {
				fmt.Println(k, v)
			}
		} else {
			for _, k := range args {
				if v, ok := metrics[k]; ok {
					fmt.Println(k, v)
				} else {
					fmt.Println(k, "not found")
				}
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(metricsCmd)

	metricsCmd.Flags().StringVarP(&metricsFile, "file", "f", "", "file to read metrics from")
}
