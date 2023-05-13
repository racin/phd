package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/racin/phd/code/evaluation/dataset"
	"github.com/racin/phd/code/evaluation/util"
	"github.com/spf13/cobra"
)

var stampID string

var uploadCmd = &cobra.Command{
	Use:   "upload dataset.json [addresses.json]",
	Short: "upload a dataset",
	Long: `The upload command uploads a set of files to a bee.

The dataset must be described in a JSON file with the following schema:

	{
		"sizes": [
			{
				"size_kb": 10,
				"count": 5
				"prefix": "abc" // optional parameter
			}
		]
	}

The command generates files containing random data with the specified sizes
and uploads them to the uploader bee. The addresses of the uploaded files
are written to stdout or to the specified addresses file, in JSON format.`,

	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		conn, err := getConnector()
		if err != nil {
			return err
		}

		var config dataset.DatasetConfig
		err = util.ReadJSONFile(args[0], &config)
		if err != nil {
			return fmt.Errorf("failed to read dataset config: %w", err)
		}

		if config.StampID == "" {
			if stampID == "" {
				fmt.Print("Provide stamp address for uploading: ")
				_, err = fmt.Scanln(&stampID)
				if err != nil {
					return fmt.Errorf("failed to read stamp address: %w", err)
				}
			}
			config.StampID = stampID
		}

		config.UploaderID = uploaderID

		dataSet, err := dataset.UploadDataset(cmd.Context(), conn, config)
		if err != nil {
			return err
		}

		if len(args) == 2 {
			err = util.WriteJSONFile(args[1], util.FilePerm, dataSet)
			if err != nil {
				return err
			}
		} else {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "\t")
			err = enc.Encode(dataSet)
			if err != nil {
				return fmt.Errorf("failed to write dataset as JSON: %w", err)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(uploadCmd)

	uploadCmd.Flags().StringVarP(&stampID, "stamp", "s", "", "The address of the stamp to use for uploads.")
}
