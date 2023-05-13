package cmd

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/racin/phd/code/evaluation/snapshot"
	"github.com/spf13/cobra"
)

var (
	uploadPath         string
	chunkLossWhiteList []int
)

// snapshotsCmd represents the snapshot command
var snapshotsCmd = &cobra.Command{
	Use:   "snapshots",
	Short: "Create and apply snapshots",
	Long:  `The snapshot command is used to create and apply snapshots.`,
}

var snapshotsCreateCmd = &cobra.Command{
	Use:   "create path",
	Short: "create snapshots",
	Long:  `Create a snapshot containing the addresses of each bee's stored chunks.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctl, err := getController(cmd.Context())
		if err != nil {
			return err
		}

		return snapshot.Create(cmd.Context(), ctl, args[0])
	},
}

var snapshotsApplyCmd = &cobra.Command{
	Use:   "apply",
	Short: "apply snapshots to bees",
	Long:  "Apply makes it possible to upload and apply snapshots to bees.",
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctl, err := getController(cmd.Context())
		if err != nil {
			return err
		}

		if uploadPath != "" {
			return snapshot.UploadAndApply(cmd.Context(), ctl, uploadPath)
		} else {
			return snapshot.Apply(cmd.Context(), ctl)
		}
	},
}

var snapshotsClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "clear snapshots from bees",
	Long:  "Clear the bee's uploaded snapshot.",
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctl, err := getController(cmd.Context())
		if err != nil {
			return err
		}

		return snapshot.Clear(cmd.Context(), ctl)
	},
}

var snapshotsChunkLossCmd = &cobra.Command{
	Use:   "chunkloss percent chunkfile snapshots [output]",
	Short: "remove certain chunks from snapshots at random",
	Long: `The chunkloss command removes a random percentage of the chunks
specified in the chunkfile from each of the snapshots in the
snapshots folder. If the chunkfile is not a file on the disk,
chunkloss command will attempt to use it as a roothash for traversal instead.
If an output folder is given, the snapshots with loss are written there.
Otherwise, the snapshots are uploaded and applied.`,
	Args: cobra.RangeArgs(3, 4),
	RunE: func(cmd *cobra.Command, args []string) error {
		if chunkLossWhiteList == nil {
			chunkLossWhiteList = append(chunkLossWhiteList, uploaderID)
		}

		percent, err := strconv.ParseFloat(args[0], 64)
		if err != nil {
			return fmt.Errorf("failed to parse chunk loss percent: %w", err)
		}

		if len(args) == 4 {
			return snapshot.ChunkLoss(cmd.Context(), percent, chunkLossWhiteList, args[1], args[2], args[3], parallel)
		}

		ctl, err := getController(cmd.Context())
		if err != nil {
			return err
		}

		chunks, err := snapshot.GetChunkLossFromFile(cmd.Context(), args[1], percent)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return err
			}
			chunks, err = snapshot.GetChunkLossTraverse(cmd.Context(), ctl.Connector(), uploaderID, args[1], percent)
			if err != nil {
				return err
			}
		}

		return snapshot.UploadAndApplyWithChunkLoss(cmd.Context(), ctl, args[2], chunks, chunkLossWhiteList...)
	},
}

func init() {
	rootCmd.AddCommand(snapshotsCmd)
	snapshotsCmd.AddCommand(snapshotsCreateCmd, snapshotsApplyCmd, snapshotsClearCmd, snapshotsChunkLossCmd)

	snapshotsApplyCmd.Flags().StringVar(&uploadPath, "upload", "", "path to snapshots to upload")

	snapshotsChunkLossCmd.Flags().IntSliceVar(&chunkLossWhiteList, "white-list", nil, "white list for chunk loss (uploader is whitelisted by default)")
}
