package cmd

import (
	"fmt"
	"log"
	"time"

	"github.com/racin/phd/code/evaluation/util"
	"github.com/racin/phd/code/evaluation/verifypersistence"
	"github.com/spf13/cobra"
)

var requestHost string
var verifyInterval time.Duration
var requestInterval time.Duration

var verifyPersistenceCmd = &cobra.Command{
	Use:   "verifypersistence downloadchunks|verify rootaddr",
	Short: "verify the persistence of all chunks",
	Long: `Verify persistence of all chunks of a file in Swarm.
	Use option downloadchunks to download each of the chunks.
	Use option verify to periodically check if the chunks is available.`,

	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		if args[0] != "downloadchunks" && args[0] != "verify" {
			return fmt.Errorf("invalid first parameter. valid options: [ downloadchunks | verify ]")
		}
		conn, err := getConnector()
		if err != nil {
			return err
		}

		if args[0] == "downloadchunks" {
			count, err := verifypersistence.DownloadAllChunksOfFile(cmd.Context(), conn, args[1], requestOutput, requestID)
			if err != nil {
				return err
			}

			log.Printf("Finished downloading %v chunks.\n", count)
			return nil
		} else if args[0] == "verify" {
			var host string
			if requestHost == "" {
				// Connect API
				thehost, disconnect, err := conn.ConnectAPI(cmd.Context(), requestID)
				if err != nil {
					return fmt.Errorf("failed to connect to API bee %d: %s", requestID, err)
				}
				host = thehost
				defer util.CheckErr(&err, disconnect)
			} else {
				host = requestHost
			}

			verifypersistence.VerifyPersistence(cmd.Context(), requestOutput, host, verifyInterval, requestInterval)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(verifyPersistenceCmd)
	verifyPersistenceCmd.Flags().StringVarP(&requestHost, "host", "", "", "Host to send requests to.")
	verifyPersistenceCmd.Flags().StringVarP(&requestOutput, "output", "o", "", "File to write response body to (defaults to stdout)")
	verifyPersistenceCmd.Flags().DurationVar(&verifyInterval, "verifyinterval", 12*time.Hour, "Interval to verify persistence")
	verifyPersistenceCmd.Flags().DurationVar(&requestInterval, "requestinterval", 4*time.Second, "Interval to request chunk")
}
