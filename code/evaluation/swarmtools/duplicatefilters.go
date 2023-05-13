package main

import (
	"fmt"
	"strings"

	"github.com/prometheus/common/log"
	"github.com/spf13/cobra"
)

// chunkInPods: Key: PodId, Value: Array with ChunkHash

type chunkMeta struct {
	chunkInfo
	Addr     string
	RootAddr string
}

var duplicatefilterCmd = &cobra.Command{
	Use:   "duplicatefilters",
	Short: "Reports the chunks duplicated in passed filters. [--filters].",
	Long:  "Reports the chunks duplicated in passed filters. [--filters].",
	Run: func(cmd *cobra.Command, args []string) {
		if input_chunkFilter == "" {
			log.Fatalf("Must pass filters [--filters].")
			return
		}

		initChunkFilter(strings.Split(input_chunkFilter, ","))
		duplicates, knownChunks := findDuplicateChunks()
		fmt.Printf("Processed %v chunks.\n", len(knownChunks))
		if len(duplicates) == 0 {
			fmt.Printf("NO CHUNKS ARE DUPLICATED.\n")
			return
		}
		fmt.Printf("##### DUPLICATED CHUNKS: #####\n")
		fmt.Printf("-------------------------\n")
		for chunk, chunkmeta := range duplicates {
			fmt.Printf("Chunk: %v. Info:\n", chunk)
			for i := 0; i < len(chunkmeta); i++ {
				fmt.Printf("---   Root: %v, Index: %v, Depth: %v\n", chunkmeta[i].RootAddr, chunkmeta[i].Index, chunkmeta[i].Depth)
			}
		}
	},
}

func init() {
	duplicatefilterCmd.Flags().StringVarP(&input_chunkFilter, "filters", "", "", "JSON-encoded file(s) to filter the chunks listed [comma separated]")

	rootCmd.AddCommand(duplicatefilterCmd)
}

func findDuplicateChunks() (duplicates map[string][]*chunkMeta, knownChunks map[string]*chunkMeta) {
	duplicates = make(map[string][]*chunkMeta)
	knownChunks = make(map[string]*chunkMeta)
	for filename, chunks := range global_chunkFilters {
		for chunkAddr, chuInfo := range chunks {
			knownMeta, ok := knownChunks[chunkAddr]
			if ok {
				// We already know about this chunk therefore it is a duplicate
				if _, ok := duplicates[chunkAddr]; !ok {
					duplicates[chunkAddr] = make([]*chunkMeta, 2)
					duplicates[chunkAddr][0] = knownMeta
					duplicates[chunkAddr][1] = &chunkMeta{*chuInfo, chunkAddr, filename}
				} else {
					duplicates[chunkAddr] = append(duplicates[chunkAddr], &chunkMeta{*chuInfo, chunkAddr, filename})
				}
			} else {
				knownChunks[chunkAddr] = &chunkMeta{*chuInfo, chunkAddr, filename}
			}
		}
	}
	return
}
