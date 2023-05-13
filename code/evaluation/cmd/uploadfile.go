package cmd

import (
	"fmt"
	"log"

	"github.com/racin/phd/code/evaluation/dataset"
	"github.com/racin/phd/code/evaluation/util"
	"github.com/spf13/cobra"
)

var uploadFileCmd = &cobra.Command{
	Use:   "uploadfile path",
	Short: "upload a file",
	Long:  `Upload a file to Bee.`,

	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		conn, err := getConnector()
		if err != nil {
			return err
		}

		uploader, disconnect, err := conn.ConnectAPI(cmd.Context(), uploaderID)
		if err != nil {
			return fmt.Errorf("failed to connect to bee %d: %w", uploaderID, err)
		}
		defer util.CheckErr(&err, disconnect)

		if stampID == "" {
			fmt.Print("Provide stamp address for uploading: ")
			_, err = fmt.Scanln(&stampID)
			if err != nil {
				return fmt.Errorf("failed to read stamp address: %w", err)
			}
		}

		rootAddr, err := dataset.UploadFile(uploader, args[0], stampID)
		if err != nil {
			return err
		}

		log.Printf("Root address: %v\n", rootAddr)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(uploadFileCmd)

	uploadFileCmd.Flags().StringVarP(&stampID, "stamp", "s", "", "The address of the stamp to use for uploads.")
}
